package migration

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/skssmd/norm/core/driver"
	"github.com/skssmd/norm/core/registry"
	"github.com/skssmd/norm/core/utils"
)

// AutoMigrator handles automatic schema migration from structs
type AutoMigrator struct {
	models []interface{}
}

// NewAutoMigrator creates a new auto migrator
func NewAutoMigrator() *AutoMigrator {
	return &AutoMigrator{
		models: make([]interface{}, 0),
	}
}

// AddModel registers a model struct for auto migration
func (am *AutoMigrator) AddModel(model interface{}) {
	am.models = append(am.models, model)
}

// DropAllTables drops all tables for registered models (useful for development)
func (am *AutoMigrator) DropAllTables() error {
	if len(am.models) == 0 {
		return nil
	}

	regInfo := registry.GetRegistryInfo()
	mode := regInfo["mode"].(string)

	var wg sync.WaitGroup
	errChan := make(chan error, 100)

	if mode == "global" || mode == "" {
		// Drop from global pools
		pools := regInfo["pools"].(map[string]interface{})
		for poolName, poolInfo := range pools {
			pool := poolInfo.(*driver.PGPool)
			wg.Add(1)
			go func(name string, p *driver.PGPool) {
				defer wg.Done()
				if err := am.dropTablesFromPool(p, fmt.Sprintf("Global:%s", name)); err != nil {
					errChan <- fmt.Errorf("global pool '%s': %w", name, err)
				}
			}(poolName, pool)
		}
	}

	if mode == "shard" {
		// Drop from shard primary pools only
		shards := regInfo["shards"].(map[string]interface{})
		for shardName, shardInfo := range shards {
			shardData := shardInfo.(map[string]interface{})

			if hasPrimary, ok := shardData["has_primary"].(bool); ok && hasPrimary {
				if primaryPool, ok := shardData["primary_pool"].(*driver.PGPool); ok {
					wg.Add(1)
					go func(shard string, p *driver.PGPool) {
						defer wg.Done()
						if err := am.dropTablesFromPool(p, fmt.Sprintf("Shard:%s:primary", shard)); err != nil {
							errChan <- fmt.Errorf("shard '%s' primary: %w", shard, err)
						}
					}(shardName, primaryPool)
				}
			}
		}
	}

	// Wait for all drops to complete
	wg.Wait()
	close(errChan)

	// Collect errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("drop tables failed with %d errors: %v", len(errors), errors)
	}

	return nil
}

// dropTablesFromPool drops all tables from a single pool
func (am *AutoMigrator) dropTablesFromPool(pool *driver.PGPool, poolLabel string) error {
	ctx := context.Background()

	fmt.Printf("🗑️  Dropping tables from %s...\n", poolLabel)

	for _, model := range am.models {
		tableName := getTableNameFromStruct(model)

		// Drop table with CASCADE to handle foreign keys
		dropSQL := fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE;", tableName)
		_, err := pool.Pool.Exec(ctx, dropSQL)
		if err != nil {
			return fmt.Errorf("failed to drop table '%s': %w", tableName, err)
		}

		fmt.Printf("  ✓ Dropped table '%s' from %s\n", tableName, poolLabel)
	}

	fmt.Printf("✅ All tables dropped from %s\n", poolLabel)
	return nil
}

