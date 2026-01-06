package norm

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/skssmd/norm/core/driver"
	"github.com/skssmd/norm/core/engine"
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

// Reset clears all registry state (connections and tables)
// Useful for testing scenarios
func Reset() {
	registry.Reset()
}

// ============================================================
// Caching Registration
// ============================================================

// RegisterRedis enables Redis caching
// addr: Redis address (e.g. "localhost:6379")
// password: Redis password (empty for none)
// db: Redis DB index (0 for default)
func RegisterRedis(addr, password string, db int) error {
	client, err := driver.ConnectRedis(addr, password, db)
	if err != nil {
		return err
	}
	registry.SetCacher(engine.NewRedisCacher(client))
	return nil
}

// EnableMemoryCache enables in-memory caching
// This is useful for development or non-distributed applications
func EnableMemoryCache() {
	registry.SetCacher(engine.NewMemoryCacher())
}


// ============================================================
// Table Registration
// ============================================================

// RegisterTable registers a table with the ORM for migrations and routing
// Usage:
//
//	norm.RegisterTable(User{}, "users").Shard("shard1").Primary()
//	norm.RegisterTable(User{}).Shard("shard1").Primary()  // Auto-generate table name
func RegisterTable(model interface{}, tableName ...string) *registry.TableModel {
	return registry.Table(model, tableName...)
}

// ============================================================
// Query Builder Functions
// ============================================================

// WithCache creates a query context with caching enabled.
// Use this to check cache BEFORE building the query logic.
// Usage: norm.WithCache(time.Minute, "key").Table(...)
func WithCache(ttl time.Duration, keys ...string) *engine.Query[any] {
	q := &engine.Query[any]{}
	return q.Cache(ttl, keys...)
}

// Table creates a query builder for the specified table or model
// Usage:
//
//	norm.Table("users").Select("id", "name", "email")  // String-based
//	norm.Table(User{Name: "John", Email: "john@example.com"}).Insert()  // Struct-based (ignores zero values)
//	norm.Table("users", "id", "orders", "user_id") // Join syntax
func Table(args ...interface{}) *engine.Query[any] {
	q := &engine.Query[any]{}
	return q.Table(args...)
}

// Model creates a type-safe query builder for a specific model
// Usage:
//
//	user, err := norm.Model(User{...}).Insert().Return()
func Model[T any](model T) *engine.Query[T] {
	q := &engine.Query[T]{}
	return q.From(model)
}

// BulkInsert creates a bulk insert builder from model
// Usage:
//
//	norm.BulkInsert(User{}, []string{"name", "email"}, [][]interface{}{{"John", "john@example.com"}, {"Jane", "jane@example.com"}})
func BulkInsert(model interface{}, columns []string, rows [][]interface{}) *engine.BulkInsertBuilder {
	return engine.BulkInsert(model, columns, rows)
}

// Raw creates a raw SQL query with explicit shard routing
// Usage:
//
//	norm.Raw("SELECT custom_function()", "shard1")
//	norm.Raw("SELECT * FROM users WHERE age > $1", "shard1", 25)
func Raw(query string, shard string, args ...interface{}) *engine.Query[any] {
	q := &engine.Query[any]{}
	q.Raw(query, args...)
	// Set the explicit shard for routing
	// We need to access the internal field, so we'll use a helper
	return setRawShard(q, shard)
}

// setRawShard is a helper to set the rawShard field
// This is a workaround since rawShard is unexported
func setRawShard[T any](q *engine.Query[T], shard string) *engine.Query[T] {
	// We need to export a method in engine.Query to set this
	// For now, let's add a SetShard method to Query
	return q.SetShard(shard)
}

// Join creates a join context for raw SQL queries
// This validates that tables are co-located before executing
// Usage:
//
//	norm.Join("users", "orders").Raw("SELECT u.*, o.* FROM users u JOIN orders o ON u.id = o.user_id")
func Join(table1, table2 string) *engine.Query[any] {
	q := &engine.Query[any]{}
	return q.Join(table1, table2)
}

// Removed F() helper - use field pointers or string literals instead
// Recommended approaches:
// 1. Field pointers: From(user).Select(&user.Name, &user.Email)
// 2. String literals: From(user).Select("name", "email")

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
			table, exists := registry.GetModel(tableName)
			if !exists {
				continue
			}

			if table.IsGlobal() {
				fmt.Printf("    • %s → Global (mode: %s)\n", tableName, dbInfo["mode"])
			} else {
				for role, shards := range table.Roles {
					shardList := make([]string, 0, len(shards))
					for shard := range shards {
						shardList = append(shardList, shard)
					}
					fmt.Printf("    • %s → Role: %s, Shards: %v\n", tableName, role, shardList)
				}
			}
		}
	}
}
