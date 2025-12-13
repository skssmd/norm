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
// sortTablesByDependency returns table names sorted by dependency (no foreign keys first)
func (am *AutoMigrator) sortTablesByDependency() []string {
	var noDeps []string
	var withDeps []string

	for _, tableName := range registry.ListTables() {
		table,exists := registry.GetTable(tableName)
		if !exists{
			fmt.Println(tableName, "is not registered")
		} // assume this returns *TableModel
		if table == nil {
			continue
		}

		hasForeignKey := false
		for _, f := range table.Fields {
			if f.Fkey != "" {
				hasForeignKey = true
				break
			}
		}

		if hasForeignKey {
			withDeps = append(withDeps, tableName)
		} else {
			noDeps = append(noDeps, tableName)
		}
	}

	return append(noDeps, withDeps...)
}

// migratePool runs auto migration on a single pool (for global mode)
func (am *AutoMigrator) migratePool(pool *driver.PGPool, poolLabel string) error {
	ctx := context.Background()

	fmt.Printf("🔄 Auto-migrating %s...\n", poolLabel)

	// Sort table names by dependency (tables without foreign keys first)
	sortedTableNames := am.sortTablesByDependency()

	for _, tableName := range sortedTableNames {
	
		// Check if table exists in DB
		existsInDB, err := am.tableExists(ctx, pool, tableName)
		if err != nil {
			return fmt.Errorf("failed to check table '%s': %w", tableName, err)
		}

		if !existsInDB {
			// Create new table
			if err := am.createTable(ctx, pool, tableName); err != nil {
				return fmt.Errorf("failed to create table '%s': %w", tableName, err)
			}
			fmt.Printf("  ✓ Created table '%s' in %s\n", tableName, poolLabel)
		} else {
			// Update existing table
			if err := am.updateTable(ctx, pool, tableName); err != nil {
				return fmt.Errorf("failed to update table '%s': %w", tableName, err)
			}
			fmt.Printf("  ✓ Updated table '%s' in %s\n", tableName, poolLabel)
		}
	}

	fmt.Printf("✅ Auto-migration completed for %s\n", poolLabel)
	return nil
}

// migratePoolForShard runs auto migration on a shard pool, only for tables registered to that shard

