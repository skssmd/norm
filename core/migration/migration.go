package migration

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/skssmd/norm/core/driver"
	"github.com/skssmd/norm/core/registry"
)

// TableSchema defines the structure of a table to be created
type TableSchema struct {
	TableName string
	Columns   []Column
	Indexes   []Index
}

// Column defines a table column
type Column struct {
	Name       string
	Type       string
	Nullable   bool
	PrimaryKey bool
	Unique     bool
	Default    string
}

// Index defines a table index
type Index struct {
	Name    string
	Columns []string
	Unique  bool
}

// Migrator handles table creation across all registered databases
type Migrator struct {
	schemas []TableSchema
}

// NewMigrator creates a new migrator instance
func NewMigrator() *Migrator {
	return &Migrator{
		schemas: make([]TableSchema, 0),
	}
}

// AddTable adds a table schema to the migrator
func (m *Migrator) AddTable(schema TableSchema) {
	m.schemas = append(m.schemas, schema)
}

// HasSchemas returns true if there are schemas to migrate
func (m *Migrator) HasSchemas() bool {
	return len(m.schemas) > 0
}

// Migrate creates all tables in all databases (except standalone shards)
// It runs migrations on:
// - Global pools (primary, replicas, write, read)
// - Shard primary pools
// It SKIPS:
// - Standalone shard pools
func (m *Migrator) Migrate() error {
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
				if err := m.migratePool(p, fmt.Sprintf("Global:%s", name)); err != nil {
					errChan <- fmt.Errorf("global pool '%s': %w", name, err)
				}
			}(poolName, pool)
		}
	}

	if mode == "shard" {
		// Migrate shard primary pools only (skip standalone)
		shards := regInfo["shards"].(map[string]interface{})
		for shardName, shardInfo := range shards {
			shardData := shardInfo.(map[string]interface{})

			// Only migrate primary pool
			if hasPrimary, ok := shardData["has_primary"].(bool); ok && hasPrimary {
				if primaryPool, ok := shardData["primary_pool"].(*driver.PGPool); ok {
					wg.Add(1)
					go func(shard string, p *driver.PGPool) {
						defer wg.Done()
						if err := m.migratePool(p, fmt.Sprintf("Shard:%s:primary", shard)); err != nil {
							errChan <- fmt.Errorf("shard '%s' primary: %w", shard, err)
						}
					}(shardName, primaryPool)
				}
			}

			// Skip standalone pools
			fmt.Printf("⊘ Skipping standalone pools for shard '%s'\n", shardName)
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
		return fmt.Errorf("migration failed with %d errors: %v", len(errors), errors)
	}

	return nil
}

// migratePool runs migrations on a single pool
func (m *Migrator) migratePool(pool *driver.PGPool, poolLabel string) error {
	ctx := context.Background()

	fmt.Printf("🔄 Migrating %s...\n", poolLabel)

	for _, schema := range m.schemas {
		sql := m.generateCreateTableSQL(schema)

		_, err := pool.Pool.Exec(ctx, sql)
		if err != nil {
			// Check if error is "table already exists"
			if strings.Contains(err.Error(), "already exists") {
				fmt.Printf("  ⚠️  Table '%s' already exists in %s\n", schema.TableName, poolLabel)
				continue
			}
			return fmt.Errorf("failed to create table '%s': %w", schema.TableName, err)
		}

		fmt.Printf("  ✓ Created table '%s' in %s\n", schema.TableName, poolLabel)

		// Create indexes
		for _, index := range schema.Indexes {
			indexSQL := m.generateCreateIndexSQL(schema.TableName, index)
			_, err := pool.Pool.Exec(ctx, indexSQL)
			if err != nil && !strings.Contains(err.Error(), "already exists") {
				return fmt.Errorf("failed to create index '%s': %w", index.Name, err)
			}
		}
	}

	fmt.Printf("✅ Migration completed for %s\n", poolLabel)
	return nil
}

// generateCreateTableSQL generates CREATE TABLE SQL
func (m *Migrator) generateCreateTableSQL(schema TableSchema) string {
	var sql strings.Builder

	sql.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n", schema.TableName))

	// Add columns
	columnDefs := make([]string, 0, len(schema.Columns))
	primaryKeys := make([]string, 0)

	for _, col := range schema.Columns {
		colDef := fmt.Sprintf("  %s %s", col.Name, col.Type)

		if col.PrimaryKey {
			primaryKeys = append(primaryKeys, col.Name)
		}

		if !col.Nullable {
			colDef += " NOT NULL"
		}

		if col.Unique && !col.PrimaryKey {
			colDef += " UNIQUE"
		}

		if col.Default != "" {
			colDef += fmt.Sprintf(" DEFAULT %s", col.Default)
		}

		columnDefs = append(columnDefs, colDef)
	}

	sql.WriteString(strings.Join(columnDefs, ",\n"))

	// Add primary key constraint
	if len(primaryKeys) > 0 {
		sql.WriteString(",\n")
		sql.WriteString(fmt.Sprintf("  PRIMARY KEY (%s)", strings.Join(primaryKeys, ", ")))
	}

	sql.WriteString("\n);")

	return sql.String()
}

// generateCreateIndexSQL generates CREATE INDEX SQL
func (m *Migrator) generateCreateIndexSQL(tableName string, index Index) string {
	uniqueStr := ""
	if index.Unique {
		uniqueStr = "UNIQUE "
	}

	return fmt.Sprintf(
		"CREATE %sINDEX IF NOT EXISTS %s ON %s (%s);",
		uniqueStr,
		index.Name,
		tableName,
		strings.Join(index.Columns, ", "),
	)
}