// AutoMigrate runs automatic migration for all registered models
func (am *AutoMigrator) AutoMigrate() error {
	if len(am.models) == 0 {
		return nil
	}

	regInfo := registry.GetRegistryInfo()
	mode := regInfo["mode"].(string)

	var wg sync.WaitGroup
	errChan := make(chan error, 100)

	if mode == "global" || mode == "" {
		// Migrate global pools
		pools := regInfo["pools"].(map[string]interface{})
		for poolName, poolInfo := range pools {
			pool := poolInfo.(*driver.PGPool)
			wg.Add(1)
			go func(name string, p *driver.PGPool) {
				defer wg.Done()
				if err := am.migratePool(p, fmt.Sprintf("Global:%s", name)); err != nil {
					errChan <- fmt.Errorf("global pool '%s': %w", name, err)
				}
			}(poolName, pool)
		}
	}

	if mode == "shard" {
		// Migrate shard pools (both primary and standalone)
		shards := regInfo["shards"].(map[string]interface{})
		for shardName, shardInfo := range shards {
			shardData := shardInfo.(map[string]interface{})

			// Migrate primary pool if exists
			if hasPrimary, ok := shardData["has_primary"].(bool); ok && hasPrimary {
				if primaryPool, ok := shardData["primary_pool"].(*driver.PGPool); ok {
					wg.Add(1)
					go func(shard string, p *driver.PGPool) {
						defer wg.Done()
						if err := am.migratePoolForShard(p, shard, fmt.Sprintf("Shard:%s:primary", shard)); err != nil {
							errChan <- fmt.Errorf("shard '%s' primary: %w", shard, err)
						}
					}(shardName, primaryPool)
				}
			}

			// Migrate standalone pools if they exist
			if standalonePools, ok := shardData["standalone_pools"].(map[string]*driver.PGPool); ok && len(standalonePools) > 0 {
				for poolKey, standalonePool := range standalonePools {
					wg.Add(1)
					go func(shard string, poolK string, p *driver.PGPool) {
						defer wg.Done()
						if err := am.migratePoolForShard(p, shard, fmt.Sprintf("Shard:%s:%s", shard, poolK)); err != nil {
							errChan <- fmt.Errorf("shard '%s' %s: %w", shard, poolK, err)
						}
					}(shardName, poolKey, standalonePool)
				}
			}
		}
	}

	// Wait for all migrations to complete
	wg.Wait()
	close(errChan)

	// Collect errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("auto migration failed with %d errors: %v", len(errors), errors)
	}

	return nil
}

// migratePool runs auto migration on a single pool (for global mode)
func (am *AutoMigrator) migratePool(pool *driver.PGPool, poolLabel string) error {
	ctx := context.Background()

	fmt.Printf("🔄 Auto-migrating %s...\n", poolLabel)

	// Sort models by dependency (tables without foreign keys first)
	sortedModels := am.sortModelsByDependency()

	for _, model := range sortedModels {
		tableName := getTableNameFromStruct(model)

		// Check if table exists
		exists, err := am.tableExists(ctx, pool, tableName)
		if err != nil {
			return fmt.Errorf("failed to check table '%s': %w", tableName, err)
		}

		if !exists {
			// Create new table
			if err := am.createTable(ctx, pool, model, tableName); err != nil {
				return fmt.Errorf("failed to create table '%s': %w", tableName, err)
			}
			fmt.Printf("  ✓ Created table '%s' in %s\n", tableName, poolLabel)
		} else {
			// Update existing table
			if err := am.updateTable(ctx, pool, model, tableName); err != nil {
				return fmt.Errorf("failed to update table '%s': %w", tableName, err)
			}
			fmt.Printf("  ✓ Updated table '%s' in %s\n", tableName, poolLabel)
		}
	}

	fmt.Printf("✅ Auto-migration completed for %s\n", poolLabel)
	return nil
}