// migratePoolForShard runs auto migration on a shard pool, only for tables registered to that shard
func (am *AutoMigrator) migratePoolForShard( pool *driver.PGPool, shardName, poolLabel string) error {
	fmt.Printf("🔄 Auto-migrating %s...\n", poolLabel)
ctx := context.Background()
	// Filter models that belong to this shard
	shardModels := am.getModelsForShard(shardName)
	if len(shardModels) == 0 {
		fmt.Printf("  ⊘ No tables registered for %s\n", poolLabel)
		return nil
	}

	// Sort models by dependency (tables without foreign keys first)
	sortedModels := am.sortModelsByDependencyFromList(shardModels)

	var migrationErrors []error

	for _, model := range sortedModels {
		tableName := getTableNameFromStruct(model)

		// Check if table exists
		exists, err := am.tableExists(ctx, pool, tableName)
		if err != nil {
			migrationErrors = append(migrationErrors, fmt.Errorf("check table '%s': %w", tableName, err))
			continue
		}

		if !exists {
			// Create new table
			if err := am.createTable(ctx, pool, tableName); err != nil {
				migrationErrors = append(migrationErrors, fmt.Errorf("create table '%s': %w", tableName, err))
				continue
			}
			fmt.Printf("  ✓ Created table '%s' in %s\n", tableName, poolLabel)
		} else {
			// Update existing table
			if err := am.updateTable(ctx, pool,  tableName); err != nil {
				migrationErrors = append(migrationErrors, fmt.Errorf("update table '%s': %w", tableName, err))
				continue
			}
			fmt.Printf("  ✓ Updated table '%s' in %s\n", tableName, poolLabel)
		}
	}

	if len(migrationErrors) > 0 {
		return fmt.Errorf("migration errors for %s: %v", poolLabel, migrationErrors)
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
func (am *AutoMigrator) createTable(ctx context.Context, pool *driver.PGPool,  tableName string) error {
	createSQL, indexSQLs := am.generateCreateTableSQL( tableName)

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
func (am *AutoMigrator) updateTable(ctx context.Context, pool *driver.PGPool, tableName string) error {
    // Look up the model from registry
    model, exists := registry.GetTable(tableName)
    if !exists {
        return fmt.Errorf("table model for '%s' not found", tableName)
    }

    existingCols, err := am.getExistingColumns(ctx, pool, tableName)
    if err != nil {
        return err
    }

    desiredCols := am.parseStructColumns(*model)

    for colName, colDef := range desiredCols {
        if _, exists := existingCols[colName]; !exists {
            alterSQL := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s;", tableName, colName, colDef)
            if _, err := pool.Pool.Exec(ctx, alterSQL); err != nil {
                return fmt.Errorf("failed to add column '%s': %w", colName, err)
            }
            fmt.Printf("    + Added column '%s' to '%s'\n", colName, tableName)
        }
    }

    if err := am.addIndexesAndConstraints(ctx, pool, tableName); err != nil {
        return err
    }

    return nil
}


// addIndexesAndConstraints adds missing indexes and foreign keys to existing tables
func (am *AutoMigrator) addIndexesAndConstraints(ctx context.Context, pool *driver.PGPool, tableName string) error {
		table, exists := registry.GetTable(tableName)

	if !exists {
		return fmt.Errorf("table not registered: %s", tableName)
	}

	for _, f := range table.Fields {
		// Index for regular or soft key
		if f.Indexed || f.Skey != "" {
			indexName := fmt.Sprintf("idx_%s_%s", tableName, f.Fieldname)
			indexSQL := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s(%s);", indexName, tableName, f.Fieldname)
			if _, err := pool.Pool.Exec(ctx, indexSQL); err != nil && !strings.Contains(err.Error(), "already exists") {
				fmt.Printf("    ⚠️  Failed to create index '%s': %v\n", indexName, err)
			}
		}

		// Foreign key
		if f.Fkey != "" {
			fkParts := strings.Split(f.Fkey, ".")
			if len(fkParts) != 2 {
				continue
			}

			onDelete := f.OnDelete
			if onDelete == "" {
				onDelete = "NO ACTION"
			}
			onUpdate := "NO ACTION" // optionally you can store this in Field struct if needed

			fkName := fmt.Sprintf("fk_%s_%s", tableName, f.Fieldname)
			fkSQL := fmt.Sprintf(
				"ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s(%s) ON DELETE %s ON UPDATE %s;",
				tableName, fkName, f.Fieldname, fkParts[0], fkParts[1], onDelete, onUpdate,
			)

			if _, err := pool.Pool.Exec(ctx, fkSQL); err != nil {
				if !strings.Contains(err.Error(), "already exists") {
					fmt.Printf("    ⚠️  Failed to create foreign key '%s': %v\n", fkName, err)
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
func (am *AutoMigrator) parseStructColumns(table registry.TableModel) map[string]string {
	columns := make(map[string]string)

	for _, f := range table.Fields {
		columns[f.Fieldname] = f.Fieldtype
	}

	return columns
}

// generateCreateTableSQL generates CREATE TABLE SQL from struct
// Returns the CREATE TABLE statement and a slice of CREATE INDEX statements
func (am *AutoMigrator) generateCreateTableSQL(tableName string) (string, []string) {
	table, exists := registry.GetTable(tableName)

	if !exists {
		panic("table not registered: " + tableName)
	}

	var sql strings.Builder
	sql.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n", tableName))

	columnDefs := make([]string, 0)
	var primaryKeys []string
	var foreignKeys []string
	var indexes []string

	for _, f := range table.Fields {
		colDef := fmt.Sprintf("  %s %s", f.Fieldname, f.Fieldtype)

		if f.Pk {
			colDef += " PRIMARY KEY"
			primaryKeys = append(primaryKeys, f.Fieldname)
		}
		if f.Unique {
			colDef += " UNIQUE"
		}
		if f.OnDelete != "" && f.Fkey != "" {
			fkParts := strings.Split(f.Fkey, ".")
			foreignKeys = append(foreignKeys,
				fmt.Sprintf("  FOREIGN KEY (%s) REFERENCES %s(%s) ON DELETE %s", f.Fieldname, fkParts[0], fkParts[1], f.OnDelete))
		}
		columnDefs = append(columnDefs, colDef)

		if f.Indexed || f.Skey != "" {
			indexes = append(indexes,
				fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_%s ON %s(%s);", tableName, f.Fieldname, tableName, f.Fieldname))
		}
	}

	sql.WriteString(strings.Join(columnDefs, ",\n"))
	sql.WriteString("\n);")

	if len(foreignKeys) > 0 {
		sql.WriteString("\n")
		sql.WriteString(strings.Join(foreignKeys, ",\n"))
	}

	return sql.String(), indexes
}

// getPostgresType maps Go types to PostgreSQL types


// parseNormTags parses norm struct tags


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
