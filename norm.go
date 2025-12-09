package norm

import (
	"fmt"
	"log"
	"strings"

	"github.com/skssmd/norm/core/migration"
	"github.com/skssmd/norm/core/registry"
)

var autoMigrator *migration.AutoMigrator

// ============================================================
// Database Connection Registration
// ============================================================

// Register starts the database connection registration process
// Returns a ConnBuilder for fluent API chaining
func Register(dsn string) *registry.ConnBuilder {
	return registry.Register(dsn)
}

// ============================================================
// Table Registration
// ============================================================

// Table registers a table model for migration and routing
// In global mode, tables are automatically registered as global
// In shard mode, you must call .Shard("name").Primary() or .Shard("name").Standalone()
func Table(model interface{}) *registry.TableBuilder {
	return registry.Table(model)
}

// ============================================================
// Migration Functions
// ============================================================

// AutoMigrate registers a model struct for automatic migration
func AutoMigrate(models ...interface{}) {
	for _, model := range models {
		autoMigrator.AddModel(model)
	}
}

// DropTables drops all registered tables from all databases (useful for development)
// WARNING: This will delete all data! Use with caution.
func DropTables() error {
	// Initialize auto migrator if not already done
	if autoMigrator == nil {
		autoMigrator = migration.NewAutoMigrator()

		// Get all registered models
		allModels := registry.GetAllModels()
		for _, model := range allModels {
			autoMigrator.AddModel(model)
		}
	}

	return autoMigrator.DropAllTables()
}

// NewMigrator creates a new migrator for table creation
func NewMigrator() *migration.Migrator {
	return migration.NewMigrator()
}

func Norm() {
	// Initialize auto migrator AFTER tables are registered
	autoMigrator = migration.NewAutoMigrator()

	// Get all registered models and add them to auto migrator
	allModels := registry.GetAllModels()
	for _, model := range allModels {
		autoMigrator.AddModel(model)
	}

	// ------------------------
	// Run Auto Migrations
	// ------------------------
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("RUNNING AUTO MIGRATIONS")
	fmt.Println(strings.Repeat("=", 60) + "\n")

	if err := autoMigrator.AutoMigrate(); err != nil {
		log.Fatal("Auto migration failed:", err)
	}

	fmt.Println("\n✅ All auto migrations completed successfully!")

	// ------------------------
	// Print Registry Summary
	// ------------------------
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("REGISTRY SUMMARY")
	fmt.Println(strings.Repeat("=", 60))

	// DB Registry Info
	dbInfo := registry.GetRegistryInfo()
	fmt.Println("\n📊 Database Connection Registry:")
	fmt.Printf("  Mode: %v\n", dbInfo["mode"])
	fmt.Printf("  Total Connection Pools: %d\n", registry.GetPoolCount())

	if pools, ok := dbInfo["pools"].(map[string]interface{}); ok && len(pools) > 0 {
		fmt.Println("\n  Global Pools:")
		for poolName := range pools {
			fmt.Printf("    • %s\n", poolName)
		}
	}

	if shards, ok := dbInfo["shards"].(map[string]map[string]interface{}); ok && len(shards) > 0 {
		fmt.Println("\n  Shards:")
		for shardName, shardInfo := range shards {
			fmt.Printf("    • %s:\n", shardName)
			if hasPrimary, ok := shardInfo["has_primary"].(bool); ok && hasPrimary {
				fmt.Println("        - Primary: ✓")
			}
			if standalonePools, ok := shardInfo["standalone_pools"].([]string); ok && len(standalonePools) > 0 {
				fmt.Printf("        - Standalone pools: %v\n", standalonePools)
			}
		}
	}

	// Table Registry Info
	fmt.Println("\n📋 Table Registry:")
	allTables := registry.ListTables()
	fmt.Printf("  Total Tables Registered: %d\n", len(allTables))

	if len(allTables) > 0 {
		fmt.Println("\n  Table Mappings:")
		for _, tableName := range allTables {
			mapping, err := registry.GetTableMapping(tableName)
			if err == nil {
				if mapping.IsGlobal() {
					fmt.Printf("    • %s → Global (mode: %s)\n", tableName, dbInfo["mode"])
				} else {
					fmt.Printf("    • %s → Shard: %s, Role: %s\n",
						tableName, mapping.ShardName(), mapping.Role())
				}
			}
		}
	}

}
