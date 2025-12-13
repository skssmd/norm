package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/skssmd/norm"
)

// Example table structs for testing
type User struct {
	ID        uint       `norm:"index;notnull;pk;auto"`
	Email     string     `norm:"name:useremail;unique;notnull"` // VARCHAR(255) - default
	Name      string     `norm:"name:fullname;notnull"`        // VARCHAR(255) - default
	Username  string     `norm:"name:uname;notnull;unique"` // VARCHAR(255) - default
	Bio       *string    `norm:"text"`           // TEXT - nullable (pointer)
	Age       *uint      `norm:""`               // BIGINT - nullable (pointer)
	CreatedAt time.Time  `norm:"notnull;default:NOW()"`
	UpdatedAt *time.Time `norm:"default:NOW()"` // TIMESTAMP - nullable (pointer)
}

type Order struct {
	ID        uint      `norm:"index;notnull;pk;auto"`
	UserID    uint      `norm:"index;notnull;fkey:users.id;ondelete:cascade"`
	Total     float64   `norm:"notnull"`
	Status    string    `norm:"max:20;default:'pending'"` // VARCHAR(20) - explicit max
	Notes     string    `norm:"text"`                     // TEXT - for long content
	CreatedAt time.Time `norm:"notnull;default:NOW()"`
}

type Profile struct {
	ID     int    `norm:"pk;auto"`
	UserID int    `norm:"skey:users.id"`
	Bio    string
}

type Analytics struct {
	ID        uint                   `norm:"index;notnull;pk;auto"`
	UserID    *uint                  `norm:"skey:users.id;ondelete:setnull"` // Soft key - nullable, app-level cascade
	EventType string                 `norm:"index;notnull;max:100"`          // VARCHAR(100)
	EventData string                 `norm:"type:JSONB"`                     // JSONB - explicit type
	Tags      []string               `norm:""`                               // TEXT[] - PostgreSQL array
	Scores    []int                  `norm:""`                               // INTEGER[] - PostgreSQL array
	Metadata  map[string]interface{} `norm:""`                               // JSONB - map type
	CreatedAt time.Time              `norm:"index;notnull;default:NOW()"`
}

type Log struct {
	ID        uint      `norm:"index;notnull;pk;auto"`
	EventType string    `norm:"index;notnull;max:100"` // VARCHAR(100)
	EventData string    `norm:"type:JSONB"`            // JSONB
	Message   string    `norm:"text;notnull"`          // TEXT - for log messages
	CreatedAt time.Time `norm:"index;notnull;default:NOW()"`
}

// TestScenario represents a database connection scenario to test
type TestScenario struct {
	Name        string
	SetupFunc   func()
	CleanupFunc func()
}

// Helper function to repeat a string
func repeatString(s string, count int) string {
	return strings.Repeat(s, count)
}

// Helper function to center text
func centerText(text string, width int) string {
	if len(text) >= width {
		return text
	}
	padding := (width - len(text)) / 2
	leftPad := strings.Repeat(" ", padding)
	rightPad := strings.Repeat(" ", width-len(text)-padding)
	return leftPad + text + rightPad
}

// runScenario executes a test scenario with setup, queries, and cleanup
func runScenario(scenario TestScenario) {
	fmt.Println("\n" + repeatString("=", 70))
	fmt.Printf("🧪 TESTING SCENARIO: %s\n", scenario.Name)
	fmt.Println(repeatString("=", 70))

	// Setup the database connection
	scenario.SetupFunc()

	// Drop tables to start fresh
	fmt.Println("\n🗑️  Cleaning up existing tables...")
	if err := norm.DropTables(); err != nil {
		log.Printf("Warning: Failed to drop tables: %v\n", err)
	}

	// Run migrations
	fmt.Println("\n🔄 Running migrations...")
	norm.Norm()

	// Run query examples
	fmt.Println("\n📝 Running query examples...")
	RunQueryExamples()

	// Cleanup
	if scenario.CleanupFunc != nil {
		scenario.CleanupFunc()
	}

	// Reset registry for next scenario
	norm.Reset()

	fmt.Println("\n✅ Scenario completed successfully!")
}

// setupScenario1_GlobalMonolith configures a single primary database with replicas
func setupScenario1_GlobalMonolith() {
	dsns := GetDSNs()
	dsn := dsns.Primary

	fmt.Println("\n📋 Scenario: Global Monolith (Primary + Replicas)")
	fmt.Println("   - Single primary database")
	fmt.Println("   - Multiple replica connections")
	fmt.Println("   - All tables are global")

	// Register primary connection
	err := norm.Register(dsn).Primary()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("  ✓ Primary connection registered")

	// Add a replica
	err = norm.Register(dsn).Replica()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("  ✓ Replica connection registered")

	// Register tables (auto-registered as global in global mode)
	norm.RegisterTable(User{}, "users")
	norm.RegisterTable(Order{}, "orders")
	norm.RegisterTable(User{}, "users")
	norm.RegisterTable(Order{}, "orders")
	norm.RegisterTable(Profile{}, "profiles")
	norm.RegisterTable(Log{}, "logs")
	norm.RegisterTable(Analytics{}, "analytics")
	norm.RegisterTable(Profile{}, "profiles")
	fmt.Println("  ✓ Tables registered (global mode)")
}

