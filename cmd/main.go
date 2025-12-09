package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/skssmd/norm"
)

// Example table structs for testing
type User struct {
	ID        int       `norm:"index;notnull;pk"`
	Email     string    `norm:"unique;notnull"` // VARCHAR(255) - default
	Name      string    `norm:"notnull"`        // VARCHAR(255) - default
	Username  string    `norm:"notnull;unique"` // VARCHAR(255) - default
	Bio       string    `norm:"text"`           // TEXT - explicit text tag
	CreatedAt time.Time `norm:"notnull;default:NOW()"`
	UpdatedAt time.Time `norm:"notnull;default:NOW()"`
}

type Order struct {
	ID        int       `norm:"index;notnull;pk"`
	UserID    int       `norm:"index;notnull;fkey:users.id;ondelete:cascade"`
	Total     float64   `norm:"notnull"`
	Status    string    `norm:"max:20;default:'pending'"` // VARCHAR(20) - explicit max
	Notes     string    `norm:"text"`                     // TEXT - for long content
	CreatedAt time.Time `norm:"notnull;default:NOW()"`
}

type Analytics struct {
	ID        int       `norm:"index;notnull;pk"`
	EventType string    `norm:"index;notnull;max:100"` // VARCHAR(100)
	EventData string    `norm:"type:JSONB"`            // JSONB
	CreatedAt time.Time `norm:"index;notnull;default:NOW()"`
}

type Log struct {
	ID        int       `norm:"index;notnull;pk"`
	EventType string    `norm:"index;notnull;max:100"` // VARCHAR(100)
	EventData string    `norm:"type:JSONB"`            // JSONB
	Message   string    `norm:"text;notnull"`          // TEXT - for log messages
	CreatedAt time.Time `norm:"index;notnull;default:NOW()"`
}

// setupGlobalMonolith configures a single primary database with replicas
func setupGlobalMonolith(dsn string) {
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

	norm.Table(User{})
	fmt.Println("✓ User table registered")

	norm.Table(Order{})
	fmt.Println("✓ Order table registered")

	norm.Table(Log{})
	fmt.Println("✓ Log table registered")
}

// setupReadWriteSplit configures separate read and write pools
func setupReadWriteSplit(dsnWrite, dsnRead1, dsnRead2 string) {
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

	norm.Table(User{})
	fmt.Println("✓ User table registered")

	norm.Table(Order{})
	fmt.Println("✓ Order table registered")

	norm.Table(Analytics{})
	fmt.Println("✓ Analytics table registered")
}

// setupSharding configures multiple shards with different tables
func setupSharding(dsn1, dsn2, dsn3 string) {
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

	// Register User table to Shard1 with primary role
	err = norm.Table(User{}).Shard("shard1").Primary()
	if err != nil {
		fmt.Println("User shard1 registration error:", err)
	} else {
		fmt.Println("✓ User table registered to shard1 (primary)")
	}

	// Register Order table to Shard1 with primary role
	err = norm.Table(Order{}).Shard("shard1").Primary()
	if err != nil {
		fmt.Println("Order shard1 registration error:", err)
	} else {
		fmt.Println("✓ Order table registered to shard1 (primary)")
	}

	// Register Analytics table to Shard2 with standalone role (won't be migrated)
	err = norm.Table(Analytics{}).Shard("shard2").Standalone()
	if err != nil {
		fmt.Println("Analytics shard2 registration error:", err)
	} else {
		fmt.Println("✓ Analytics table registered to shard2 (standalone - no migration)")
	}
	err = norm.Table(Log{}).Shard("shard2").Standalone()
	if err != nil {
		fmt.Println("Analytics shard2 registration error:", err)
	} else {
		fmt.Println("✓ Analytics table registered to shard2 (standalone - no migration)")
	}

}

// setupShardingWithReadWrite configures shards with table-based read/write roles
// Tables are assigned roles (primary/read/write) which determine which pool to use
func setupShardingWithReadWrite(dsn1, dsn2 string) {
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
	err = norm.Table(User{}).Shard("shard1").Primary()
	if err != nil {
		fmt.Println("User registration error:", err)
	} else {
		fmt.Println("✓ User table → Shard1, Role: primary (transactional)")
	}

	// Order table: Role = "write" (write-heavy operations)
	// When querying, the router will use write pool if available, else primary
	err = norm.Table(Order{}).Shard("shard2").Write()
	if err != nil {
		fmt.Println("Order registration error:", err)
	} else {
		fmt.Println("✓ Order table → Shard2, Role: write (write-heavy)")
	}

	// Analytics table: Role = "read" (read-only, reporting)
	// When querying, the router will use read pool if available, else primary
	err = norm.Table(Analytics{}).Shard("shard2").Read()
	if err != nil {
		fmt.Println("Analytics registration error:", err)
	} else {
		fmt.Println("✓ Analytics table → Shard2, Role: read (read-only)")
	}

	fmt.Println("\n📝 Note: The 'read' and 'write' roles are stored in the table registry.")
	fmt.Println("   The query router will use these roles to select the appropriate pool.")
	fmt.Println("   Example: Analytics table with 'read' role → uses read pool for SELECT queries")
}

func main() {
	// Load connection strings from environment variables
	dsn := os.Getenv("DATABASE_DSN")

	dsn2 := os.Getenv("DATABASE_DSN2")

	dsn3 := os.Getenv("DATABASE_DSN3")

	// Choose ONE setup function to run:
	// setupGlobalMonolith(dsn)                    // Single DB with replicas
	//setupReadWriteSplit(dsn, dsn3, dsn2) // Separate read/write DBs
	setupSharding(dsn, dsn2, dsn3) // Multiple shards
	// setupShardingWithReadWrite(dsn, dsn2)       // Shards with table-based read/write roles

	// Drop all tables (useful for development to start fresh)
	// Uncomment to drop tables before migration
	if err := norm.DropTables(); err != nil {
		log.Fatal("Failed to drop tables:", err)
	}

	// Run auto migrations and print registry summary
	norm.Norm()
}
