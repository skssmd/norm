# 11. Caching

Norm provides built-in caching support to dramatically improve query performance for frequently accessed data.

> **Performance Note**: Norm uses a "Top-Level Cache" optimization that checks the cache **before** building the query, resulting in **<0.1ms latency** on cache hits (matching Raw SQL performance).

## Quick Start

```go
// 1. Initialize cache (once at startup)
norm.EnableMemoryCache()
// or
norm.RegisterRedis("localhost:6379", "", 0)

// 2. Use WithCache() at the start of the chain
var users []User

norm.WithCache(5 * time.Minute).
    Table("users").
    Select().
    All(ctx, &users)
```

## Initialization

### Memory Cache (Development/Single Instance)

```go
norm.EnableMemoryCache()
```

**Pros:** Fast, no dependencies  
**Cons:** Not shared, lost on restart

### Redis Cache (Production/Distributed)

```go
err := norm.RegisterRedis("localhost:6379", "", 0)
if err != nil {
    log.Fatal(err)
}
```

**Pros:** Shared, persistent, scalable  
**Cons:** Requires Redis server

### Fallback Pattern

```go
err := norm.RegisterRedis("localhost:6379", "", 0)
if err != nil {
    log.Printf("Redis unavailable, using memory cache")
    norm.EnableMemoryCache()
}
```

## Basic Usage

### Simple Caching

Use `norm.WithCache(ttl)` to enable caching for a query.

```go
// Cache for 5 minutes
norm.WithCache(5 * time.Minute).
    Table("users").
    Select().
    All(ctx, &users)
```

**First run:** Database query + cache storage  
**Subsequent runs:** Instant return from cache (<0.1ms)

### Cache with WHERE Clause

```go
// Each unique query gets its own cache entry automatically
norm.WithCache(time.Minute).
    Table("users").
    Select().
    Where("status = $1", "active").
    All(ctx, &users)
```

## Optional Cache Keys

`WithCache` accepts up to **2 optional keys** for targeted invalidation:

```go
WithCache(ttl time.Duration, keys ...string)
```

### Why Use Keys?

Keys allow you to **invalidate specific subsets** of cached data instead of clearing everything.

### Single Key

```go
// Tag query with "active"
norm.WithCache(time.Minute, "active").
    Table("users").
    Select().
    Where("status = $1", "active").
    All(ctx, &users)

// Later: Invalidate only "active" queries
norm.Table("users").
    Update("status", "inactive").
    Where("id = $1", userId).
    InvalidateCacheReferenced("active").
    Exec(ctx)
```

**Cache key format:** `users:active:hash...`

### Two Keys

```go
// Tag with "userid" AND "productid"
norm.WithCache(time.Minute, "userid", "productid").
    Table("users").
    Select().
    Where("status = $1 AND tier = $2", "active", "premium").
    All(ctx, &users)

// Invalidate all "productid" queries
InvalidateCacheReferenced("productid")

// Or invalidate all "userid" queries
InvalidateCacheReferenced("userid")
```

**Cache key format:** `users:active:premium:hash...`


## Caching JOINs

Norm's caching works seamlessly with both native and app-side joins.

### Native JOIN (Co-located)

```go
// Automatic cache key: users:orders:hash...
norm.WithCache(time.Minute).
    Table("users", "id", "orders", "user_id").
    Select("users.name", "orders.total").
    All(ctx, &results)
```

### App-Side JOIN (Distributed)

```go
// Works across shards with <0.1ms cache hit latency
norm.WithCache(time.Minute).
    Table("users", "id", "analytics", "user_id").
    Select("users.name", "analytics.event_type").
    All(ctx, &results)
```

### JOIN with Custom Keys

```go
// Tag expensive joins
norm.WithCache(time.Hour, "key1", "key2").
    Table("users", "id", "orders", "user_id").
    Select("users.name", "orders.total").
    Where("orders.status = $1", "completed").
    All(ctx, &results)
```

## Cache Invalidation

### Referenced Invalidation (Recommended)

Invalidates **any** cache entry containing the key.

```go
// Invalidate ALL queries involving "key1"
norm.Table("users").
    Update("name", "New Name").
    Where("id = $1", userId).
    InvalidateCacheReferenced("key1").
    Exec(ctx)

// Invalidates:
// - users:hash...
// - users:key1:hash...
// - users:key1:key2:hash...
```

### Strict Invalidation

Invalidates only caches with **exact key sequence**.

```go
// Only invalidates queries with BOTH "key1" AND "key2"
norm.Table("users").
    Update("tier", "basic").
    InvalidateCache("key1", "key2").
    Exec(ctx)

// Invalidates: users:key1:key2:hash... ✓
// Keeps: users:key1:hash... ✗
```

## Cache Key Format

Generated automatically:

```
table1:table2:key1:key2:hash
```

**Examples:**

| Query | Cache Key |
|-------|-----------|
| `WithCache(ttl).Table("users")` | `users:abc123...` |
| `WithCache(ttl, "key1").Table("users")` | `users:key1:abc123...` |
| `WithCache(ttl, "key1", "key2").Table("users")` | `users:key1:key2:abc123...` |
| `WithCache(ttl).Table("users", "orders")` | `users:orders:abc123...` |

## Best Practices

1. **Always invalidate on writes**
   ```go
   norm.Table("users").
       Update("...").
       InvalidateCacheReferenced("users").
       Exec(ctx)
   ```

2. **Use appropriate TTLs**
   - Frequently changing: `30s` - `5m`
   - Moderately stable: `15m` - `1h`
   - Rarely changing: `1h` - `24h`

3. **Tag expensive queries**
   ```go
   norm.WithCache(time.Hour, "key1", "key2")
   ```

## Complete Example

```go
package main

import (
    "context"
    "time"
    "github.com/skssmd/norm"
)

func main() {
    // Initialize
    norm.EnableMemoryCache()
    
    ctx := context.Background()
    
    // Cache simple query
    var users []User
    norm.WithCache(5 * time.Minute, "key1").
        Table("users").
        Select().
        Where("status = $1", "active").
        All(ctx, &users)
    
    // Cache JOIN query
    var userOrders []UserOrder
    norm.WithCache(10 * time.Minute, "key2").
        Table("users", "id", "orders", "user_id").
        Select("users.name", "orders.total").
        All(ctx, &userOrders)
    
    // Update and invalidate
    norm.Table("users").
        Update("status", "inactive").
        Where("id = $1", 123).
        InvalidateCacheReferenced("key1").
        Exec(ctx)
}
```

## See Also

- [06. Select Queries](06-select.md)
- [09. Joins](09-joins.md)
- [10. Raw SQL](10-raw-sql.md)

---

For more detailed information, see the comprehensive [caching.md](../caching.md) guide.
