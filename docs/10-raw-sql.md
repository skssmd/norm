# Raw SQL Queries

Complete guide to executing raw SQL queries in Norm ORM with proper routing and safety.

## Table of Contents
- [Overview](#overview)
- [Three Approaches](#three-approaches)
- [Table-Based Raw Queries](#table-based-raw-queries)
- [Explicit Shard Raw Queries](#explicit-shard-raw-queries)
- [Join-Based Raw Queries](#join-based-raw-queries)
- [Struct Scanning with Raw SQL](#struct-scanning-with-raw-sql)
- [Best Practices](#best-practices)

---

## Overview

Norm provides three safe approaches for executing raw SQL queries while maintaining proper routing:

1. **Table-Based** - Automatic routing via table name (safest)
2. **Explicit Shard** - Full control with manual shard specification
3. **Join-Based** - Native joins with co-location validation

All raw SQL queries support:
- ✅ **Automatic routing** - No manual shard selection needed (table-based)
- ✅ **Struct scanning** - Map results directly to Go structs
- ✅ **Parameterized queries** - Safe from SQL injection
- ✅ **Co-location validation** - Prevents cross-shard join errors

---

## Three Approaches

### Quick Comparison

| Approach | Routing | Use Case | Safety Level |
|----------|---------|----------|--------------|
| **Table-Based** | Automatic | Standard queries | ⭐⭐⭐ Safest |
| **Explicit Shard** | Manual | Advanced/custom | ⭐⭐ Advanced |
| **Join-Based** | Validated | Native joins | ⭐⭐⭐ Safe |

---

## Table-Based Raw Queries

Use the table name for automatic routing - **recommended for most use cases**.

### Basic Example

```go
var users []User

err := norm.Table("users").
    Raw("SELECT * FROM users WHERE age > $1", 25).
    All(ctx, &users)

if err != nil {
    log.Fatal(err)
}

for _, u := range users {
    fmt.Printf("User: %s\n", u.Name)
}
```

### With WHERE Conditions

```go
err := norm.Table("users").
    Raw("SELECT * FROM users WHERE status = $1 AND created_at > $2", 
        "active", 
        time.Now().AddDate(0, -1, 0)).
    All(ctx, &users)
```

### Single Row with First()

```go
var user User

err := norm.Table("users").
    Raw("SELECT * FROM users WHERE username = $1", "alice").
    First(ctx, &user)

if err != nil {
    log.Fatal(err)
}

fmt.Printf("Found: %s (%s)\n", user.Name, user.Email)
```

### Custom Struct Scanning

```go
type UserSummary struct {
    Fullname string
    Email    string
    Age      int
}

var summaries []UserSummary

err := norm.Table("users").
    Raw("SELECT fullname, useremail as email, age FROM users WHERE age > $1", 21).
    All(ctx, &summaries)
```

### How It Works

1. Table name (`"users"`) is used to determine the correct shard/pool
2. Query is executed on the appropriate database
3. Results are scanned into the provided struct

---

## Explicit Shard Raw Queries

For advanced use cases where you need full control over shard selection.

### Basic Example

```go
// Execute on specific shard
err := norm.Raw("SELECT COUNT(*) FROM users", "shard1").
    All(ctx, nil)
```

### Custom Functions

```go
type Result struct {
    Value int
}

var result Result

err := norm.Raw("SELECT custom_function($1) as value", "shard1", 42).
    First(ctx, &result)
```

### Cross-Table Queries (Same Shard)

```go
// Both tables must be on the same shard
err := norm.Raw(`
    SELECT u.fullname, o.total 
    FROM users u 
    JOIN orders o ON u.id = o.user_id 
    WHERE u.status = $1
`, "shard1", "active").
    All(ctx, &results)
```

### When to Use

- Custom database functions
- Complex queries spanning multiple tables on the same shard
- Performance-critical queries where you know the exact shard
- Administrative queries

### ⚠️ Important

- You must ensure the query targets tables on the specified shard
- No automatic validation - you're responsible for correctness
- Requires shard-based registry setup

---

## Join-Based Raw Queries

For native SQL joins with automatic co-location validation.

### Basic Join

```go
type UserProfile struct {
    Fullname string
    Bio      string
}

var userProfiles []UserProfile

err := norm.Join("users", "profiles").
    Raw(`
        SELECT u.fullname, p.bio 
        FROM users u 
        JOIN profiles p ON u.id = p.user_id 
        WHERE u.username = $1
    `, "alice").
    All(ctx, &userProfiles)

if err != nil {
    log.Fatal(err)
}

for _, up := range userProfiles {
    fmt.Printf("%s: %s\n", up.Fullname, up.Bio)
}
```

### Complex Join with Aliasing

```go
type OrderDetail struct {
    UserName   string  `norm:"name:user_name"`
    OrderTotal float64 `norm:"name:order_total"`
    OrderDate  string  `norm:"name:order_date"`
}

var details []OrderDetail

err := norm.Join("users", "orders").
    Raw(`
        SELECT 
            u.fullname as user_name,
            o.total as order_total,
            o.created_at as order_date
        FROM users u
        INNER JOIN orders o ON u.id = o.user_id
        WHERE o.status = $1
        ORDER BY o.created_at DESC
        LIMIT $2
    `, "completed", 10).
    All(ctx, &details)
```

### Error Handling: Non-Co-Located Tables

```go
// This will return an error if tables are on different shards
err := norm.Join("users", "analytics").
    Raw("SELECT u.fullname FROM users u JOIN analytics a ON u.id = a.user_id").
    All(ctx, nil)

if err != nil {
    // Error: "tables 'users' and 'analytics' are not co-located (different shards/pools)"
    fmt.Printf("Expected error: %v\n", err)
}
```

### How It Works

1. Validates that both tables are on the same shard/pool
2. Returns error if tables are on different shards
3. Executes query on the shared pool
4. Scans results into struct

### When to Use

- Native SQL joins between co-located tables
- Complex join logic that's easier to express in raw SQL
- Performance-critical joins
- When you need full SQL control but want routing safety

---

## Struct Scanning with Raw SQL

All raw SQL queries support automatic struct scanning.

### Basic Scanning

```go
type User struct {
    ID       uint
    Name     string `norm:"name:fullname"`
    Email    string `norm:"name:useremail"`
    Username string `norm:"name:uname"`
}

var users []User

err := norm.Table("users").
    Raw("SELECT id, fullname, useremail, uname FROM users").
    All(ctx, &users)
```

### Custom Struct with Tags

```go
type UserReport struct {
    FullName  string `norm:"name:fullname"`
    UserEmail string `norm:"name:useremail"`
    TotalOrders int  `norm:"name:order_count"`
}

var reports []UserReport

err := norm.Table("users").
    Raw(`
        SELECT 
            u.fullname,
            u.useremail,
            COUNT(o.id) as order_count
        FROM users u
        LEFT JOIN orders o ON u.id = o.user_id
        GROUP BY u.id, u.fullname, u.useremail
    `).
    All(ctx, &reports)
```

### Partial Field Mapping

```go
// Struct doesn't need all columns
type UserBasic struct {
    Name  string `norm:"name:fullname"`
    Email string `norm:"name:useremail"`
    // Other columns are ignored
}

var basics []UserBasic

err := norm.Table("users").
    Raw("SELECT * FROM users").  // Selects all, but only maps Name and Email
    All(ctx, &basics)
```

### Single Row Scanning

```go
var user User

err := norm.Table("users").
    Raw("SELECT * FROM users WHERE id = $1", 123).
    First(ctx, &user)
```

### Scanning Join Results

```go
type UserWithOrders struct {
    UserName   string  `norm:"name:fullname"`
    OrderTotal float64 `norm:"name:total"`
    OrderDate  string  `norm:"name:created_at"`
}

var results []UserWithOrders

err := norm.Join("users", "orders").
    Raw(`
        SELECT u.fullname, o.total, o.created_at
        FROM users u
        JOIN orders o ON u.id = o.user_id
        WHERE u.username = $1
    `, "alice").
    All(ctx, &results)
```

### Field Mapping Rules

Norm maps columns to struct fields in this order:

1. **Exact match with `norm:"name:column_name"` tag**
2. **Table-prefixed match** (e.g., `users.fullname` → field with `name:fullname`)
3. **Case-insensitive field name match**

```go
type Example struct {
    // Priority 1: Explicit tag
    UserEmail string `norm:"name:useremail"`  // Maps to "useremail" column
    
    // Priority 2: Table-prefixed
    // If column is "users.fullname", matches field with name:fullname tag
    
    // Priority 3: Case-insensitive
    CreatedAt time.Time  // Maps to "created_at" or "createdat"
}
```

---

## Best Practices

### 1. Use Parameterized Queries

✅ **Good:**
```go
norm.Table("users").
    Raw("SELECT * FROM users WHERE username = $1", username).
    All(ctx, &users)
```

❌ **Bad (SQL Injection Risk):**
```go
query := fmt.Sprintf("SELECT * FROM users WHERE username = '%s'", username)
norm.Table("users").Raw(query).All(ctx, &users)
```

### 2. Choose the Right Approach

```go
// ✅ Table-based for standard queries
norm.Table("users").Raw("SELECT * FROM users WHERE age > $1", 25)

// ✅ Join-based for co-located joins
norm.Join("users", "orders").Raw("SELECT u.*, o.* FROM users u JOIN orders o ON u.id = o.user_id")

// ✅ Explicit shard only when necessary
norm.Raw("SELECT custom_function()", "shard1")
```

### 3. Use Struct Scanning

```go
// ✅ Type-safe with struct scanning
var users []User
norm.Table("users").Raw("SELECT * FROM users").All(ctx, &users)

// ❌ Less type-safe without scanning
norm.Table("users").Raw("SELECT * FROM users").All(ctx, nil)
```

### 4. Handle Errors Properly

```go
err := norm.Join("users", "analytics").
    Raw("SELECT u.fullname FROM users u JOIN analytics a ON u.id = a.user_id").
    All(ctx, &results)

if err != nil {
    if strings.Contains(err.Error(), "not co-located") {
        // Handle cross-shard join error
        log.Println("Tables are on different shards, use App-Side join instead")
    } else {
        log.Fatal(err)
    }
}
```

### 5. Prefer ORM Methods When Possible

```go
// ✅ Use ORM methods for standard operations
norm.Table("users").Select().Where("age > $1", 25).All(ctx, &users)

// ⚠️ Use raw SQL only when needed
norm.Table("users").Raw("SELECT * FROM users WHERE age > $1", 25).All(ctx, &users)
```

---

## Complete Examples

### Example 1: User Analytics Report

```go
func GetUserAnalytics(minOrders int) ([]UserAnalytics, error) {
    type UserAnalytics struct {
        UserName     string  `norm:"name:fullname"`
        TotalOrders  int     `norm:"name:order_count"`
        TotalSpent   float64 `norm:"name:total_spent"`
        AvgOrderSize float64 `norm:"name:avg_order"`
    }
    
    var analytics []UserAnalytics
    
    err := norm.Table("users").
        Raw(`
            SELECT 
                u.fullname,
                COUNT(o.id) as order_count,
                COALESCE(SUM(o.total), 0) as total_spent,
                COALESCE(AVG(o.total), 0) as avg_order
            FROM users u
            LEFT JOIN orders o ON u.id = o.user_id
            GROUP BY u.id, u.fullname
            HAVING COUNT(o.id) >= $1
            ORDER BY total_spent DESC
        `, minOrders).
        All(context.Background(), &analytics)
    
    return analytics, err
}
```

### Example 2: Complex Join with Filtering

```go
func GetRecentUserOrders(username string, days int) ([]OrderDetail, error) {
    type OrderDetail struct {
        OrderID    int       `norm:"name:order_id"`
        UserName   string    `norm:"name:user_name"`
        Total      float64   `norm:"name:total"`
        Status     string    `norm:"name:status"`
        OrderDate  time.Time `norm:"name:order_date"`
    }
    
    var orders []OrderDetail
    
    cutoffDate := time.Now().AddDate(0, 0, -days)
    
    err := norm.Join("users", "orders").
        Raw(`
            SELECT 
                o.id as order_id,
                u.fullname as user_name,
                o.total,
                o.status,
                o.created_at as order_date
            FROM users u
            INNER JOIN orders o ON u.id = o.user_id
            WHERE u.username = $1 
              AND o.created_at > $2
            ORDER BY o.created_at DESC
        `, username, cutoffDate).
        All(context.Background(), &orders)
    
    return orders, err
}
```

### Example 3: Shard-Specific Query

```go
func GetShardStatistics(shardName string) (*ShardStats, error) {
    type ShardStats struct {
        UserCount  int `norm:"name:user_count"`
        OrderCount int `norm:"name:order_count"`
    }
    
    var stats ShardStats
    
    err := norm.Raw(`
        SELECT 
            (SELECT COUNT(*) FROM users) as user_count,
            (SELECT COUNT(*) FROM orders) as order_count
    `, shardName).
        First(context.Background(), &stats)
    
    return &stats, err
}
```

---

## Migration from Other ORMs

### From GORM

```go
// GORM
db.Raw("SELECT * FROM users WHERE age > ?", 25).Scan(&users)

// Norm (table-based)
norm.Table("users").Raw("SELECT * FROM users WHERE age > $1", 25).All(ctx, &users)
```

### From sqlx

```go
// sqlx
db.Select(&users, "SELECT * FROM users WHERE age > $1", 25)

// Norm (table-based)
norm.Table("users").Raw("SELECT * FROM users WHERE age > $1", 25).All(ctx, &users)
```

---

## Limitations

1. **Explicit shard mode** requires shard-based registry setup
2. **Join validation** only checks first two tables in join context
3. **No App-Side joins** - use `norm.Table()` join syntax for distributed joins
4. **PostgreSQL only** - uses `$1, $2` parameter syntax

---

## Next Steps

- Learn about [Struct Scanning](./05-crud-operations.md#struct-scanning)
- Explore [JOIN Operations](./05-crud-operations.md#join-operations)
- Understand [Database Connections](./01-database-connections.md)
