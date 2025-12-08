package main

import (
	"fmt"
	"log"
	"os"

	"github.com/skssmd/norm"
	"github.com/skssmd/norm/core/registry"
)

// Example table structs
type User struct{}
type Order struct{}
type Analytics struct{}

// setupGlobalMonolith configures a single primary database with replicas
func setupGlobalMonolith(dsn string) {
	fmt.Println("=== Global Monolith Scenario ===")

	// Register primary connection
	err := registry.Register(dsn).Primary()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("✓ Primary connection registered")

	// Add a replica
	err = registry.Register(dsn).Replica()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("✓ Replica connection registered")

	// Trying to add Read pool alongside Primary should fail
	err = registry.Register(dsn).Read()
	if err != nil {
		fmt.Println("Expected error:", err)
	}

	// Register tables to global
	fmt.Println("\n=== Table Registration ===")

	err = registry.Table(User{}).Global()
	if err != nil {
		fmt.Println("User registration error:", err)
	} else {
		fmt.Println("✓ User table registered to global")
	}

	err = registry.Table(Order{}).Global()
	if err != nil {
		fmt.Println("Order registration error:", err)
	} else {
		fmt.Println("✓ Order table registered to global")
	}
}

// setupReadWriteSplit configures separate read and write pools
func setupReadWriteSplit(dsnWrite, dsnRead1, dsnRead2 string) {
	fmt.Println("=== Read/Write Split Scenario ===")

	// Register write pool
	err := registry.Register(dsnWrite).Write()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("✓ Write pool registered")

	// Register multiple read pools
	err = registry.Register(dsnRead1).Read()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("✓ Read pool 1 registered")

	err = registry.Register(dsnRead2).Read()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("✓ Read pool 2 registered")

	// Trying to add another Write should fail
	err = registry.Register(dsnWrite).Write()
	if err != nil {
		fmt.Println("Expected error:", err)
	}

	// Register tables to global
	fmt.Println("\n=== Table Registration ===")

	err = registry.Table(User{}).Global()
	if err != nil {
		fmt.Println("User registration error:", err)
	} else {
		fmt.Println("✓ User table registered to global")
	}

	err = registry.Table(Order{}).Global()
	if err != nil {
		fmt.Println("Order registration error:", err)
	} else {
		fmt.Println("✓ Order table registered to global")
	}
}

// setupSharding configures multiple shards with different tables
func setupSharding(dsn1, dsn2 string) {
	fmt.Println("=== Shard Scenario ===")

	// Shard 1 primary
	err := registry.Register(dsn1).Shard("shard1").Primary()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("✓ Shard1 primary registered")

	// Shard 1 standalone table
	err = registry.Register(dsn1).Shard("shard1").Standalone()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("✓ Shard1 standalone registered")

	// Shard 2 primary
	err = registry.Register(dsn2).Shard("shard2").Primary()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("✓ Shard2 primary registered")

	// Trying to register global Primary when shards exist should fail
	err = registry.Register(dsn1).Primary()
	if err != nil {
		fmt.Println("Expected error:", err)
	}

	// Register tables to shards
	fmt.Println("\n=== Table Registration ===")

	// Try to register User table to global (should fail because mode is "shard")
	err = registry.Table(User{}).Global()
	if err != nil {
		fmt.Println("Expected error - User global registration:", err)
	}

	// Register Order table to Shard1 with primary role
	err = registry.Table(Order{}).Shard("shard1").Primary()
	if err != nil {
		fmt.Println("Order shard1 registration error:", err)
	} else {
		fmt.Println("✓ Order table registered to shard1 (primary)")
	}

	// Register Analytics table to Shard2 with standalone role
	err = registry.Table(Analytics{}).Shard("shard2").Standalone()
	if err != nil {
		fmt.Println("Analytics shard2 registration error:", err)
	} else {
		fmt.Println("✓ Analytics table registered to shard2 (standalone)")
	}
}

// setupShardingWithReadWrite configures shards with table-based read/write roles
// Tables are assigned roles (primary/read/write) which determine which pool to use
func setupShardingWithReadWrite(dsn1, dsn2 string) {
	fmt.Println("=== Shard with Table-Based Read/Write Roles ===")
	fmt.Println("(Tables registered with 'read' or 'write' roles)")

	// Shard 1: Primary pool (handles both read+write for transactional tables)
	err := registry.Register(dsn1).Shard("shard1").Primary()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("✓ Shard1 primary registered")

	// Shard 2: Primary pool
	err = registry.Register(dsn2).Shard("shard2").Primary()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("✓ Shard2 primary registered")

	// Register tables with different role strategies
	fmt.Println("\n=== Table Registration with Roles ===")

	// User table: Role = "primary" (transactional, needs consistency)
	err = registry.Table(User{}).Shard("shard1").Primary()
	if err != nil {
		fmt.Println("User registration error:", err)
	} else {
		fmt.Println("✓ User table → Shard1, Role: primary (transactional)")
	}

	// Order table: Role = "write" (write-heavy operations)
	// When querying, the router will use write pool if available, else primary
	err = registry.Table(Order{}).Shard("shard2").Write()
	if err != nil {
		fmt.Println("Order registration error:", err)
	} else {
		fmt.Println("✓ Order table → Shard2, Role: write (write-heavy)")
	}

	// Analytics table: Role = "read" (read-only, reporting)
	// When querying, the router will use read pool if available, else primary
	err = registry.Table(Analytics{}).Shard("shard2").Read()
	if err != nil {
		fmt.Println("Analytics registration error:", err)
	} else {
		fmt.Println("✓ Analytics table → Shard2, Role: read (read-only)")
	}

	fmt.Println("\n📝 Note: The 'read' and 'write' roles are stored in TableRegistry.")
	fmt.Println("   The query router will use these roles to select the appropriate pool.")
	fmt.Println("   Example: Analytics table with 'read' role → uses read pool for SELECT queries")
}

func main() {
	// Load connection strings from environment variables
	dsn := os.Getenv("DATABASE_DSN")

	dsn2 := os.Getenv("DATABASE_DSN2")

	// Choose ONE setup function to run:
	// setupGlobalMonolith(dsn)                    // Single DB with replicas
	setupReadWriteSplit(dsn, dsn, dsn2) // Separate read/write DBs
	// setupSharding(dsn, dsn2)                    // Multiple shards
	//setupShardingWithReadWrite(dsn, dsn2) // Shards with table-based read/write roles

	// Print registry summary
	norm.Norm()
}
