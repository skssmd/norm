# JOIN Operations

Complete guide to JOIN operations in Norm ORM.

## Table of Contents
- [Overview](#overview)
- [Join Types](#join-types)
- [Native JOIN](#native-join)
- [App-Side JOIN](#app-side-join)
- [Distributed JOIN](#distributed-join)
- [Best Practices](#best-practices)

---

## Overview

Norm supports multiple types of joins with automatic routing:
- ✅ **Native Join** - Standard SQL JOIN (same database)
- ✅ **App-Side Join** - Application-level join (skey relationships)
- ✅ **Distributed Join** - Cross-database join (sharded architectures)

---

## Join Types

### Join Strategies

| Join Type | Key Type | Execution Flow |
|-----------|----------|----------------|
| **Native Join** | `fkey` (Same DB) | **Database Side**: Standard SQL `JOIN` executed by DB engine. |
| **Non-Native** | `fkey` (Sharded) | **ORM Side**: Engine detects and splits queries, fetches from shards, and merges results. |
| **Soft Key** | `skey` (Any) | **ORM Side**: Engine fetches IDs, queries related map, and links results. |

---

## Native JOIN

When both tables are on the same database/shard, Norm uses standard SQL JOIN:

### Basic JOIN Syntax

```go
// JOIN users and orders tables
// Syntax: Table(leftTable, leftKey, rightTable, rightKey)
err := norm.Table("users", "id", "orders", "user_id").
    Select("users.fullname", "orders.total").
    Where("users.username = $1", "alice").
    All(ctx, &results)
```

**Generated SQL:**
```sql
SELECT users.fullname, orders.total 
FROM users 
INNER JOIN orders ON users.id = orders.user_id 
WHERE users.username = $1
```

### JOIN with Struct Scanning

```go
type UserOrder struct {
    UserName   string  `norm:"name:fullname"`
    OrderTotal float64 `norm:"name:total"`
}

var userOrders []UserOrder

err := norm.Table("users", "id", "orders", "user_id").
    Select("users.fullname", "orders.total").
    Where("users.username = $1", "alice").
    All(ctx, &userOrders)

if err != nil {
    log.Fatal(err)
}

for _, item := range userOrders {
    fmt.Printf("User: %s, Total: %.2f\n", item.UserName, item.OrderTotal)
}
```

### JOIN with Aliasing

Use aliases to avoid column name conflicts:

```go
type UserOrderDetail struct {
    UserID    int     `norm:"name:user_id_alias"`
    UserName  string  `norm:"name:fullname"`
    OrderID   int     `norm:"name:order_id_alias"`
    Total     float64 `norm:"name:total"`
}

var details []UserOrderDetail

err := norm.Table("users", "id", "orders", "user_id").
    Select(
        "users.id as user_id_alias",
        "users.fullname",
        "orders.id as order_id_alias",
        "orders.total",
    ).
    All(ctx, &details)
```

---

## App-Side JOIN

When using soft keys (`skey` tag), Norm performs an application-level join:

```go
// Profile struct with skey
type Profile struct {
    ID     int    `norm:"pk;auto"`
    UserID int    `norm:"skey:users.id"`  // Soft key - app-level join
    Bio    string
}

type UserProfile struct {
    Fullname string
    Bio      string
}

var userProfiles []UserProfile

// Automatically uses App-Side Join due to skey
err := norm.Table("users", "id", "profiles", "user_id").
    Select("users.fullname", "profiles.bio").
    Where("users.username = $1", "alice").
    All(ctx, &userProfiles)
```

**How it works:**
1. Fetches users matching the WHERE clause
2. Extracts user IDs
3. Fetches profiles with matching user_ids
4. Combines results in application memory

---

## Distributed JOIN

When tables are on different shards, Norm automatically performs a distributed join:

```go
// Users on shard1, Analytics on shard2
type UserAnalytics struct {
    UserName  string `norm:"name:fullname"`
    EventType string `norm:"name:event_type"`
}

var userAnalytics []UserAnalytics

err := norm.Table("users", "id", "analytics", "user_id").
    Select("users.fullname", "analytics.event_type").
    Where("users.username = $1", "alice").
    All(ctx, &userAnalytics)
```

**How it works:**
1. Queries left table (users) on its shard
2. Extracts join keys
3. Queries right table (analytics) on its shard with IN clause
4. Combines results in application memory

---

## Best Practices

### 1. Use Struct Scanning for Type Safety

```go
// ✅ Good - Type-safe
type UserOrder struct {
    UserName string `norm:"name:fullname"`
    Total    float64 `norm:"name:total"`
}
var results []UserOrder
norm.Table("users", "id", "orders", "user_id").Select(...).All(ctx, &results)
```

### 2. Select Only Needed Columns

```go
// ✅ Good - Efficient
Select("users.fullname", "orders.total")

// ❌ Bad - Wasteful
Select()  // Selects all columns from both tables
```

### 3. Use WHERE to Filter Early

```go
// ✅ Good - Filter before join
norm.Table("users", "id", "orders", "user_id").
    Select(...).
    Where("users.status = $1", "active").
    All(ctx, &results)
```

### 4. Be Aware of Join Type

```go
// Native join - fast (same database)
norm.Table("users", "id", "orders", "user_id")

// Distributed join - slower (different shards)
norm.Table("users", "id", "analytics", "user_id")
```

---

## Complete Example

```go
func GetUserOrders(username string) ([]UserOrder, error) {
    type UserOrder struct {
        UserName   string    `norm:"name:fullname"`
        UserEmail  string    `norm:"name:useremail"`
        OrderID    uint      `norm:"name:order_id"`
        OrderTotal float64   `norm:"name:total"`
        Status     string    `norm:"name:status"`
        OrderDate  time.Time `norm:"name:created_at"`
    }
    
    var orders []UserOrder
    
    err := norm.Table("users", "id", "orders", "user_id").
        Select(
            "users.fullname",
            "users.useremail",
            "orders.id as order_id",
            "orders.total",
            "orders.status",
            "orders.created_at",
        ).
        Where("users.username = $1", username).
        OrderBy("orders.created_at DESC").
        All(context.Background(), &orders)
    
    return orders, err
}
```

---

## Next Steps

- Learn about [Raw SQL](./10-raw-sql.md) for custom join queries
- Explore [SELECT Operations](./06-select.md) for struct scanning details
- See [Model Definition](./02-model-definition.md) for fkey vs skey tags