// migratePoolForShard runs auto migration on a shard pool, only for tables registered to that shard
func (am *AutoMigrator) migratePoolForShard(pool *driver.PGPool, shardName string, poolLabel string) error {
	ctx := context.Background()

	fmt.Printf("🔄 Auto-migrating %s...\n", poolLabel)

	// Filter models that belong to this shard
	shardModels := am.getModelsForShard(shardName)

	if len(shardModels) == 0 {
		fmt.Printf("  ⊘ No tables registered for %s\n", poolLabel)
		return nil
	}

	// Sort models by dependency (tables without foreign keys first)
	sortedModels := am.sortModelsByDependencyFromList(shardModels)

	for _, model := range sortedModels {
		tableName := getTableNameFromStruct(model)

		// Check if table exists
		exists, err := am.tableExists(ctx, pool, tableName)
		if err != nil {
			return fmt.Errorf("failed to check table '%s': %w", tableName, err)
		}

		if !exists {
			// Create new table
			if err := am.createTable(ctx, pool, model, tableName); err != nil {
				return fmt.Errorf("failed to create table '%s': %w", tableName, err)
			}
			fmt.Printf("  ✓ Created table '%s' in %s\n", tableName, poolLabel)
		} else {
			// Update existing table
			if err := am.updateTable(ctx, pool, model, tableName); err != nil {
				return fmt.Errorf("failed to update table '%s': %w", tableName, err)
			}
			fmt.Printf("  ✓ Updated table '%s' in %s\n", tableName, poolLabel)
		}
	}

	fmt.Printf("✅ Auto-migration completed for %s\n", poolLabel)
	return nil
}

// getModelsForShard returns only models that are registered to the specified shard
func (am *AutoMigrator) getModelsForShard(shardName string) []interface{} {
	var shardModels []interface{}

	for _, model := range am.models {
		// Get the registered table name for this model
		tableName := getTableNameFromStruct(model)

		// Try to get table mapping using the table name
		mapping, err := registry.GetTableMapping(tableName)
		if err != nil {
			continue // Skip if not found
		}

		// Include if table is mapped to this shard (primary or standalone)
		if mapping.ShardName() == shardName {
			shardModels = append(shardModels, model)
		}
	}

	return shardModels
}

// sortModelsByDependency sorts models so tables without foreign keys are created first
func (am *AutoMigrator) sortModelsByDependency() []interface{} {
	return am.sortModelsByDependencyFromList(am.models)
}

// sortModelsByDependencyFromList sorts a given list of models by dependency
func (am *AutoMigrator) sortModelsByDependencyFromList(models []interface{}) []interface{} {
	// Separate models into those with and without foreign keys
	var noDeps []interface{}
	var withDeps []interface{}

	for _, model := range models {
		hasForeignKey := false

		t := reflect.TypeOf(model)
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}

		// Check if model has any foreign key tags
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			normTag := field.Tag.Get("norm")
			if strings.Contains(normTag, "fkey:") {
				hasForeignKey = true
				break
			}
		}

		if hasForeignKey {
			withDeps = append(withDeps, model)
		} else {
			noDeps = append(noDeps, model)
		}
	}

	// Return models without dependencies first, then those with dependencies
	return append(noDeps, withDeps...)
}

