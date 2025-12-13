package main

import (
	"fmt"
	"log"

	"github.com/skssmd/norm"
)

// // Example table structs for testing
// type User struct {
// 	ID        uint       `norm:"index;notnull;pk;auto"`
// 	Email     string     `norm:"name:useremail;unique;notnull"` // VARCHAR(255) - default
// 	Name      string     `norm:"name:fullname;notnull"`        // VARCHAR(255) - default
// 	Username  string     `norm:"name:uname;notnull;unique"` // VARCHAR(255) - default
// 	Bio       *string    `norm:"text"`           // TEXT - nullable (pointer)
// 	Age       *uint      `norm:""`               // BIGINT - nullable (pointer)
// 	CreatedAt time.Time  `norm:"notnull;default:NOW()"`
// 	UpdatedAt *time.Time `norm:"default:NOW()"` // TIMESTAMP - nullable (pointer)
// }

// type Order struct {
// 	ID        uint      `norm:"index;notnull;pk;auto"`
// 	UserID    uint      `norm:"index;notnull;fkey:users.id;ondelete:cascade"`
// 	Total     float64   `norm:"notnull"`
// 	Status    string    `norm:"max:20;default:'pending'"` // VARCHAR(20) - explicit max
// 	Notes     string    `norm:"text"`                     // TEXT - for long content
// 	CreatedAt time.Time `norm:"notnull;default:NOW()"`
// }

// type Analytics struct {
// 	ID        uint                   `norm:"index;notnull;pk;auto"`
// 	UserID    *uint                  `norm:"skey:users.id;ondelete:setnull"` // Soft key - nullable, app-level cascade
// 	EventType string                 `norm:"index;notnull;max:100"`          // VARCHAR(100)
// 	EventData string                 `norm:"type:JSONB"`                     // JSONB - explicit type
// 	Tags      []string               `norm:""`                               // TEXT[] - PostgreSQL array
// 	Scores    []int                  `norm:""`                               // INTEGER[] - PostgreSQL array
// 	Metadata  map[string]interface{} `norm:""`                               // JSONB - map type
// 	CreatedAt time.Time              `norm:"index;notnull;default:NOW()"`
// }

// type Log struct {
// 	ID        uint      `norm:"index;notnull;pk;auto"`
// 	EventType string    `norm:"index;notnull;max:100"` // VARCHAR(100)
// 	EventData string    `norm:"type:JSONB"`            // JSONB
// 	Message   string    `norm:"text;notnull"`          // TEXT - for log messages
// 	CreatedAt time.Time `norm:"index;notnull;default:NOW()"`
// }

// setupGlobalMonolith configures a single primary database with replicas
func setupGlobalMonolith() {
	dsns := GetDSNs()
	dsn := dsns.Primary
	fmt.Println("=== Global Monolith Scenario ===")

	// Register primary connection
	err := norm.Register(dsn).Primary()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("✓ Primary connection registered")

	// Add a replica
	err = norm.Register(dsn).Replica()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("✓ Replica connection registered")

	// Trying to add Read pool alongside Primary should fail
	err = norm.Register(dsn).Read()
	if err != nil {
		fmt.Println("Expected error:", err)
	}

	// Register tables (auto-registered as global in global mode)
	fmt.Println("\n=== Table Registration ===")

	norm.RegisterTable(User{}, "users")
	fmt.Println("✓ User table registered")

	norm.RegisterTable(Order{}, "orders")
	fmt.Println("✓ Order table registered")

	norm.RegisterTable(Profile{}, "profiles")
	fmt.Println("✓ Profile table registered")

	norm.RegisterTable(Log{}, "logs")
	fmt.Println("✓ Log table registered")
}

// setupReadWriteSplit configures separate read and write pools
func setupReadWriteSplit() {
	dsns := GetDSNs()
	dsnWrite := dsns.Primary
	dsnRead1 := dsns.Replica1
	dsnRead2 := dsns.Replica2
	fmt.Println("=== Read/Write Split Scenario ===")

	// Register write pool
	err := norm.Register(dsnWrite).Write()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("✓ Write pool registered")

	// Register multiple read pools
	err = norm.Register(dsnRead1).Read()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("✓ Read pool 1 registered")

	err = norm.Register(dsnRead2).Read()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("✓ Read pool 2 registered")

	// Trying to add another Write should fail
	err = norm.Register(dsnWrite).Write()
	if err != nil {
		fmt.Println("Expected error:", err)
	}

	// Register tables (auto-registered as global in global mode)
	fmt.Println("\n=== Table Registration ===")

	norm.RegisterTable(User{}, "users")
	fmt.Println("✓ User table registered")

	norm.RegisterTable(Order{}, "orders")
	fmt.Println("✓ Order table registered")

	norm.RegisterTable(Analytics{}, "analytics")
	fmt.Println("✓ Analytics table registered")
}

