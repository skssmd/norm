# 11. Caching

Norm provides built-in caching support to dramatically improve query performance for frequently accessed data.

## Quick Start

```go
// 1. Initialize cache (once at startup)
norm.EnableMemoryCache()
// or
norm.RegisterRedis("localhost:6379", "", 0)

// 2. Add .Cache() to any query
var users []User
norm.Table("users").
    Select().
    Cache(5 * time.Minute).
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

```go
// Cache for 5 minutes
norm.Table("users").
    Select().
    Cache(5 * time.Minute).
    All(ctx, &users)
```

**First run:** Database query + cache storage  
**Subsequent runs:** Instant return from cache

### Cache with WHERE Clause

```go
// Each unique query gets its own cache entry
norm.Table("users").
    Select().
    Where("status = $1", "active").
    Cache(time.Minute).
    All(ctx, &users)
```

## Optional Cache Keys

The `.Cache()` method accepts up to **2 optional keys** for targeted invalidation:

```go
Cache(ttl time.Duration, keys ...string)
```

### Why Use Keys?

Keys allow you to **invalidate specific subsets** of cached data instead of clearing everything.

### Single Key

```go
// Tag query with "active"
norm.Table("users").
    Select().
    Where("status = $1", "active").
    Cache(time.Minute, "active").
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
// Tag with "active" AND "premium"
norm.Table("users").
    Select().
    Where("status = $1 AND tier = $2", "active", "premium").
    Cache(time.Minute, "active", "premium").
    All(ctx, &users)

// Invalidate all "premium" queries
InvalidateCacheReferenced("premium")

// Or invalidate all "active" queries
InvalidateCacheReferenced("active")
```

**Cache key format:** `users:active:premium:hash...`

### Common Key Patterns

```go
// By status
Cache(ttl, "active")
Cache(ttl, "pending")
Cache(ttl, "archived")

// By role
Cache(ttl, "admin")
Cache(ttl, "user")

// By tier
Cache(ttl, "premium")
Cache(ttl, "trial")

// Hierarchical
Cache(ttl, "reports", "monthly")
Cache(ttl, "reports", "yearly")
```

## Caching JOINs

### Native JOIN (Co-located)

```go
// Automatic cache key: users:orders:hash...
norm.Table("users", "id", "orders", "user_id").
    Select("users.name", "orders.total").
    Cache(time.Minute).
    All(ctx, &results)
```

### App-Side JOIN (Distributed)

```go
// Works with skey or different shards
norm.Table("users", "id", "analytics", "user_id").
    Select("users.name", "analytics.event_type").
    Cache(time.Minute).
    All(ctx, &results)
```

### JOIN with Custom Keys

```go
// Tag expensive joins
norm.Table("users", "id", "orders", "user_id").
    Select("users.name", "orders.total").
    Where("orders.status = $1", "completed").
    Cache(time.Hour, "reports", "completed").
    All(ctx, &results)

// Cache key: users:orders:reports:completed:hash...
```

## Cache Invalidation

### Referenced Invalidation (Recommended)

Invalidates **any** cache entry containing the key.

```go
// Invalidate ALL queries involving "users"
norm.Table("users").
    Update("name", "New Name").
    Where("id = $1", userId).
    InvalidateCacheReferenced("users").
    Exec(ctx)

// Invalidates:
// - users:hash...
// - users:active:hash...
// - users:orders:hash...
```

### Strict Invalidation

Invalidates only caches with **exact key sequence**.

```go
// Only invalidates queries with BOTH "active" AND "premium"
norm.Table("users").
    Update("tier", "basic").
    InvalidateCache("active", "premium").
    Exec(ctx)

// Invalidates: users:active:premium:hash... ✓
// Keeps: users:active:hash... ✗
```

## Cache Key Format

Generated automatically:

```
table1:table2:key1:key2:hash
```

**Examples:**

| Query | Cache Key |
|-------|-----------|
| `Table("users").Cache(ttl)` | `users:abc123...` |
| `Table("users").Cache(ttl, "active")` | `users:active:abc123...` |
| `Table("users").Cache(ttl, "active", "premium")` | `users:active:premium:abc123...` |
| `Table("users", "id", "orders", "user_id").Cache(ttl)` | `users:orders:abc123...` |

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
   Cache(time.Hour, "dashboard", "analytics")
   ```

4. **Use consistent naming**
   ```go
   // Good
   "active", "inactive", "pending"
   
   // Avoid
   "active", "not_active", "isActive"
   ```

5. **Monitor cache hits**
   ```
   [CACHE] Status: HIT   <- Good!
   [CACHE] Status: MISS  <- First run
   ```

## Performance Tips

**Cache these:**
- Dashboard queries
- User profiles
- Product catalogs
- Configuration data
- Reports

**Don't cache these:**
- Real-time data
- Frequently updated counters
- Write-heavy data

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
    norm.Table("users").
        Select().
        Where("status = $1", "active").
        Cache(5 * time.Minute, "active").
        All(ctx, &users)
    
    // Cache JOIN query
    var userOrders []UserOrder
    norm.Table("users", "id", "orders", "user_id").
        Select("users.name", "orders.total").
        Cache(10 * time.Minute, "orders").
        All(ctx, &userOrders)
    
    // Update and invalidate
    norm.Table("users").
        Update("status", "inactive").
        Where("id = $1", 123).
        InvalidateCacheReferenced("active").
        Exec(ctx)
}
```

## See Also

- [06. Select Queries](06-select.md)
- [09. Joins](09-joins.md)
- [10. Raw SQL](10-raw-sql.md)

---

For more detailed information, see the comprehensive [caching.md](../caching.md) guide.
