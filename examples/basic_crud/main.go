package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/skssmd/norm"
)

// User model definition
type User struct {
	ID        uint       `norm:"index;notnull;pk;auto"`
	Email     string     `norm:"name:useremail;unique;notnull"`
	Name      string     `norm:"name:fullname;notnull"`
	Username  string     `norm:"name:uname;notnull;unique"`
	Age       *uint      `norm:""`
	CreatedAt time.Time  `norm:"notnull;default:NOW()"`
	UpdatedAt *time.Time `norm:"default:NOW()"`
}

func main() {
	// Get database connection string from environment
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/norm_examples?sslmode=disable"
	}

	fmt.Println("🚀 Norm ORM - Basic CRUD Example")
	fmt.Println("==================================\n")

	// 1. Register database connection
	fmt.Println("📡 Connecting to database...")
	err := norm.Register(dsn).Primary()
	if err != nil {
		log.Fatal("Failed to register database:", err)
	}
	fmt.Println("✅ Connected!\n")

	// 2. Register tables
	fmt.Println("📋 Registering tables...")
	norm.RegisterTable(User{}, "users")
	fmt.Println("✅ Tables registered!\n")

	// 3. Run migrations
	fmt.Println("🔄 Running migrations...")
	norm.Norm()

	ctx := context.Background()

	// Clean up existing data
	norm.Table("users").Delete().Exec(ctx)

	// ==========================================
	// CREATE (INSERT)
	// ==========================================
	fmt.Println("\n📝 INSERT Operations")
	fmt.Println("--------------------")

	// Single insert
	age := uint(29)
	user := User{
		Name:     "Alice Williams",
		Email:    "alice@example.com",
		Username: "alice",
		Age:      &age,
	}

	rowsAffected, err := norm.Table(user).Insert().Exec(ctx)
	if err != nil {
		log.Fatal("Insert failed:", err)
	}
	fmt.Printf("✅ Inserted %d user\n", rowsAffected)

	// Bulk insert
	users := []User{
		{Name: "Bob Brown", Email: "bob@example.com", Username: "bob"},
		{Name: "Charlie Davis", Email: "charlie@example.com", Username: "charlie"},
		{Name: "Diana Evans", Email: "diana@example.com", Username: "diana"},
	}

	rowsAffected, err = norm.Table("users").BulkInsert(users).Exec(ctx)
	if err != nil {
		log.Fatal("Bulk insert failed:", err)
	}
	fmt.Printf("✅ Bulk inserted %d users\n", rowsAffected)

	// ==========================================
	// READ (SELECT)
	// ==========================================
	fmt.Println("\n📖 SELECT Operations")
	fmt.Println("--------------------")

	// Count all users
	count, err := norm.Table("users").Select().Count(ctx)
	if err != nil {
		log.Fatal("Count failed:", err)
	}
	fmt.Printf("Total users: %d\n", count)

	// Scan single user
	var foundUser User
	err = norm.Table("users").
		Select().
		Where("username = $1", "alice").
		First(ctx, &foundUser)
	if err != nil {
		log.Fatal("First failed:", err)
	}
	fmt.Printf("✅ Found user: %s (%s)\n", foundUser.Name, foundUser.Email)

	// Scan multiple users
	var allUsers []User
	err = norm.Table("users").
		Select().
		OrderBy("fullname ASC").
		All(ctx, &allUsers)
	if err != nil {
		log.Fatal("All failed:", err)
	}
	fmt.Printf("✅ Scanned %d users:\n", len(allUsers))
	for _, u := range allUsers {
		fmt.Printf("   - %s (%s)\n", u.Name, u.Username)
	}

	// ==========================================
	// UPDATE
	// ==========================================
	fmt.Println("\n✏️  UPDATE Operations")
	fmt.Println("--------------------")

	// Update single field
	rowsAffected, err = norm.Table("users").
		Update("fullname", "Alice Williams Updated").
		Where("username = $1", "alice").
		Exec(ctx)
	if err != nil {
		log.Fatal("Update failed:", err)
	}
	fmt.Printf("✅ Updated %d user(s)\n", rowsAffected)

	// Update multiple fields
	rowsAffected, err = norm.Table("users").
		Update("fullname", "Bob Brown Jr.", "useremail", "bob.jr@example.com").
		Where("username = $1", "bob").
		Exec(ctx)
	if err != nil {
		log.Fatal("Update failed:", err)
	}
	fmt.Printf("✅ Updated %d user(s)\n", rowsAffected)

	// ==========================================
	// DELETE
	// ==========================================
	fmt.Println("\n🗑️  DELETE Operations")
	fmt.Println("--------------------")

	rowsAffected, err = norm.Table("users").
		Delete().
		Where("username = $1", "diana").
		Exec(ctx)
	if err != nil {
		log.Fatal("Delete failed:", err)
	}
	fmt.Printf("✅ Deleted %d user(s)\n", rowsAffected)

	// Final count
	count, err = norm.Table("users").Select().Count(ctx)
	if err != nil {
		log.Fatal("Count failed:", err)
	}
	fmt.Printf("\n📊 Final user count: %d\n", count)

	fmt.Println("\n✅ Example completed successfully!")
}
