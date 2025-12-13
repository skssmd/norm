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
	fmt.Println("QUERY BUILDER EXAMPLES")
	fmt.Println(strings.Repeat("=", 60) + "\n")

	// ========================================
	// STEP 1: POPULATE DATABASE (Bulk Inserts)
	// ========================================
	fmt.Println("📦 STEP 1: POPULATING DATABASE")
	fmt.Println(strings.Repeat("-", 60))

	// Bulk insert users
	fmt.Println("\n1. Bulk inserting users...")
	bulkUsers := []User{
		{Name: "Alice Williams", Email: "alice@example.com", Username: "alicew"},
		{Name: "Bob Brown", Email: "bob@example.com", Username: "bobb"},
		{Name: "Charlie Davis", Email: "charlie@example.com", Username: "charlied"},
		{Name: "Diana Evans", Email: "diana@example.com", Username: "dianae"},
		{Name: "Eve Foster", Email: "eve@example.com", Username: "evef"},
	}

	rowsAffected, err := norm.Table("users").
		BulkInsert(bulkUsers).
		Exec(ctx)
	if err != nil {
		log.Printf("  ❌ Error: %v\n", err)
	} else {
		fmt.Printf("  ✅ Bulk inserted %d user(s)\n", rowsAffected)
	}

	// Insert individual users with different patterns
	fmt.Println("\n2. Inserting individual users...")
	age := uint(29)
	rowsAffected, err = norm.Table(User{
		Name:  "Frank Green",
		Email: "frank@example.com",
		Age:   &age,
		Username: "frankg",
	}).Insert().Exec(ctx)
	if err != nil {
		log.Printf("  ❌ Error: %v\n", err)
	} else {
		fmt.Printf("  ✅ Inserted %d user(s)\n", rowsAffected)
	}

	// ========================================
	// STEP 2: SELECT QUERIES (Read Data)
	// ========================================
	fmt.Println("\n\n📖 STEP 2: SELECT QUERIES")
	fmt.Println(strings.Repeat("-", 60))

	// Example 1: Simple SELECT with specific columns
	fmt.Println("\n1. Simple SELECT (specific columns):")
	count, err := norm.Table("users").
		Select("id", "fullname", "useremail").
		Count(ctx)
	if err != nil {
		log.Printf("  ❌ Error: %v\n", err)
	} else {
		fmt.Printf("  ✅ Found %d users\n", count)
	}

	// Example 2: SELECT all fields
	fmt.Println("\n2. SELECT all fields (*):")
	count, err = norm.Table("users").
		Select().
		Count(ctx)
	if err != nil {
		log.Printf("  ❌ Error: %v\n", err)
	} else {
		fmt.Printf("  ✅ Total users: %d\n", count)
	}

	// Example 3: SELECT with WHERE clause
	fmt.Println("\n3. SELECT with WHERE clause:")
	count, err = norm.Table("users").
		Select("fullname", "useremail").
		Where("uname = $1", "alicew").
		Count(ctx)
	if err != nil {
		log.Printf("  ❌ Error: %v\n", err)
	} else {
		fmt.Printf("  ✅ Found %d user(s) with username 'alicew'\n", count)
	}

	// Example 4: SELECT with multiple WHERE conditions
	fmt.Println("\n4. SELECT with multiple WHERE conditions:")
	count, err = norm.Table("users").
		Select("fullname", "useremail").
		Where("created_at > $1 AND uname LIKE $2", time.Now().AddDate(0, -1, 0), "%e%").
		Count(ctx)
	if err != nil {
		log.Printf("  ❌ Error: %v\n", err)
	} else {
		fmt.Printf("  ✅ Found %d recent user(s) with 'e' in username\n", count)
	}

	// Example 5: SELECT with ORDER BY
	fmt.Println("\n5. SELECT with ORDER BY:")
	count, err = norm.Table("users").
		Select("id", "fullname", "useremail").
		OrderBy("fullname ASC").
		Count(ctx)
	if err != nil {
		log.Printf("  ❌ Error: %v\n", err)
	} else {
		fmt.Printf("  ✅ Found %d users (ordered by name)\n", count)
	}

	// Example 6: SELECT with LIMIT and OFFSET (pagination)
	fmt.Println("\n6. SELECT with pagination (LIMIT 3, OFFSET 0):")
	count, err = norm.Table("users").
		Select("id", "fullname", "useremail").
		OrderBy("created_at DESC").
		Pagination(3, 0).
		Count(ctx)
	if err != nil {
		log.Printf("  ❌ Error: %v\n", err)
	} else {
		fmt.Printf("  ✅ Page 1: %d users\n", count)
	}

	// ========================================
	// STEP 3: UPDATE QUERIES (Modify Data)
	// ========================================
	fmt.Println("\n\n✏️  STEP 3: UPDATE QUERIES")
	fmt.Println(strings.Repeat("-", 60))

	// Example 1: Update single field
	fmt.Println("\n1. Update single field:")
	rowsAffected, err = norm.Table("users").
		Update("fullname", "Alice Williams Updated").
		Where("uname = $1", "alicew").
		Exec(ctx)
	if err != nil {
		log.Printf("  ❌ Error: %v\n", err)
	} else {
		fmt.Printf("  ✅ Updated %d user(s)\n", rowsAffected)
	}

	// Example 2: Update multiple fields (pair-based)
	fmt.Println("\n2. Update multiple fields (pair-based):")
	rowsAffected, err = norm.Table("users").
		Update("fullname", "Bob Brown Jr.", "useremail", "bob.jr@example.com").
		Where("uname = $1", "bobb").
		Exec(ctx)
	if err != nil {
		log.Printf("  ❌ Error: %v\n", err)
	} else {
		fmt.Printf("  ✅ Updated %d user(s)\n", rowsAffected)
	}

	// Example 3: Struct-based update (only non-zero fields)
	fmt.Println("\n3. Struct-based update (non-zero fields only):")
	rowsAffected, err = norm.Table(User{
		Name: "Charlie Davis Modified",
		// Email is empty, will be ignored (keeps old value)
	}).Update().Where("uname = $1", "charlied").Exec(ctx)
	if err != nil {
		log.Printf("  ❌ Error: %v\n", err)
	} else {
		fmt.Printf("  ✅ Updated %d user(s)\n", rowsAffected)
	}

	// Example 4: Update with complex WHERE clause
	fmt.Println("\n4. Update with complex WHERE clause:")
	rowsAffected, err = norm.Table("users").
		Update("fullname", "Updated User").
		Where("created_at > $1 AND uname LIKE $2", time.Now().AddDate(0, -1, 0), "%e%").
		Exec(ctx)
	if err != nil {
		log.Printf("  ❌ Error: %v\n", err)
	} else {
		fmt.Printf("  ✅ Updated %d user(s) matching criteria\n", rowsAffected)
	}

	// ========================================
	// STEP 4: DELETE QUERIES (Remove Data)
	// ========================================
	fmt.Println("\n\n🗑️  STEP 4: DELETE QUERIES")
	fmt.Println(strings.Repeat("-", 60))

	// Example 1: Delete with simple condition
	fmt.Println("\n1. Delete with simple condition:")
	rowsAffected, err = norm.Table("users").
		Delete().
		Where("uname = $1", "evef").
		Exec(ctx)
	if err != nil {
		log.Printf("  ❌ Error: %v\n", err)
	} else {
		fmt.Printf("  ✅ Deleted %d user(s)\n", rowsAffected)
	}

	// Example 2: Delete with complex condition
	fmt.Println("\n2. Delete with complex condition:")
	rowsAffected, err = norm.Table("users").
		Delete().
		Where("useremail LIKE $1 AND created_at < $2", "%@example.com", time.Now().Add(time.Hour)).
		Exec(ctx)
	if err != nil {
		log.Printf("  ❌ Error: %v\n", err)
	} else {
		fmt.Printf("  ✅ Deleted %d user(s) matching criteria\n", rowsAffected)
	}

	// ========================================
	// STEP 5: VERIFY FINAL STATE (Select Again)
	// ========================================
	fmt.Println("\n\n🔍 STEP 5: VERIFY FINAL STATE")
	fmt.Println(strings.Repeat("-", 60))

	// Count remaining users
	fmt.Println("\n1. Count remaining users:")
	count, err = norm.Table("users").
		Select().
		Count(ctx)
	if err != nil {
		log.Printf("  ❌ Error: %v\n", err)
	} else {
		fmt.Printf("  ✅ Total remaining users: %d\n", count)
	}

	// List all remaining users
	fmt.Println("\n2. List all remaining users (ordered by name):")
	count, err = norm.Table("users").
		Select("id", "fullname", "useremail", "uname").
		OrderBy("fullname ASC").
		Count(ctx)
	if err != nil {
		log.Printf("  ❌ Error: %v\n", err)
	} else {
		fmt.Printf("  ✅ Listed %d user(s)\n", count)
	}

	// Check specific user still exists
	fmt.Println("\n3. Verify specific user exists (Alice):")
	count, err = norm.Table("users").
		Select("fullname", "useremail").
		Where("uname = $1", "alicew").
		Count(ctx)
	if err != nil {
		log.Printf("  ❌ Error: %v\n", err)
	} else {
		if count > 0 {
			fmt.Printf("  ✅ User 'alicew' still exists\n")
		} else {
			fmt.Printf("  ⚠️  User 'alicew' not found\n")
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("✅ All query examples completed!")
	fmt.Println(strings.Repeat("=", 60))
}