// tableExists checks if a table exists in the database
func (am *AutoMigrator) tableExists(ctx context.Context, pool *driver.PGPool, tableName string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = $1
		);
	`

	var exists bool
	err := pool.Pool.QueryRow(ctx, query, tableName).Scan(&exists)
	return exists, err
}

// createTable creates a new table from struct
func (am *AutoMigrator) createTable(ctx context.Context, pool *driver.PGPool, model interface{}, tableName string) error {
	createSQL, indexSQLs := am.generateCreateTableSQL(model, tableName)

	// Execute CREATE TABLE statement
	_, err := pool.Pool.Exec(ctx, createSQL)
	if err != nil {
		return err
	}

	// Execute CREATE INDEX statements separately
	for _, indexSQL := range indexSQLs {
		_, err := pool.Pool.Exec(ctx, indexSQL)
		if err != nil && !strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

// updateTable updates an existing table schema
func (am *AutoMigrator) updateTable(ctx context.Context, pool *driver.PGPool, model interface{}, tableName string) error {
	// Get existing columns
	existingCols, err := am.getExistingColumns(ctx, pool, tableName)
	if err != nil {
		return err
	}

	// Get desired columns from struct
	desiredCols := am.parseStructColumns(model)

	// Find columns to add
	for colName, colDef := range desiredCols {
		if _, exists := existingCols[colName]; !exists {
			// Add new column
			alterSQL := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s;", tableName, colName, colDef)
			if _, err := pool.Pool.Exec(ctx, alterSQL); err != nil {
				return fmt.Errorf("failed to add column '%s': %w", colName, err)
			}
			fmt.Printf("    + Added column '%s' to '%s'\n", colName, tableName)
		}
	}

	// Add indexes and foreign keys for existing tables
	if err := am.addIndexesAndConstraints(ctx, pool, model, tableName); err != nil {
		return err
	}

	// Note: We don't drop columns automatically for safety
	// You can add logic here to detect and handle column type changes

	return nil
}

// addIndexesAndConstraints adds missing indexes and foreign keys to existing tables
func (am *AutoMigrator) addIndexesAndConstraints(ctx context.Context, pool *driver.PGPool, model interface{}, tableName string) error {
	t := reflect.TypeOf(model)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		colName := utils.ToSnakeCase(field.Name)
		normTag := field.Tag.Get("norm")
		tags := parseNormTags(normTag)

		// Add index if specified
		if _, ok := tags["index"]; ok {
			indexName := fmt.Sprintf("idx_%s_%s", tableName, colName)
			indexSQL := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s(%s);", indexName, tableName, colName)
			if _, err := pool.Pool.Exec(ctx, indexSQL); err != nil && !strings.Contains(err.Error(), "already exists") {
				fmt.Printf("    ⚠️  Failed to create index '%s': %v\n", indexName, err)
			}
		}

		// Add index for soft key (application-level relationship)
		if _, ok := tags["skey"]; ok {
			indexName := fmt.Sprintf("idx_%s_%s", tableName, colName)
			indexSQL := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s(%s);", indexName, tableName, colName)
			if _, err := pool.Pool.Exec(ctx, indexSQL); err != nil && !strings.Contains(err.Error(), "already exists") {
				fmt.Printf("    ⚠️  Failed to create index '%s': %v\n", indexName, err)
			}
		}

		// Add foreign key if specified (database-level constraint)
		if fkey, ok := tags["fkey"]; ok {
			fkParts := strings.Split(fkey.(string), ".")
			if len(fkParts) == 2 {
				onDelete := "NO ACTION"
				if od, ok := tags["ondelete"]; ok {
					onDelete = strings.ToUpper(od.(string))
				}

				onUpdate := "NO ACTION"
				if ou, ok := tags["onupdate"]; ok {
					onUpdate = strings.ToUpper(ou.(string))
				}

				fkName := fmt.Sprintf("fk_%s_%s", tableName, colName)
				fkSQL := fmt.Sprintf(
					"ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s(%s) ON DELETE %s ON UPDATE %s;",
					tableName, fkName, colName, fkParts[0], fkParts[1], onDelete, onUpdate)

				if _, err := pool.Pool.Exec(ctx, fkSQL); err != nil {
					if !strings.Contains(err.Error(), "already exists") {
						fmt.Printf("    ⚠️  Failed to create foreign key '%s': %v\n", fkName, err)
					}
				}
			}
		}
	}

	return nil
}

// getExistingColumns retrieves existing columns from a table
func (am *AutoMigrator) getExistingColumns(ctx context.Context, pool *driver.PGPool, tableName string) (map[string]string, error) {
	query := `
		SELECT column_name, data_type 
		FROM information_schema.columns 
		WHERE table_schema = 'public' 
		AND table_name = $1;
	`

	rows, err := pool.Pool.Query(ctx, query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns := make(map[string]string)
	for rows.Next() {
		var colName, dataType string
		if err := rows.Scan(&colName, &dataType); err != nil {
			return nil, err
		}
		columns[colName] = dataType
	}

	return columns, rows.Err()
}

// parseStructColumns parses struct fields into column definitions
func (am *AutoMigrator) parseStructColumns(model interface{}) map[string]string {
	columns := make(map[string]string)

	t := reflect.TypeOf(model)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		colName := utils.ToSnakeCase(field.Name)
		colType := am.getPostgresType(field)

		columns[colName] = colType
	}

	return columns
}

// generateCreateTableSQL generates CREATE TABLE SQL from struct
// Returns the CREATE TABLE statement and a slice of CREATE INDEX statements
func (am *AutoMigrator) generateCreateTableSQL(model interface{}, tableName string) (string, []string) {
	var sql strings.Builder

	sql.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n", tableName))

	t := reflect.TypeOf(model)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	columnDefs := make([]string, 0)
	var primaryKeys []string
	var foreignKeys []string
	var indexes []string

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		colName := utils.ToSnakeCase(field.Name)
		colType := am.getPostgresType(field)

		// Parse norm tags
		normTag := field.Tag.Get("norm")
		tags := parseNormTags(normTag)

		colDef := fmt.Sprintf("  %s %s", colName, colType)

		// Handle primary key - make it auto-incrementing
		if _, ok := tags["pk"]; ok {
			// For integer types, use SERIAL/BIGSERIAL for auto-increment
			switch colType {
			case "INTEGER":
				colDef = fmt.Sprintf("  %s SERIAL", colName)
			case "BIGINT":
				colDef = fmt.Sprintf("  %s BIGSERIAL", colName)
			}
			primaryKeys = append(primaryKeys, colName)
		}

		// Handle constraints
		if _, ok := tags["notnull"]; ok {
			colDef += " NOT NULL"
		}

		if _, ok := tags["unique"]; ok {
			colDef += " UNIQUE"
		}

		if defaultVal, ok := tags["default"]; ok {
			colDef += fmt.Sprintf(" DEFAULT %s", defaultVal.(string))
		}

		columnDefs = append(columnDefs, colDef)

		// Handle foreign keys (database-level constraint)
		if fkey, ok := tags["fkey"]; ok {
			fkParts := strings.Split(fkey.(string), ".")
			if len(fkParts) == 2 {
				onDelete := "NO ACTION"
				if od, ok := tags["ondelete"]; ok {
					onDelete = strings.ToUpper(od.(string))
				}

				onUpdate := "NO ACTION"
				if ou, ok := tags["onupdate"]; ok {
					onUpdate = strings.ToUpper(ou.(string))
				}

				foreignKeys = append(foreignKeys,
					fmt.Sprintf("  FOREIGN KEY (%s) REFERENCES %s(%s) ON DELETE %s ON UPDATE %s",
						colName, fkParts[0], fkParts[1], onDelete, onUpdate))
			}
		}

		// Handle soft keys (application-level relationship, no DB constraint)
		// skey creates an indexed column for logical foreign key relationships
		// Supports ondelete:cascade|setnull for application-level behavior
		if skey, ok := tags["skey"]; ok {
			_ = skey // Store metadata for application-level cascade/setnull
			// Create index for soft key (improves query performance)
			indexes = append(indexes,
				fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_%s ON %s(%s);",
					tableName, colName, tableName, colName))
		}

		// Handle indexes
		if _, ok := tags["index"]; ok {
			indexes = append(indexes,
				fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_%s ON %s(%s);",
					tableName, colName, tableName, colName))
		}
	}

	sql.WriteString(strings.Join(columnDefs, ",\n"))

	// Add primary key constraint
	if len(primaryKeys) > 0 {
		sql.WriteString(",\n")
		sql.WriteString(fmt.Sprintf("  PRIMARY KEY (%s)", strings.Join(primaryKeys, ", ")))
	}

	// Add foreign key constraints
	if len(foreignKeys) > 0 {
		sql.WriteString(",\n")
		sql.WriteString(strings.Join(foreignKeys, ",\n"))
	}

	sql.WriteString("\n);")

	return sql.String(), indexes
}

// getPostgresType maps Go types to PostgreSQL types
func (am *AutoMigrator) getPostgresType(field reflect.StructField) string {
	normTag := field.Tag.Get("norm")
	tags := parseNormTags(normTag)

	// Check for explicit type in tag
	if sqlType, ok := tags["type"]; ok {
		return sqlType.(string)
	}

	// Handle pointer types (nullable fields)
	fieldType := field.Type
	if fieldType.Kind() == reflect.Ptr {
		fieldType = fieldType.Elem()
	}

	// Map Go types to PostgreSQL types
	switch fieldType.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		return "INTEGER"
	case reflect.Int64:
		return "BIGINT"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return "BIGINT"
	case reflect.Uint64:
		return "BIGINT"
	case reflect.Float32:
		return "REAL"
	case reflect.Float64:
		return "DOUBLE PRECISION"
	case reflect.Bool:
		return "BOOLEAN"
	case reflect.String:
		// Check for explicit text tag
		if _, ok := tags["text"]; ok {
			return "TEXT"
		}
		// Check for max length
		if maxLen, ok := tags["max"]; ok {
			return fmt.Sprintf("VARCHAR(%s)", maxLen.(string))
		}
		// Default to VARCHAR(255)
		return "VARCHAR(255)"
	case reflect.Struct:
		// Handle time.Time (including *time.Time)
		typeName := fieldType.String()
		if typeName == "time.Time" {
			return "TIMESTAMP"
		}
		// All other structs default to JSONB
		return "JSONB"
	case reflect.Slice:
		// Handle []byte as BYTEA
		if fieldType.Elem().Kind() == reflect.Uint8 {
			return "BYTEA"
		}

		// Check for explicit array type tag (e.g., norm:"type:TEXT[]")
		// This allows PostgreSQL native arrays

		// Map Go slice types to PostgreSQL arrays
		elemKind := fieldType.Elem().Kind()
		switch elemKind {
		case reflect.String:
			return "TEXT[]" // PostgreSQL text array
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
			return "INTEGER[]" // PostgreSQL integer array
		case reflect.Int64:
			return "BIGINT[]" // PostgreSQL bigint array
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
			return "BIGINT[]" // PostgreSQL bigint array
		case reflect.Uint64:
			return "BIGINT[]" // PostgreSQL bigint array
		case reflect.Float32:
			return "REAL[]" // PostgreSQL real array
		case reflect.Float64:
			return "DOUBLE PRECISION[]" // PostgreSQL double precision array
		case reflect.Bool:
			return "BOOLEAN[]" // PostgreSQL boolean array
		default:
			// Complex types (structs, nested slices) → JSONB
			return "JSONB"
		}
	case reflect.Map:
		// Maps are stored as JSONB
		return "JSONB"
	default:
		return "TEXT"
	}
}

// parseNormTags parses norm struct tags
func parseNormTags(tag string) map[string]interface{} {
	tags := make(map[string]interface{})
	if tag == "" {
		return tags
	}

	parts := strings.Split(tag, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if strings.Contains(part, ":") {
			kv := strings.SplitN(part, ":", 2)
			tags[kv[0]] = kv[1]
		} else {
			tags[part] = true
		}
	}

	return tags
}

// getTableNameFromStruct determines the table name for a model.
// It first prefers the explicitly registered table name from the registry,
// and only falls back to deriving from the struct name (snake_case + plural)
// when no registration is found. This keeps migrations consistent with
// RegisterTable(.., "custom_name").
func getTableNameFromStruct(model interface{}) string {
	// Prefer explicit registration from the table registry
	if registered := registry.GetRegisteredTableName(model); registered != "" {
		return registered
	}

	// Fallback: derive from struct name
	t := reflect.TypeOf(model)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	name := t.Name()
	// Convert to snake_case and pluralize
	snakeName := utils.ToSnakeCase(name)
	return utils.Pluralize(snakeName)
}