// setupSharding configures multiple shards with different tables
func setupSharding() {
	dsns := GetDSNs()
	dsn1 := dsns.Primary
	dsn2 := dsns.Replica1
	dsn3 := dsns.Replica2
	fmt.Println("=== Shard Scenario ===")

	// Shard 1 primary (for general tables)
	err := norm.Register(dsn1).Shard("shard1").Primary()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("✓ Shard1 primary registered")

	// Shard 2 standalone (for isolated tables that don't need migration)
	err = norm.Register(dsn2).Shard("shard2").Standalone()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("✓ Shard2 standalone registered")
	err = norm.Register(dsn3).Shard("shard3").Standalone()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("✓ Shard3 standalone registered")

	// Trying to add standalone to shard1 (should fail - can't be both primary and standalone)
	err = norm.Register(dsn1).Shard("shard1").Standalone()
	if err != nil {
		fmt.Println("Expected error:", err)
	}

	// Trying to register global Primary when shards exist should fail
	err = norm.Register(dsn1).Primary()
	if err != nil {
		fmt.Println("Expected error:", err)
	}

	// Register tables to shards
	fmt.Println("\n=== Table Registration ===")

	// In shard mode, tables must be explicitly assigned to shards
	// norm.Table(User{}) would not auto-register in shard mode

	// Register User table to Shard1 with pr
	// imary role
	err = norm.RegisterTable(User{}, "users").Primary("shard1")
	if err != nil {
		fmt.Println("User shard1 registration error:", err)
	} else {
		fmt.Println("✓ User table registered to shard1 (primary)")
	}

	// Register Order table to Shard1 with primary role
	err = norm.RegisterTable(Order{}, "orders").Primary("shard1")
	if err != nil {
		fmt.Println("Order shard1 registration error:", err)
	} else {
		fmt.Println("✓ Order table registered to shard1 (primary)")
	}

	// Register Analytics table to Shard2 with standalone role (won't be migrated)
	if err != nil {
		fmt.Println("Analytics shard2 registration error:", err)
	} else {
		fmt.Println("✓ Analytics table registered to shard2 (standalone - no migration)")
	}

	// Register Profile table to Shard1
	err = norm.RegisterTable(Profile{}, "profiles").Primary("shard1")
	if err != nil {
		fmt.Println("Profile shard1 registration error:", err)
	} else {
		fmt.Println("✓ Profile table registered to shard1 (primary)")
	}
	err = norm.RegisterTable(Log{}, "logs").Standalone("shard2")
	if err != nil {
		fmt.Println("Analytics shard2 registration error:", err)
	} else {
		fmt.Println("✓ Analytics table registered to shard2 (standalone - no migration)")
	}

}

// setupShardingWithReadWrite configures shards with table-based read/write roles
// Tables are assigned roles (primary/read/write) which determine which pool to use
func setupShardingWithReadWrite() {
	dsns := GetDSNs()
	dsn1 := dsns.Primary
	dsn2 := dsns.Replica1
	fmt.Println("=== Shard with Table-Based Read/Write Roles ===")
	fmt.Println("(Tables registered with 'read' or 'write' roles)")

	// Shard 1: Primary pool (handles both read+write for transactional tables)
	err := norm.Register(dsn1).Shard("shard1").Primary()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("✓ Shard1 primary registered")

	// Shard 2: Primary pool
	err = norm.Register(dsn2).Shard("shard2").Primary()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("✓ Shard2 primary registered")

	// Register tables with different role strategies
	fmt.Println("\n=== Table Registration with Roles ===")

	// User table: Role = "primary" (transactional, needs consistency)
	err = norm.RegisterTable(User{}).Primary("shard1")
	if err != nil {
		fmt.Println("User registration error:", err)
	} else {
		fmt.Println("✓ User table → Shard1, Role: primary (transactional)")
	}

	// Order table: Role = "write" (write-heavy operations)
	// When querying, the router will use write pool if available, else primary
	err = norm.RegisterTable(Order{}).Write("shard2")
	if err != nil {
		fmt.Println("Order registration error:", err)
	} else {
		fmt.Println("✓ Order table → Shard2, Role: write (write-heavy)")
	}

	// Analytics table: Role = "read" (read-only, reporting)
	// When querying, the router will use read pool if available, else primary
	if err != nil {
		fmt.Println("Analytics registration error:", err)
	} else {
		fmt.Println("✓ Analytics table → Shard2, Role: read (read-only)")
	}

	norm.RegisterTable(Profile{}, "profiles").Primary("shard1")

	fmt.Println("\n📝 Note: The 'read' and 'write' roles are stored in the table registry.")
	fmt.Println("   The query router will use these roles to select the appropriate pool.")
	fmt.Println("   Example: Analytics table with 'read' role → uses read pool for SELECT queries")
}

func maina() {
	// Load connection strings from environment variables
	// dsns := GetDSNs()

	// Choose ONE setup function to run:
	// setupGlobalMonolith()                    // Single DB with replicas
	//setupReadWriteSplit() // Separate read/write DBs
	setupSharding() // Multiple shards
	// setupShardingWithReadWrite()       // Shards with table-based read/write roles

	// Drop all tables (useful for development to start fresh)
	// Uncomment to drop tables before migration
	if err := norm.DropTables(); err != nil {
		log.Fatal("Failed to drop tables:", err)
	}

	// Run auto migrations and print registry summary
	norm.Norm()

	// Run query builder examples
	RunQueryExamples()
}
