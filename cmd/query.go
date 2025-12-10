package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/skssmd/norm"
)

func RunQueryExamples() {
	ctx := context.Background()

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("QUERY BUILDER EXAMPLES - STRING-BASED")
	fmt.Println(strings.Repeat("=", 60) + "\n")

	// ========================================
	// SELECT Examples
	// ========================================
	fmt.Println("📖 SELECT Examples:")
	fmt.Println(strings.Repeat("-", 60))

	// Example 1: Simple SELECT
	fmt.Println("\n1. Simple SELECT:")
	count, err := norm.Table("users").
		Select("id", "name", "email").
		Count(ctx)
	if err != nil {
		log.Printf("  ❌ Error: %v\n", err)
	} else {
		fmt.Printf("  ✅ Found %d users\n", count)
	}

	// Example 2: SELECT with WHERE
	fmt.Println("\n2. SELECT with WHERE:")
	count, err = norm.Table("users").
		Select("name", "email", "created_at").
		Where("created_at > $1", time.Now().AddDate(0, -1, 0)).
		Count(ctx)
	if err != nil {
		log.Printf("  ❌ Error: %v\n", err)
	} else {
		fmt.Printf("  ✅ Found %d recent users\n", count)
	}

	// Example 3: SELECT all fields
	fmt.Println("\n3. SELECT all fields:")
	count, err = norm.Table("users").
		Select().
		Count(ctx)
	if err != nil {
		log.Printf("  ❌ Error: %v\n", err)
	} else {
		fmt.Printf("  ✅ Total users: %d\n", count)
	}

	// Example 4: SELECT with pagination
	fmt.Println("\n4. SELECT with pagination:")
	count, err = norm.Table("users").
		Select("id", "name", "email").
		OrderBy("created_at DESC").
		Pagination(10, 0).
		Count(ctx)
	if err != nil {
		log.Printf("  ❌ Error: %v\n", err)
	} else {
		fmt.Printf("  ✅ Page 1: %d users\n", count)
	}

	// ========================================
	// INSERT Examples
	// ========================================
	fmt.Println("\n\n📝 INSERT Examples:")
	fmt.Println(strings.Repeat("-", 60))

	// Example 1: Insert with model
	fmt.Println("\n1. Struct-based insert (non-zero values only):")
	age := uint(29)
	rowsAffected, err := norm.Table(User{
		Name:  "John Doe",
		Email: "john.doe@example.com",
		Age:   &age,
		// Username is empty, will be ignored
	}).Insert().Exec(ctx)
	if err != nil {
		log.Printf("  ❌ Error: %v\n", err)
	} else {
		fmt.Printf("  ✅ Inserted %d user(s)\n", rowsAffected)
	}
	rowsAffected, err = norm.Table(User{
		Name:     "John Doe",
		Email:    "john.doe@example.com",
		Age:      &age,
		Username: "dp2",
		// Username is empty, will be ignored
	}).Insert().Exec(ctx)
	if err != nil {
		log.Printf("  ❌ Error: %v\n", err)
	} else {
		fmt.Printf("  ✅ Inserted %d user(s)\n", rowsAffected)
	}

	// Example 2: Insert another user
	fmt.Println("\n2. Insert another user:")
	user2 := User{
		Name:     "Jane Smith",
		Email:    "jane.smith@example.com",
		Username: "janesmith",
	}
	rowsAffected, err = norm.Table("users").
		Insert(user2).
		Exec(ctx)
	if err != nil {
		log.Printf("  ❌ Error: %v\n", err)
	} else {
		fmt.Printf("  ✅ Inserted %d user(s)\n", rowsAffected)
	}

	// ========================================
	// UPDATE Examples
	// ========================================
	fmt.Println("\n\n✏️  UPDATE Examples:")
	fmt.Println(strings.Repeat("-", 60))

	// Example 1: Update single field (pair-based)
	fmt.Println("\n1. Update single field:")
	rowsAffected, err = norm.Table("users").
		Update("name", "John Doe Updated").
		Where("username = $1", "janesmith").
		Exec(ctx)
	if err != nil {
		log.Printf("  ❌ Error: %v\n", err)
	} else {
		fmt.Printf("  ✅ Updated %d user(s)\n", rowsAffected)
	}

	// Example 2: Pair-based update with multiple fields
	fmt.Println("\n2. Pair-based update (multiple fields):")
	rowsAffected, err = norm.Table("users").
		Update("name", "John Pair Updated", "email", "john.updated@example.com").
		Where("username = $1", "janesmith").
		Exec(ctx)
	if err != nil {
		log.Printf("  ❌ Error: %v\n", err)
	} else {
		fmt.Printf("  ✅ Updated %d user(s)\n", rowsAffected)
	}

	// Example 3: Struct-based update (only non-zero fields)
	fmt.Println("\n3. Struct-based update (non-zero only):")
	rowsAffected, err = norm.Table(User{
		Name: "Jane Struct Updated",
		// Email is empty, will be ignored (keeps old value)
	}).Update().Where("username = $1", "janesmith").Exec(ctx)
	if err != nil {
		log.Printf("  ❌ Error: %v\n", err)
	} else {
		fmt.Printf("  ✅ Updated %d user(s)\n", rowsAffected)
	}

	// ========================================
	// DELETE Examples
	// ========================================
	fmt.Println("\n\n🗑️  DELETE Examples:")
	fmt.Println(strings.Repeat("-", 60))

	// Example 1: Delete with condition
	fmt.Println("\n1. Delete with condition:")
	rowsAffected, err = norm.Table("users").
		Delete().
		Where("email = $1", "test@example.com").
		Exec(ctx)
	if err != nil {
		log.Printf("  ❌ Error: %v\n", err)
	} else {
		fmt.Printf("  ✅ Deleted %d user(s)\n", rowsAffected)
	}

	// ========================================
	// BULK INSERT Examples
	// ========================================
	fmt.Println("\n\n📦 BULK INSERT Examples:")
	fmt.Println(strings.Repeat("-", 60))

	// Example 1: Struct-based bulk insert (recommended)
	fmt.Println("\n1. Struct-based bulk insert:")

	// Create a slice of users
	bulkUsers := []User{
		{Name: "Alice Williams", Email: "alice@example.com", Username: "alicew"},
		{Name: "Bob Brown", Email: "bob@example.com", Username: "bobb"},
		{Name: "Charlie Davis", Email: "charlie@example.com", Username: "charlied"},
	}

	// Insert all at once
	rowsAffected, err = norm.Table("users").
		BulkInsert(bulkUsers).
		Exec(ctx)
	if err != nil {
		log.Printf("  ❌ Error: %v\n", err)
	} else {
		fmt.Printf("  ✅ Bulk inserted %d user(s)\n", rowsAffected)
	}

	// Example 2: Generate bulk data in a loop
	fmt.Println("\n2. Generate bulk data in loop:")

	generatedUsers := make([]User, 0)
	for i := 1; i <= 5; i++ {
		generatedUsers = append(generatedUsers, User{
			Name:     fmt.Sprintf("Generated User %d", i),
			Email:    fmt.Sprintf("generated%d@example.com", i),
			Username: fmt.Sprintf("gen%d", i),
		})
	}

	rowsAffected, err = norm.Table("users").
		BulkInsert(generatedUsers).
		Exec(ctx)
	if err != nil {
		log.Printf("  ❌ Error: %v\n", err)
	} else {
		fmt.Printf("  ✅ Bulk inserted %d generated user(s)\n", rowsAffected)
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("✅ All query examples completed!")
	fmt.Println(strings.Repeat("=", 60))
}
