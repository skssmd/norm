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
		log.Panicf("  ❌ Error: %v\n", err)
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
		log.Panicf("  ❌ Error: %v\n", err)
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
		log.Panicf("  ❌ Error: %v\n", err)
	} else {
		if count != 6 {
			log.Panicf("  ❌ Expected 6 users, got %d\n", count)
		}
		fmt.Printf("  ✅ Found %d users\n", count)
	}

	// Example 2: SELECT all fields
	fmt.Println("\n2. SELECT all fields (*):")
	count, err = norm.Table("users").
		Select().
		Count(ctx)
	if err != nil {
		log.Panicf("  ❌ Error: %v\n", err)
	} else {
		if count != 6 {
			log.Panicf("  ❌ Expected 6 users, got %d\n", count)
		}
		fmt.Printf("  ✅ Total users: %d\n", count)
	}

	// Example 3: SELECT with WHERE clause
	fmt.Println("\n3. SELECT with WHERE clause:")
	count, err = norm.Table("users").
		Select("fullname", "useremail").
		Where("uname = $1", "alicew").
		Count(ctx)
	if err != nil {
		log.Panicf("  ❌ Error: %v\n", err)
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
		log.Panicf("  ❌ Error: %v\n", err)
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
		log.Panicf("  ❌ Error: %v\n", err)
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
		log.Panicf("  ❌ Error: %v\n", err)
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
		log.Panicf("  ❌ Error: %v\n", err)
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
		log.Panicf("  ❌ Error: %v\n", err)
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
		log.Panicf("  ❌ Error: %v\n", err)
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
		log.Panicf("  ❌ Error: %v\n", err)
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
		log.Panicf("  ❌ Error: %v\n", err)
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
		log.Panicf("  ❌ Error: %v\n", err)
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
		log.Panicf("  ❌ Error: %v\n", err)
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
		log.Panicf("  ❌ Error: %v\n", err)
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
		log.Panicf("  ❌ Error: %v\n", err)
	} else {
		if count > 0 {
			fmt.Printf("  ✅ User 'alicew' still exists\n")
		} else {
			fmt.Printf("  ⚠️  User 'alicew' not found\n")
		}
	}

	// ========================================
	// STEP 6: JOIN QUERIES (Multi-table)
	// ========================================
	fmt.Println("\n\n🔗 STEP 6: JOIN QUERIES")
	fmt.Println(strings.Repeat("-", 60))

	// 1. Populate Orders for Join Test
	fmt.Println("\n1. Populating orders for join test...")
	
	// We need a user ID first. Let's assume ID 1 exists (Alice)
	// But Step 4 deleted users, so we must re-populate first!
	fmt.Println("   (Re-populating users for join test...)")
	// Populate Users
	bulkUsersForJoins := []User{
		{Name: "Alice Williams", Email: "alice@example.com", Username: "alicew"},
		{Name: "Bob Brown", Email: "bob@example.com", Username: "bobb"},
	}
	norm.Table("users").BulkInsert(bulkUsersForJoins).Exec(ctx)

	// IDs are auto-incrementing. If previous users were 1-6, new ones might be 7-8.
	// We need to fetch the actual IDs to use for orders.
	var joinUsers []User
	norm.Table("users").Select().Where("uname IN ($1, $2)", "alicew", "bobb").All(ctx, &joinUsers)
	
	aliceID := uint(1)
	bobID := uint(2)
	
	for _, u := range joinUsers {
		if u.Username == "alicew" {
			aliceID = u.ID
		} else if u.Username == "bobb" {
			bobID = u.ID
		}
	}
	
	bulkOrders := []Order{
		{UserID: aliceID, Total: 99.99, Status: "completed", Notes: "First order"},
		{UserID: aliceID, Total: 149.50, Status: "pending", Notes: "Second order"},
		{UserID: bobID, Total: 29.99, Status: "shipped", Notes: "Bob's order"},
	}

	rowsAffected, err = norm.Table("orders").
		BulkInsert(bulkOrders).
		Exec(ctx)
	if err != nil {
		log.Panicf("  ❌ Error inserting orders: %v\n", err)
	} else {
		fmt.Printf("  ✅ Inserted %d orders\n", rowsAffected)
	}

	// 2. Perform Join Query
	fmt.Println("\n2. Performing JOIN query (Users + Orders)...")
	fmt.Println("   Query: SELECT users.fullname, orders.total FROM users JOIN orders ON users.id = orders.user_id WHERE users.uname = 'alicew'")
	
	// Note: We're not scanning results yet, just verifying execution
	err = norm.Table("users", "id", "orders", "user_id").
		Select("users.fullname", "orders.total").
		Where("users.uname = $1", "alicew").
		All(ctx, nil) // nil dest for now as scanning isn't implemented
	
	if err != nil {
		// We expect an error about scanning not implemented, but the query execution should succeed
		if strings.Contains(err.Error(), "scanning not yet implemented") {
			fmt.Println("  ✅ Join query executed successfully (scanning pending implementation)")
		} else {
			log.Panicf("  ❌ Join query failed: %v\n", err)
		}
	} else {
		fmt.Println("  ✅ Join query executed successfully")
	}

	// 3. Populate Profiles for Skey Test
	fmt.Println("\n3. Populating profiles for Skey test...")
	bulkProfiles := []Profile{
		{UserID: 1, Bio: "Alice's Bio"},
		{UserID: 2, Bio: "Bob's Bio"},
	}
	_, err = norm.Table("profiles").BulkInsert(bulkProfiles).Exec(ctx)
	if err != nil {
		log.Panicf("  ❌ Error inserting profiles: %v\n", err)
	} else {
		fmt.Println("  ✅ Inserted profiles")
	}

	// 4. Perform Skey Join Query (User + Profile)
	// This should use App-Side Join because Profile.UserID is an skey
	fmt.Println("\n4. Performing Skey JOIN query (Users + Profiles)...")
	fmt.Println("   Query: SELECT users.fullname, profiles.bio FROM users JOIN profiles ON users.id = profiles.user_id")
	fmt.Println("   (Should use App-Side Join due to skey)")

	err = norm.Table("users", "id", "profiles", "user_id").
		Select("users.fullname", "profiles.bio").
		Where("users.uname = $1", "alicew").
		All(ctx, nil)
	
	if err != nil {
		log.Panicf("  ❌ Skey Join query failed: %v\n", err)
	} else {
		fmt.Println("  ✅ Skey Join query executed successfully")
	}

	// 5. Perform Non-Native Join Query (User + Analytics)
	// This will only work in Scenario 4 where they are on different shards
	// In other scenarios, they might be co-located (Native Join) or Analytics might not exist
	fmt.Println("\n5. Performing Distributed JOIN query (Users + Analytics)...")
	fmt.Println("   Query: SELECT users.fullname, analytics.event_type FROM users JOIN analytics ON users.id = analytics.user_id")
	
	// Insert some analytics data first
	userID := uint(1)
	norm.Table("analytics").Insert(Analytics{
		UserID: &userID,
		EventType: "login",
		EventData: "{}",
	}).Exec(ctx)

	err = norm.Table("users", "id", "analytics", "user_id").
		Select("users.fullname", "analytics.event_type").
		Where("users.uname = $1", "alicew").
		All(ctx, nil)

	if err != nil {
		log.Panicf("  ❌ Distributed Join query failed: %v\n", err)
	} else {
		fmt.Println("  ✅ Distributed Join query executed successfully")
	}

	// ========================================
	// STEP 7: STRUCT SCANNING
	// ========================================
	fmt.Println("\n\n📥 STEP 7: STRUCT SCANNING")
	fmt.Println(strings.Repeat("-", 60))

	// Re-populate data since previous steps deleted it (or Step 6 added some)
	fmt.Println("Re-populating data for scanning tests...")
	
	// Clean up first to avoid Unique constraint violations from Step 6
	norm.Table("users").Delete().Exec(ctx)
	norm.Table("profiles").Delete().Exec(ctx)
	norm.Table("orders").Delete().Exec(ctx)
	
	// Populate Users
	bulkUsers = []User{
		{Name: "Alice Williams", Email: "alice@example.com", Username: "alicew"},
		{Name: "Bob Brown", Email: "bob@example.com", Username: "bobb"},
	}
	norm.Table("users").BulkInsert(bulkUsers).Exec(ctx)
	
	// Fetch users to get their actual IDs (auto-increment continues from previous inserts)
	var currentUsers []User
	norm.Table("users").Select().All(ctx, &currentUsers)
	
	// Reuse aliceID and bobID variables from STEP 6
	aliceID = 0
	bobID = 0
	for _, u := range currentUsers {
		if u.Username == "alicew" {
			aliceID = u.ID
		} else if u.Username == "bobb" {
			bobID = u.ID
		}
	}
	
	// Populate Profiles with correct user IDs
	if len(currentUsers) > 0 {
		bulkProfiles = []Profile{}
		for _, u := range currentUsers {
			bulkProfiles = append(bulkProfiles, Profile{UserID: int(u.ID), Bio: u.Name + "'s Bio"})
		}
		norm.Table("profiles").BulkInsert(bulkProfiles).Exec(ctx)
	}
	
	// Populate Orders with correct user IDs
	bulkOrders = []Order{
		{UserID: aliceID, Total: 99.99, Status: "completed", Notes: "First order"},
		{UserID: aliceID, Total: 149.50, Status: "pending", Notes: "Second order"},
		{UserID: bobID, Total: 29.99, Status: "shipped", Notes: "Bob's order"},
	}
	norm.Table("orders").BulkInsert(bulkOrders).Exec(ctx)
	fmt.Printf("  ✅ Populated %d users, %d profiles, %d orders\n", len(currentUsers), len(bulkProfiles), len(bulkOrders))

	// 1. Scan single struct
	fmt.Println("\n1. Scanning single user into struct via First()...")
	var u User
	err = norm.Table("users").Select().Where("uname = $1", "alicew").First(ctx, &u)
	if err != nil {
		log.Panicf("  ❌ Error: %v\n", err)
	} else {
		fmt.Printf("  ✅ Scanned User: %s (%s)\n", u.Name, u.Email)
	}

	// 2. Scan slice of structs (limit 1 to avoid huge output)
	fmt.Println("\n2. Scanning multiple users into slice via All()...")
	var users []User
	err = norm.Table("users").Select().Limit(1).All(ctx, &users)
	if err != nil {
		log.Panicf("  ❌ Error: %v\n", err)
	} else {
		fmt.Printf("  ✅ Scanned %d users\n", len(users))
		for _, user := range users {
			fmt.Printf("     - %s\n", user.Name)
		}
	}

	// 3. Scan App-Side Join Results (User struct + Map?)
	// Currently scanning only supports scanning into keys matching struct fields.
	// If we join users + profiles, and select users.fullname, profiles.bio
	// And scan into struct with Fullname and Bio fields
	
	type UserProfile struct {
		Fullname string
		Bio      string
	}
	
	fmt.Println("\n3. Scanning Skey Join result into custom struct...")
	var userProfiles []UserProfile
	err = norm.Table("users", "id", "profiles", "user_id").
		Select("users.fullname", "profiles.bio").
		Where("users.uname = $1", "alicew").
		All(ctx, &userProfiles)
		
	if err != nil {
		log.Panicf("  ❌ Error: %v\n", err)
	} else {
		fmt.Printf("  ✅ Scanned %d joined rows\n", len(userProfiles))
		for _, up := range userProfiles {
			fmt.Printf("     - %s: %s\n", up.Fullname, up.Bio)
		}
	}

	// 4. Scan using Tags and Aliases (for robust mapping)
	fmt.Println("\n4. Scanning with explicit tags and aliasing (User + Profile)...")
	
	type UserAndProfile struct {
		UserName string `norm:"name:fullname"`   // Match column "fullname" or "users.fullname"
		UserBio  string `norm:"name:bio"`        // Match column "bio" or "profiles.bio"
		UID      int    `norm:"name:user_id_alias"` // Verify aliasing support
	}
	
	var userAndProfiles []UserAndProfile
	
	// Use manual select with aliasing for the ID to avoid ambiguity/conflict if needed
	// and to prove we can map to arbitrary struct fields
	err = norm.Table("users", "id", "profiles", "user_id").
		Select("users.fullname", "profiles.bio", "users.id as user_id_alias").
		Where("users.uname = $1", "alicew").
		All(ctx, &userAndProfiles)
		
	if err != nil {
		log.Panicf("  ❌ Error: %v\n", err)
	} else {
		if len(userAndProfiles) == 0 {
			log.Panicf("  ❌ Expected > 0 results for User+Profile scan, got 0\n")
		}
		fmt.Printf("  ✅ Scanned %d rows into UserAndProfile\n", len(userAndProfiles))
		for _, item := range userAndProfiles {
			fmt.Printf("     - Name: %s, Bio: %s, ID: %d\n", item.UserName, item.UserBio, item.UID)
		}
	}

	// 5. Scan User + Orders (Native/Standard Join)
	fmt.Println("\n5. Scanning User + Orders (Native/Standard Join)...")
	type UserOrder struct {
		UserName   string  `norm:"name:fullname"`
		OrderTotal float64 `norm:"name:total"`
	}
	var userOrders []UserOrder
	
	// Alice has 2 orders
	err = norm.Table("users", "id", "orders", "user_id").
		Select("users.fullname", "orders.total").
		Where("users.uname = $1", "alicew").
		All(ctx, &userOrders)

	if err != nil {
		log.Panicf("  ❌ Error: %v\n", err)
	} else {
		if len(userOrders) != 2 {
			log.Panicf("  ❌ Expected 2 orders for Alice, got %d\n", len(userOrders))
		}
		fmt.Printf("  ✅ Scanned %d rows into UserOrder\n", len(userOrders))
		for _, item := range userOrders {
			fmt.Printf("     - Name: %s, Total: %.2f\n", item.UserName, item.OrderTotal)
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("✅ All query examples completed!")
	fmt.Println(strings.Repeat("=", 60))
}
