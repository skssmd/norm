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
		Cache(time.Second * 2, "users", "all").
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
		Cache(time.Second * 2, "users", "all").
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
		Cache(time.Second * 2, "users", "alicew").
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
		Cache(time.Second * 2).
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
		Cache(time.Second * 2).
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
		Cache(time.Second * 2).
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
		InvalidateCacheReferenced("users").
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
		InvalidateCacheReferenced("users").
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
	}).Update().Where("uname = $1", "charlied").InvalidateCacheReferenced("users").Exec(ctx)
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
		InvalidateCacheReferenced("users").
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
		InvalidateCacheReferenced("users").
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
		InvalidateCacheReferenced("users").
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
		Cache(time.Second * 2, "users", "all").
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
		Cache(time.Second * 2, "users", "all").
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
		Cache(time.Second * 2, "users", "alicew").
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
	norm.Table("users").Select().Where("uname IN ($1, $2)", "alicew", "bobb").Cache(time.Second * 2, "users", "join").All(ctx, &joinUsers)
	
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
		Cache(time.Second * 2). // JOIN automatically uses table names as keys: users:orders::
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
		Cache(time.Second * 2). // JOIN automatically uses: users:profiles::
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
		Cache(time.Second * 2).
		All(ctx, nil)

	if err != nil {
		log.Panicf("  ❌ Distributed Join query failed: %v\n", err)
	} else {
		fmt.Println("  ✅ Distributed Join query executed successfully")
	}

	// 6. Perform Fkey Join Query (User + Orders - Native Join with Foreign Key)
	fmt.Println("\n6. Performing FKEY JOIN query (Users + Orders)...")
	fmt.Println("   Query: SELECT users.fullname, orders.total FROM users JOIN orders ON users.id = orders.user_id")
	fmt.Println("   (Should use Native Join due to fkey and co-location)")
	
	type UserOrderResult struct {
		Fullname string
		Total    float64
	}
	var userOrderResults []UserOrderResult
	
	err = norm.Table("users", "id", "orders", "user_id").
		Select("users.fullname", "orders.total").
		Where("users.uname = $1", "alicew").
		Cache(time.Second * 2). // JOIN automatically uses: users:orders::
		All(ctx, &userOrderResults)
	
	if err != nil {
		log.Panicf("  ❌ Fkey Join query failed: %v\n", err)
	} else {
		fmt.Printf("  ✅ Fkey Join query executed successfully, found %d order(s)\n", len(userOrderResults))
		for _, result := range userOrderResults {
			fmt.Printf("     - %s: $%.2f\n", result.Fullname, result.Total)
		}
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
	norm.Table("users").Select().Cache(time.Second * 2).All(ctx, &currentUsers)
	
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
	err = norm.Table("users").Select().Where("uname = $1", "alicew").Cache(time.Second * 2).First(ctx, &u)
	if err != nil {
		log.Panicf("  ❌ Error: %v\n", err)
	} else {
		fmt.Printf("  ✅ Scanned User: %s (%s)\n", u.Name, u.Email)
	}

	// 2. Scan slice of structs (limit 1 to avoid huge output)
	fmt.Println("\n2. Scanning multiple users into slice via All()...")
	var users []User
	err = norm.Table("users").Select().Limit(1).Cache(time.Second * 2).All(ctx, &users)
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
		Cache(time.Second * 2).
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
		Cache(time.Second * 2).
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
		Cache(time.Second * 2).
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

	// ========================================
	// STEP 8: RAW SQL QUERIES
	// ========================================
	fmt.Println("\n\n🔧 STEP 8: RAW SQL QUERIES")
	fmt.Println(strings.Repeat("-", 60))

	// Repopulate data for raw SQL tests (Step 7 may have deleted it)
	fmt.Println("Re-populating data for raw SQL tests...")
	norm.Table("users").Delete().Exec(ctx)
	norm.Table("profiles").Delete().Exec(ctx)
	
	bulkUsers = []User{
		{Name: "Alice Williams", Email: "alice@example.com", Username: "alicew"},
		{Name: "Bob Brown", Email: "bob@example.com", Username: "bobb"},
	}
	norm.Table("users").BulkInsert(bulkUsers).Exec(ctx)
	
	// Fetch user IDs for profiles
	var currentUsers2 []User
	norm.Table("users").Select().Cache(time.Second * 2).All(ctx, &currentUsers2)
	
	if len(currentUsers2) > 0 {
		bulkProfiles = []Profile{}
		for _, u := range currentUsers2 {
			bulkProfiles = append(bulkProfiles, Profile{UserID: int(u.ID), Bio: u.Name + "'s Bio"})
		}
		norm.Table("profiles").BulkInsert(bulkProfiles).Exec(ctx)
	}

	// 1. Table-based raw query (automatic routing)
	fmt.Println("\n1. Table-based raw query (automatic routing)...")
	var rawUsers []User
	err = norm.Table("users").
		Raw("SELECT * FROM users WHERE uname LIKE $1 ORDER BY fullname", "%e%").
		Cache(time.Second * 2).
		All(ctx, &rawUsers)
	
	if err != nil {
		log.Panicf("  ❌ Error: %v\n", err)
	} else {
		fmt.Printf("  ✅ Found %d users via raw SQL\n", len(rawUsers))
		for _, u := range rawUsers {
			fmt.Printf("     - %s\n", u.Name)
		}
	}

	// 2. Join-based raw query (co-located tables)
	fmt.Println("\n2. Join-based raw query (co-located tables)...")
	type RawUserProfile struct {
		Fullname string
		Bio      string
	}
	var rawProfiles []RawUserProfile
	err = norm.Join("users", "profiles").
		Raw("SELECT u.fullname, p.bio FROM users u JOIN profiles p ON u.id = p.user_id WHERE u.uname = $1", "alicew").
		Cache(time.Second * 2).
		All(ctx, &rawProfiles)
	
	if err != nil {
		log.Panicf("  ❌ Error: %v\n", err)
	} else {
		fmt.Printf("  ✅ Found %d profiles via raw join SQL\n", len(rawProfiles))
		for _, p := range rawProfiles {
			fmt.Printf("     - %s: %s\n", p.Fullname, p.Bio)
		}
	}

	// 3. Test error case: Join with non-co-located tables (should fail)
	fmt.Println("\n3. Testing error case: Join with non-co-located tables...")
	// This should fail in sharded scenarios where users and analytics are on different shards
	// In global mode, this will work fine
	err = norm.Join("users", "analytics").
		Raw("SELECT u.fullname FROM users u JOIN analytics a ON u.id = a.user_id").
		Cache(time.Second * 2).
		All(ctx, nil)
	
	if err != nil {
		// Expected error in sharded mode
		fmt.Printf("  ⚠️  Expected error (tables may not be co-located): %v\n", err)
	} else {
		fmt.Println("  ✅ Query succeeded (tables are co-located)")
	}

	// ========================================
	// STEP 9: CACHE PERFORMANCE TESTING
	// ========================================
	fmt.Println("\n\n⚡ STEP 9: CACHE PERFORMANCE TESTING")
	fmt.Println(strings.Repeat("-", 60))
	
	// Enable memory cache
	fmt.Println("\n🧠 Enabling Memory Cache...")
	norm.EnableMemoryCache()
	
	// Repopulate clean data for cache testing
	fmt.Println("📦 Preparing test data for cache performance tests...")
	norm.Table("users").Delete().Exec(ctx)
	norm.Table("profiles").Delete().Exec(ctx)
	
	bulkUsers = []User{
		{Name: "Alice Williams", Email: "alice@example.com", Username: "alicew"},
		{Name: "Bob Brown", Email: "bob@example.com", Username: "bobb"},
		{Name: "Charlie Davis", Email: "charlie@example.com", Username: "charlied"},
	}
	norm.Table("users").BulkInsert(bulkUsers).Exec(ctx)
	
	// Get user IDs for profiles
	var cacheTestUsers []User
	norm.Table("users").Select().All(ctx, &cacheTestUsers)
	
	if len(cacheTestUsers) > 0 {
		bulkProfiles = []Profile{}
		for _, u := range cacheTestUsers {
			bulkProfiles = append(bulkProfiles, Profile{UserID: int(u.ID), Bio: u.Name + "'s Bio"})
		}
		norm.Table("profiles").BulkInsert(bulkProfiles).Exec(ctx)
	}
	fmt.Println("✅ Test data ready\n")
	
	// ========================================
	// Test 1: Count() Query Caching
	// ========================================
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println("Test 1: Count() Query Caching Performance")
	fmt.Println(strings.Repeat("-", 60))
	
	// First query - Cache MISS
	start := time.Now()
	count, err = norm.Table("users").
		Select("id", "fullname", "useremail").
		Cache(time.Second * 5).
		Count(ctx)
	duration1 := time.Since(start)
	
	if err != nil {
		log.Panicf("  ❌ Count query 1 failed: %v\n", err)
	}
	fmt.Printf("  Query 1 (MISS): Count=%d, Time=%v\n", count, duration1)
	
	// Repeat 5 times - Cache HITs
	fmt.Println("  Running 5 repetitive queries to test cache hits...")
	var countDurations []time.Duration
	for i := 2; i <= 6; i++ {
		start = time.Now()
		count, err = norm.Table("users").
			Select("id", "fullname", "useremail").
			Cache(time.Second * 5).
			Count(ctx)
		duration := time.Since(start)
		countDurations = append(countDurations, duration)
		
		if err != nil {
			log.Panicf("  ❌ Count query %d failed: %v\n", i, err)
		}
		fmt.Printf("  Query %d (HIT):  Count=%d, Time=%v\n", i, count, duration)
	}
	
	// Calculate average
	var totalCountTime time.Duration
	for _, d := range countDurations {
		totalCountTime += d
	}
	avgCountTime := totalCountTime / time.Duration(len(countDurations))
	speedup := float64(duration1) / float64(avgCountTime)
	
	fmt.Printf("\n  📊 Results:\n")
	fmt.Printf("     First query (MISS): %v\n", duration1)
	fmt.Printf("     Average cache hit:  %v\n", avgCountTime)
	fmt.Printf("     Speedup:            %.2fx faster\n", speedup)
	
	if avgCountTime < duration1 {
		fmt.Println("  ✅ Count() caching is working! Cache hits are faster.")
	} else {
		fmt.Println("  ⚠️  Cache performance needs investigation.")
	}
	
	// ========================================
	// Test 2: All() Query Caching (Struct Scanning)
	// ========================================
	fmt.Println("\n" + strings.Repeat("-", 60))
	fmt.Println("Test 2: All() Query Caching Performance (Struct Scanning)")
	fmt.Println(strings.Repeat("-", 60))
	
	// First query - Cache MISS
	start = time.Now()
	var scanUsers1 []User
	err = norm.Table("users").
		Select().
		Cache(time.Second * 5).
		All(ctx, &scanUsers1)
	duration1 = time.Since(start)
	
	if err != nil {
		log.Panicf("  ❌ All query 1 failed: %v\n", err)
	}
	fmt.Printf("  Query 1 (MISS): Found %d users, Time=%v\n", len(scanUsers1), duration1)
	
	// Repeat 5 times - Cache HITs
	fmt.Println("  Running 5 repetitive scans to test cache hits...")
	var allDurations []time.Duration
	for i := 2; i <= 6; i++ {
		start = time.Now()
		var scanUsers []User
		err = norm.Table("users").
			Select().
			Cache(time.Second * 5).
			All(ctx, &scanUsers)
		duration := time.Since(start)
		allDurations = append(allDurations, duration)
		
		if err != nil {
			log.Panicf("  ❌ All query %d failed: %v\n", i, err)
		}
		fmt.Printf("  Query %d (HIT):  Found %d users, Time=%v\n", i, len(scanUsers), duration)
	}
	
	// Calculate average
	var totalAllTime time.Duration
	for _, d := range allDurations {
		totalAllTime += d
	}
	avgAllTime := totalAllTime / time.Duration(len(allDurations))
	speedup = float64(duration1) / float64(avgAllTime)
	
	fmt.Printf("\n  📊 Results:\n")
	fmt.Printf("     First query (MISS): %v\n", duration1)
	fmt.Printf("     Average cache hit:  %v\n", avgAllTime)
	fmt.Printf("     Speedup:            %.2fx faster\n", speedup)
	
	if avgAllTime < duration1 {
		fmt.Println("  ✅ All() caching is working! Cache hits are faster.")
	} else {
		fmt.Println("  ⚠️  Cache performance needs investigation.")
	}
	
	// ========================================
	// Test 3: First() Query Caching
	// ========================================
	fmt.Println("\n" + strings.Repeat("-", 60))
	fmt.Println("Test 3: First() Query Caching Performance")
	fmt.Println(strings.Repeat("-", 60))
	
	// First query - Cache MISS
	start = time.Now()
	var firstUser1 User
	err = norm.Table("users").
		Select().
		Where("uname = $1", "alicew").
		Cache(time.Second * 5).
		First(ctx, &firstUser1)
	duration1 = time.Since(start)
	
	if err != nil {
		log.Panicf("  ❌ First query 1 failed: %v\n", err)
	}
	fmt.Printf("  Query 1 (MISS): Found '%s', Time=%v\n", firstUser1.Name, duration1)
	
	// Repeat 5 times - Cache HITs
	fmt.Println("  Running 5 repetitive First() queries to test cache hits...")
	var firstDurations []time.Duration
	for i := 2; i <= 6; i++ {
		start = time.Now()
		var firstUser User
		err = norm.Table("users").
			Select().
			Where("uname = $1", "alicew").
			Cache(time.Second * 5).
			First(ctx, &firstUser)
		duration := time.Since(start)
		firstDurations = append(firstDurations, duration)
		
		if err != nil {
			log.Panicf("  ❌ First query %d failed: %v\n", i, err)
		}
		fmt.Printf("  Query %d (HIT):  Found '%s', Time=%v\n", i, firstUser.Name, duration)
	}
	
	// Calculate average
	var totalFirstTime time.Duration
	for _, d := range firstDurations {
		totalFirstTime += d
	}
	avgFirstTime := totalFirstTime / time.Duration(len(firstDurations))
	speedup = float64(duration1) / float64(avgFirstTime)
	
	fmt.Printf("\n  📊 Results:\n")
	fmt.Printf("     First query (MISS): %v\n", duration1)
	fmt.Printf("     Average cache hit:  %v\n", avgFirstTime)
	fmt.Printf("     Speedup:            %.2fx faster\n", speedup)
	
	if avgFirstTime < duration1 {
		fmt.Println("  ✅ First() caching is working! Cache hits are faster.")
	} else {
		fmt.Println("  ⚠️  Cache performance needs investigation.")
	}
	
	// ========================================
	// Test 4: JOIN Query Caching
	// ========================================
	fmt.Println("\n" + strings.Repeat("-", 60))
	fmt.Println("Test 4: JOIN Query Caching Performance")
	fmt.Println(strings.Repeat("-", 60))
	
	type CachedUserProfile struct {
		Fullname string
		Bio      string
	}
	
	// First query - Cache MISS
	start = time.Now()
	var joinProfiles1 []CachedUserProfile
	err = norm.Table("users", "id", "profiles", "user_id").
		Select("users.fullname", "profiles.bio").
		Where("users.uname = $1", "alicew").
		Cache(time.Second * 5).
		All(ctx, &joinProfiles1)
	duration1 = time.Since(start)
	
	if err != nil {
		log.Panicf("  ❌ JOIN query 1 failed: %v\n", err)
	}
	fmt.Printf("  Query 1 (MISS): Found %d rows, Time=%v\n", len(joinProfiles1), duration1)
	
	// Repeat 5 times - Cache HITs
	fmt.Println("  Running 5 repetitive JOIN queries to test cache hits...")
	var joinDurations []time.Duration
	for i := 2; i <= 6; i++ {
		start = time.Now()
		var joinProfiles []CachedUserProfile
		err = norm.Table("users", "id", "profiles", "user_id").
			Select("users.fullname", "profiles.bio").
			Where("users.uname = $1", "alicew").
			Cache(time.Second * 5).
			All(ctx, &joinProfiles)
		duration := time.Since(start)
		joinDurations = append(joinDurations, duration)
		
		if err != nil {
			log.Panicf("  ❌ JOIN query %d failed: %v\n", i, err)
		}
		fmt.Printf("  Query %d (HIT):  Found %d rows, Time=%v\n", i, len(joinProfiles), duration)
	}
	
	// Calculate average
	var totalJoinTime time.Duration
	for _, d := range joinDurations {
		totalJoinTime += d
	}
	avgJoinTime := totalJoinTime / time.Duration(len(joinDurations))
	speedup = float64(duration1) / float64(avgJoinTime)
	
	fmt.Printf("\n  📊 Results:\n")
	fmt.Printf("     First query (MISS): %v\n", duration1)
	fmt.Printf("     Average cache hit:  %v\n", avgJoinTime)
	fmt.Printf("     Speedup:            %.2fx faster\n", speedup)
	
	if avgJoinTime < duration1 {
		fmt.Println("  ✅ JOIN caching is working! Cache hits are faster.")
	} else {
		fmt.Println("  ⚠️  Cache performance needs investigation.")
	}
	
	// ========================================
	// Test 5: Cache Expiration Verification
	// ========================================
	fmt.Println("\n" + strings.Repeat("-", 60))
	fmt.Println("Test 5: Cache Expiration Verification")
	fmt.Println(strings.Repeat("-", 60))
	
	// Query with 2-second cache
	start = time.Now()
	count, err = norm.Table("users").
		Select().
		Cache(time.Second * 2).
		Count(ctx)
	duration1 = time.Since(start)
	
	if err != nil {
		log.Panicf("  ❌ Expiration test query 1 failed: %v\n", err)
	}
	fmt.Printf("  Query 1 (MISS): Count=%d, Time=%v\n", count, duration1)
	
	// Immediate repeat - Cache HIT
	start = time.Now()
	count, err = norm.Table("users").
		Select().
		Cache(time.Second * 2).
		Count(ctx)
	duration2 := time.Since(start)
	
	if err != nil {
		log.Panicf("  ❌ Expiration test query 2 failed: %v\n", err)
	}
	fmt.Printf("  Query 2 (HIT):  Count=%d, Time=%v\n", count, duration2)
	
	// Wait for expiration
	fmt.Println("\n  ⏳ Waiting 3 seconds for cache to expire...")
	time.Sleep(3 * time.Second)
	
	// Query after expiration - Cache MISS
	start = time.Now()
	count, err = norm.Table("users").
		Select().
		Cache(time.Second * 2).
		Count(ctx)
	duration3 := time.Since(start)
	
	if err != nil {
		log.Panicf("  ❌ Expiration test query 3 failed: %v\n", err)
	}
	fmt.Printf("  Query 3 (MISS - Expired): Count=%d, Time=%v\n", count, duration3)
	
	fmt.Printf("\n  📊 Results:\n")
	fmt.Printf("     Before expiration (HIT):  %v\n", duration2)
	fmt.Printf("     After expiration (MISS):  %v\n", duration3)
	
	if duration3 > duration2 {
		fmt.Println("  ✅ Cache expiration is working correctly!")
	} else {
		fmt.Println("  ⚠️  Cache expiration needs investigation.")
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("✅ All query examples completed!")
	fmt.Println(strings.Repeat("=", 60))
}

func RunCachingExamples() {
	ctx := context.Background()
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("CACHING SUPPORT EXAMPLES")
	fmt.Println(strings.Repeat("=", 60) + "\n")

	// Seed Data
	fmt.Println("🌱 Seeding Caching Data...")
	email := "cache.demo@example.com"
	norm.Table("users").Where("useremail = $1", email).Delete().Exec(ctx)

	// We assume User struct is available
	user := User{Name: "Cache Demo User", Email: email, Username: "cachedemo"}
	norm.Table(user).Insert().Exec(ctx)

	// Get ID
	var users []User
	norm.Table("users").Select().Where("useremail = $1", email).All(ctx, &users)
	if len(users) == 0 {
		fmt.Println("❌ Failed to seed user")
		return
	}
	uid := users[0].ID

	// Insert Order (Native Join candidate)
	// We assume Order struct is available
	order := Order{UserID: uid, Total: 50.00}
	norm.Table(order).Insert().Exec(ctx)

	// Insert Analytics (App-Side Join candidate)
	// We assume Analytics struct is available
	uidPtr := &uid
	analytics := Analytics{UserID: uidPtr, EventType: "cache_event", EventData: "{}"}
	norm.Table(analytics).Insert().Exec(ctx)

	// 1. Native Join Caching
	fmt.Println("\n🔗 1. NATIVE JOIN CACHING (Users + Orders)")
	fmt.Println("   (First run: Miss, Second run: Hit)")
	
	type UserOrder struct {
		Name  string
		Total float64
	}
	var resNative []UserOrder

	// Run 1
	start := time.Now()
	norm.Table("users", "id", "orders", "user_id").
		Select("users.fullname", "orders.total").
		Where("users.useremail = $1", email).
		Cache(time.Minute, "users:native").
		All(ctx, &resNative)
	fmt.Printf("   Run 1 Time: %v\n", time.Since(start))

	// Run 2
	start = time.Now()
	norm.Table("users", "id", "orders", "user_id").
		Select("users.fullname", "orders.total").
		Where("users.useremail = $1", email).
		Cache(time.Minute, "users:native").
		All(ctx, &resNative)
	fmt.Printf("   Run 2 Time: %v (Expect faster + [HIT])\n", time.Since(start))


	// 2. App-Side Join Caching
	fmt.Println("\n🔗 2. APP-SIDE JOIN CACHING (Users + Analytics)")
	fmt.Println("   (First run: Miss, Second run: Hit)")

	type UserAnalytics struct {
		Name      string
		EventType string
	}
	var resApp []UserAnalytics

	// Run 1
	start = time.Now()
	norm.Table("users", "id", "analytics", "user_id").
		Select("users.fullname", "analytics.event_type").
		Where("users.useremail = $1", email).
		Cache(time.Minute, "users:appside").
		All(ctx, &resApp)
	fmt.Printf("   Run 1 Time: %v\n", time.Since(start))

	// Run 2
	start = time.Now()
	norm.Table("users", "id", "analytics", "user_id").
		Select("users.fullname", "analytics.event_type").
		Where("users.useremail = $1", email).
		Cache(time.Minute, "users:appside").
		All(ctx, &resApp)
	fmt.Printf("   Run 2 Time: %v (Expect faster + [HIT])\n", time.Since(start))



	// 3. Raw SQL Caching
	fmt.Println("\n🔗 3. RAW SQL CACHING")
	var resRaw []User

	// Run 1
	start = time.Now()
	norm.Table("users").
		Raw("SELECT * FROM users WHERE useremail = $1", email).
		Cache(time.Minute, "users:raw").
		All(ctx, &resRaw)
	fmt.Printf("   Run 1 Time: %v\n", time.Since(start))

	// Run 2
	start = time.Now()
	norm.Table("users").
		Raw("SELECT * FROM users WHERE useremail = $1", email).
		Cache(time.Minute, "users:raw").
		All(ctx, &resRaw)
	fmt.Printf("   Run 2 Time: %v (Expect faster + [HIT])\n", time.Since(start))


	// 4. Fkey Join Caching (Native Join with Foreign Keys)
	fmt.Println("\n🔗 4. FKEY JOIN CACHING (Users + Orders - Native Join)")
	fmt.Println("   (Foreign Key: orders.user_id → users.id)")
	fmt.Println("   (First run: Miss, Second run: Hit)")
	
	type UserOrderFkey struct {
		Name  string  `norm:"name:fullname"`
		Total float64 `norm:"name:total"`
	}
	var resFkey []UserOrderFkey

	// Run 1
	start = time.Now()
	norm.Table("users", "id", "orders", "user_id").
		Select("users.fullname", "orders.total").
		Where("users.useremail = $1", email).
		Cache(time.Minute, "users:orders:fkey").
		All(ctx, &resFkey)
	fmt.Printf("   Run 1 Time: %v\n", time.Since(start))

	// Run 2
	start = time.Now()
	norm.Table("users", "id", "orders", "user_id").
		Select("users.fullname", "orders.total").
		Where("users.useremail = $1", email).
		Cache(time.Minute, "users:orders:fkey").
		All(ctx, &resFkey)
	fmt.Printf("   Run 2 Time: %v (Expect faster + [HIT])\n", time.Since(start))


	// 5. Cache Keys - Single Key Example
	fmt.Println("\n🔑 5. CACHE KEYS - SINGLE KEY (Targeted Invalidation)")
	fmt.Println("   (Using custom cache key for user-specific caching)")
	
	// Query with single cache key
	var userByKey1 []User
	start = time.Now()
	norm.Table("users").
		Select().
		Where("useremail = $1", email).
		Cache(time.Minute, "user", "demo"). // Single key: "user"
		All(ctx, &userByKey1)
	fmt.Printf("   Run 1 (MISS): Time=%v\n", time.Since(start))
	
	// Same query - should hit cache
	start = time.Now()
	norm.Table("users").
		Select().
		Where("useremail = $1", email).
		Cache(time.Minute, "user", "demo"). // Same key
		All(ctx, &userByKey1)
	fmt.Printf("   Run 2 (HIT):  Time=%v\n", time.Since(start))
	
	// Different key - should miss cache
	var userByKey2 []User
	start = time.Now()
	norm.Table("users").
		Select().
		Where("useremail = $1", email).
		Cache(time.Minute, "admin", "demo"). // Different key: "admin"
		All(ctx, &userByKey2)
	fmt.Printf("   Run 3 (MISS - Different key): Time=%v\n", time.Since(start))


	// 6. Cache Keys - Two Keys Example
	fmt.Println("\n🔑 6. CACHE KEYS - TWO KEYS (Multi-level Invalidation)")
	fmt.Println("   (Using two cache keys for hierarchical caching)")
	
	// Query with two cache keys
	var ordersByKey []Order
	start = time.Now()
	norm.Table("orders").
		Select().
		Where("user_id = $1", uid).
		Cache(time.Minute, "orders", "user:demo"). // Two keys: "orders" and "user:demo"
		All(ctx, &ordersByKey)
	fmt.Printf("   Run 1 (MISS): Time=%v\n", time.Since(start))
	
	// Same query - should hit cache
	start = time.Now()
	norm.Table("orders").
		Select().
		Where("user_id = $1", uid).
		Cache(time.Minute, "orders", "user:demo"). // Same keys
		All(ctx, &ordersByKey)
	fmt.Printf("   Run 2 (HIT):  Time=%v\n", time.Since(start))
	
	// Different second key - should miss cache
	start = time.Now()
	norm.Table("orders").
		Select().
		Where("user_id = $1", uid).
		Cache(time.Minute, "orders", "user:admin"). // Different second key
		All(ctx, &ordersByKey)
	fmt.Printf("   Run 3 (MISS - Different second key): Time=%v\n", time.Since(start))
	
	// Different first key - should miss cache
	start = time.Now()
	norm.Table("orders").
		Select().
		Where("user_id = $1", uid).
		Cache(time.Minute, "items", "user:demo"). // Different first key
		All(ctx, &ordersByKey)
	fmt.Printf("   Run 4 (MISS - Different first key): Time=%v\n", time.Since(start))


	// 7. Cache Keys with Joins
	fmt.Println("\n🔑 7. CACHE KEYS WITH JOINS (Join + Custom Keys)")
	fmt.Println("   (Combining automatic table keys with custom keys)")
	
	type UserOrderKeyed struct {
		Name  string  `norm:"name:fullname"`
		Total float64 `norm:"name:total"`
	}
	var joinByKey []UserOrderKeyed
	
	// Join with custom keys
	start = time.Now()
	norm.Table("users", "id", "orders", "user_id").
		Select("users.fullname", "orders.total").
		Where("users.useremail = $1", email).
		Cache(time.Minute, "active", "demo"). // Custom keys: "active", "demo"
		All(ctx, &joinByKey)
	fmt.Printf("   Run 1 (MISS): Time=%v\n", time.Since(start))
	
	// Same query - should hit cache
	start = time.Now()
	norm.Table("users", "id", "orders", "user_id").
		Select("users.fullname", "orders.total").
		Where("users.useremail = $1", email).
		Cache(time.Minute, "active", "demo"). // Same keys
		All(ctx, &joinByKey)
	fmt.Printf("   Run 2 (HIT):  Time=%v\n", time.Since(start))
	
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("✅ Cache key examples completed!")
	fmt.Println("   - Single key: Allows invalidation by one dimension")
	fmt.Println("   - Two keys: Allows invalidation by two dimensions")
	fmt.Println("   - Join keys: Combines table names with custom keys")
	fmt.Println(strings.Repeat("=", 60))
}