// setupScenario2_ReadWriteSplit configures separate read and write pools
func setupScenario2_ReadWriteSplit() {
	dsns := GetDSNs()
	dsnWrite := dsns.Primary
	dsnRead1 := dsns.Replica1
	dsnRead2 := dsns.Replica2

	fmt.Println("\n📋 Scenario: Read/Write Split")
	fmt.Println("   - Dedicated write pool")
	fmt.Println("   - Multiple read pools")
	fmt.Println("   - Load balancing across read replicas")

	// Register write pool
	err := norm.Register(dsnWrite).Write()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("  ✓ Write pool registered")

	// Register multiple read pools
	err = norm.Register(dsnRead1).Read()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("  ✓ Read pool 1 registered")

	err = norm.Register(dsnRead2).Read()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("  ✓ Read pool 2 registered")

	// Register tables (auto-registered as global in global mode)
	norm.RegisterTable(User{}, "users")
	norm.RegisterTable(Order{}, "orders")
	norm.RegisterTable(Profile{}, "profiles")
	norm.RegisterTable(Analytics{}, "analytics")
	fmt.Println("  ✓ Tables registered (global mode)")
}

// setupScenario3_Sharding configures multiple shards with different tables
func setupScenario3_Sharding() {
	dsns := GetDSNs()
	dsn1 := dsns.Primary
	dsn2 := dsns.Replica1
	dsn3 := dsns.Replica2

	fmt.Println("\n📋 Scenario: Sharding")
	fmt.Println("   - Multiple database shards")
	fmt.Println("   - Tables distributed across shards")
	fmt.Println("   - Shard-specific routing")

	// Shard 1 primary (for general tables)
	err := norm.Register(dsn1).Shard("shard1").Primary()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("  ✓ Shard1 primary registered")

	// Shard 2 standalone (for isolated tables)
	err = norm.Register(dsn2).Shard("shard2").Standalone()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("  ✓ Shard2 standalone registered")

	// Shard 3 standalone
	err = norm.Register(dsn3).Shard("shard3").Standalone()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("  ✓ Shard3 standalone registered")

	// Register tables to shards
	err = norm.RegisterTable(User{}, "users").Primary("shard1")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("  ✓ User table → shard1 (primary)")

	err = norm.RegisterTable(Order{}, "orders").Primary("shard1")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("  ✓ Order table → shard1 (primary)")

	err = norm.RegisterTable(Analytics{}, "analytics").Standalone("shard2")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("  ✓ Analytics table → shard2 (standalone)")

	err = norm.RegisterTable(Profile{}, "profiles").Primary("shard1")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("  ✓ Profile table → shard1 (primary)")
}

// setupScenario4_ShardingWithRoles configures shards with table-based read/write roles
func setupScenario4_ShardingWithRoles() {
	dsns := GetDSNs()
	dsn1 := dsns.Primary
	dsn2 := dsns.Replica1

	fmt.Println("\n📋 Scenario: Sharding with Table-Based Roles")
	fmt.Println("   - Multiple shards with primary pools")
	fmt.Println("   - Tables assigned specific roles (primary/read/write)")
	fmt.Println("   - Role-based query routing")

	// Shard 1: Primary pool
	err := norm.Register(dsn1).Shard("shard1").Primary()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("  ✓ Shard1 primary registered")

	// Shard 2: Primary pool
	err = norm.Register(dsn2).Shard("shard2").Primary()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("  ✓ Shard2 primary registered")

	// Register tables with different role strategies
	err = norm.RegisterTable(User{}, "users").Primary("shard1")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("  ✓ User table → Shard1, Role: primary")

	err = norm.RegisterTable(Order{}, "orders").Write("shard2")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("  ✓ Order table → Shard2, Role: write")

	err = norm.RegisterTable(Analytics{}, "analytics").Read("shard2")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("  ✓ Analytics table → Shard2, Role: read")

	err = norm.RegisterTable(Profile{}, "profiles").Primary("shard1")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("  ✓ Profile table → Shard1, Role: primary")
}

func main() {
	fmt.Println("╔" + repeatString("=", 68) + "╗")
	fmt.Println("║" + centerText("NORM ORM - COMPREHENSIVE DATABASE SCENARIO TESTS", 68) + "║")
	fmt.Println("╚" + repeatString("=", 68) + "╝")

	// Define all test scenarios
	scenarios := []TestScenario{
		{
			Name:      "1. Global Monolith (Primary + Replicas)",
			SetupFunc: setupScenario1_GlobalMonolith,
		},
		{
			Name:      "2. Read/Write Split",
			SetupFunc: setupScenario2_ReadWriteSplit,
		},
		{
			Name:      "3. Sharding (Multi-Database)",
			SetupFunc: setupScenario3_Sharding,
		},
		{
			Name:      "4. Sharding with Table-Based Roles",
			SetupFunc: setupScenario4_ShardingWithRoles,
		},
	}

	// Run each scenario
	for i, scenario := range scenarios {
		fmt.Printf("\n\n🔄 Running scenario %d of %d...\n", i+1, len(scenarios))
		runScenario(scenario)

		// Add separator between scenarios
		if i < len(scenarios)-1 {
			fmt.Println("\n" + repeatString("─", 70))
			fmt.Println("Preparing for next scenario...")
			fmt.Println(repeatString("─", 70))
		}
	}

	// Final summary
	fmt.Println("\n\n╔" + repeatString("=", 68) + "╗")
	fmt.Println("║" + centerText("ALL SCENARIOS COMPLETED SUCCESSFULLY! ✅", 68) + "║")
	fmt.Println("╚" + repeatString("=", 68) + "╝")
	fmt.Printf("\n📊 Total scenarios tested: %d\n", len(scenarios))
	fmt.Println("✅ All database connection types validated")
	fmt.Println("✅ All query operations verified")
	fmt.Println("✅ CRUD workflow tested across all scenarios")
}

// Example table structs for testing
