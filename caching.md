# Caching in Norm

Norm provides a robust caching system with support for both in-memory and Redis backends. It features automatic key generation, targeted invalidation, and support for complex queries including JOINs.

## Table of Contents

- [Initialization](#initialization)
  - [Memory Cache](#memory-cache)
  - [Redis Cache](#redis-cache)
  - [Fallback Strategy](#fallback-strategy)
- [Basic Usage](#basic-usage)
  - [Simple Caching](#simple-caching)
  - [Cache TTL](#cache-ttl)
  - [Cache Behavior](#cache-behavior)
- [Optional Cache Keys](#optional-cache-keys)
  - [Why Use Custom Keys?](#why-use-custom-keys)
  - [Single Key Example](#single-key-example)
  - [Two Keys Example](#two-keys-example)
  - [Cache Key Format](#cache-key-format)
- [Advanced Caching](#advanced-caching)
  - [Caching JOIN Queries](#caching-join-queries)
  - [Caching Raw SQL](#caching-raw-sql)
- [Cache Invalidation](#cache-invalidation)
  - [Referenced Invalidation](#1-referenced-invalidation-broad)
  - [Strict Invalidation](#2-strict-invalidation-targeted)
- [Best Practices](#best-practices)
- [Performance Tips](#performance-tips)

---

## Initialization

Before using caching, you must initialize a cache provider. Norm supports two backends: **Memory Cache** (in-process) and **Redis** (distributed).

### Memory Cache

Ideal for development, testing, or single-instance applications.

**Advantages:**
- Zero external dependencies
- Fast (no network overhead)
- Simple setup

**Limitations:**
- Not shared across multiple instances
- Lost on application restart
- Limited by available RAM

```go
import "github.com/skssmd/norm"

func main() {
    // Enable memory cache
    norm.EnableMemoryCache()
    
    // Now you can use .Cache() on queries
}
```

### Redis Cache

Recommended for production and distributed systems.

**Advantages:**
- Shared across multiple application instances
- Persistent (survives restarts)
- Scalable and battle-tested

**Requirements:**
- Redis server running and accessible

```go
import (
    "log"
    "github.com/skssmd/norm"
)

func main() {
    // Register Redis cache
    // Parameters: address, password, database
    err := norm.RegisterRedis("localhost:6379", "", 0)
    if err != nil {
        log.Fatal("Failed to connect to Redis:", err)
    }
    
    // Cache is now ready to use
}
```

**Redis Configuration Options:**

```go
// Local Redis (default port)
norm.RegisterRedis("localhost:6379", "", 0)

// Remote Redis with authentication
norm.RegisterRedis("redis.example.com:6379", "your-password", 0)

// Using specific database
norm.RegisterRedis("localhost:6379", "", 5) // Use DB 5
```

### Fallback Strategy

A common pattern is to try Redis first, then fall back to memory cache:

```go
func initCache() {
    // Try Redis first
    err := norm.RegisterRedis("localhost:6379", "", 0)
    if err != nil {
        log.Printf("Redis not available (%v), using memory cache", err)
        norm.EnableMemoryCache()
    } else {
        log.Println("Redis cache initialized")
    }
}
```

---

## Basic Usage

### Simple Caching

Add `.Cache(ttl)` to any query to enable caching:

```go
import (
    "context"
    "time"
)

ctx := context.Background()

// Cache for 5 minutes
var users []User
err := norm.Table("users").
    Select().
    Cache(5 * time.Minute).
    All(ctx, &users)
```

**First execution:** Query hits the database, result is cached
**Subsequent executions:** Returns cached data instantly (no database query)

### Cache TTL

The Time-To-Live (TTL) determines how long data stays in cache:

```go
// Cache for 1 minute
Cache(time.Minute)

// Cache for 1 hour
Cache(time.Hour)

// Cache for 30 seconds
Cache(30 * time.Second)

// Cache for 24 hours
Cache(24 * time.Hour)
```

### Cache Behavior

**Cache Hit:**
```
[CACHE] Key: users:abc123hash...
[CACHE] Status: HIT
```
- Data returned from cache
- No database query executed
- Extremely fast response

**Cache Miss:**
```
[CACHE] Key: users:abc123hash...
[CACHE] Status: MISS (Pulling from DB)
```
- Query executes against database
- Result is cached for future requests
- Normal query performance

---

## Optional Cache Keys

The `.Cache()` method accepts up to **2 optional string keys** for more granular cache control.

```go
Cache(ttl time.Duration, keys ...string)
```

### Why Use Custom Keys?

Custom keys enable **targeted cache invalidation**. Instead of invalidating all user queries, you can invalidate only specific subsets:

- **Without keys:** Invalidating "users" clears ALL user queries
- **With keys:** Invalidating "users:active" clears only active user queries

**Use cases:**
- User roles/permissions (`"admin"`, `"user"`)
- Data states (`"active"`, `"pending"`, `"archived"`)
- Feature flags (`"premium"`, `"trial"`)
- Regions/tenants (`"us-east"`, `"eu-west"`)

### Single Key Example

Use one key to categorize queries:

```go
// Cache active users with key "active"
var activeUsers []User
norm.Table("users").
    Select().
    Where("status = $1", "active").
    Cache(time.Minute, "active"). // Single key
    All(ctx, &activeUsers)

// Cache admin users with key "admin"
var adminUsers []User
norm.Table("users").
    Select().
    Where("role = $1", "admin").
    Cache(time.Minute, "admin"). // Different key
    All(ctx, &adminUsers)

// These create separate cache entries:
// - users:active:hash...
// - users:admin:hash...
```

**Invalidation:**
```go
// Only invalidates queries tagged with "active"
norm.Table("users").
    Update("status", "inactive").
    Where("id = $1", userId).
    InvalidateCacheReferenced("active"). // Only "active" queries cleared
    Exec(ctx)
```

### Two Keys Example

Use two keys for hierarchical categorization:

```go
// Cache premium active users
var premiumActive []User
norm.Table("users").
    Select().
    Where("status = $1 AND tier = $2", "active", "premium").
    Cache(time.Minute, "active", "premium"). // Two keys
    All(ctx, &premiumActive)

// Cache trial active users
var trialActive []User
norm.Table("users").
    Select().
    Where("status = $1 AND tier = $2", "active", "trial").
    Cache(time.Minute, "active", "trial"). // Different second key
    All(ctx, &trialActive)

// These create separate cache entries:
// - users:active:premium:hash...
// - users:active:trial:hash...
```

**Granular Invalidation:**
```go
// Invalidate only premium users
norm.Table("users").
    Update("tier", "basic").
    Where("id = $1", userId).
    InvalidateCacheReferenced("premium"). // Only "premium" queries
    Exec(ctx)

// Invalidate all active users (both premium and trial)
norm.Table("users").
    Update("status", "inactive").
    Where("id = $1", userId).
    InvalidateCacheReferenced("active"). // All "active" queries
    Exec(ctx)
```

### Cache Key Format

Cache keys are automatically generated in this format:

```
table1:table2:key1:key2:hash
```

**Components:**
- **Tables:** Automatically included (e.g., `users`, `users:orders`)
- **Custom Keys:** Your optional keys (up to 2)
- **Hash:** SHA256 hash of query + arguments (ensures uniqueness)

**Examples:**

| Query | Generated Key |
|-------|---------------|
| `Table("users").Cache(ttl)` | `users:abc123...` |
| `Table("users").Cache(ttl, "active")` | `users:active:abc123...` |
| `Table("users").Cache(ttl, "active", "premium")` | `users:active:premium:abc123...` |
| `Table("users", "id", "orders", "user_id").Cache(ttl)` | `users:orders:abc123...` |
| `Table("users", "id", "orders", "user_id").Cache(ttl, "recent")` | `users:orders:recent:abc123...` |

---

## Advanced Caching

### Caching JOIN Queries

Norm automatically handles caching for both **native** and **app-side** JOINs.

**Native JOIN (co-located tables):**
```go
// Tables on same database/shard
var results []UserOrder
norm.Table("users", "id", "orders", "user_id").
    Select("users.name", "orders.total").
    Where("users.status = $1", "active").
    Cache(time.Minute, "active"). // Optional key
    All(ctx, &results)

// Cache key: users:orders:active:hash...
```

**App-Side JOIN (distributed/skey):**
```go
// Tables on different shards or using skey
var results []UserAnalytics
norm.Table("users", "id", "analytics", "user_id").
    Select("users.name", "analytics.event_type").
    Cache(time.Minute).
    All(ctx, &results)

// Cache key: users:analytics:hash...
// Entire merged result is cached
```

**JOIN with Custom Keys:**
```go
// Tag expensive join queries for targeted invalidation
norm.Table("users", "id", "orders", "user_id").
    Select("users.name", "orders.total").
    Where("orders.status = $1", "completed").
    Cache(time.Hour, "reports", "completed"). // Two keys
    All(ctx, &results)

// Cache key: users:orders:reports:completed:hash...
```

### Caching Raw SQL

Raw SQL queries can be cached by specifying the table for routing:

```go
// Single table raw query
var users []User
norm.Table("users").
    Raw("SELECT * FROM users WHERE created_at > $1", time.Now().AddDate(0, -1, 0)).
    Cache(time.Minute, "recent").
    All(ctx, &users)

// JOIN raw query (tables must be co-located)
type Result struct {
    Name  string
    Total float64
}
var results []Result
norm.Join("users", "orders").
    Raw("SELECT u.name, o.total FROM users u JOIN orders o ON u.id = o.user_id").
    Cache(time.Minute).
    All(ctx, &results)
```

---

## Cache Invalidation

Norm provides two invalidation strategies to keep cached data fresh.

### 1. Referenced Invalidation (Broad)

**Method:** `InvalidateCacheReferenced(keys...)`  
**Pattern:** `*<key>*` (matches any cache entry containing the key)

Use this to invalidate **any** cache entry that references a specific table or tag. This is the **safest and most common** strategy.

```go
// Invalidate ALL queries involving "users" table
norm.Table("users").
    Update("name", "New Name").
    Where("id = $1", userId).
    InvalidateCacheReferenced("users").
    Exec(ctx)

// Invalidates:
// - users:hash...
// - users:active:hash...
// - users:orders:hash...
// - users:profiles:premium:hash...
```

**Invalidate by custom tag:**
```go
// Invalidate all queries tagged with "active"
norm.Table("users").
    Update("status", "inactive").
    Where("id = $1", userId).
    InvalidateCacheReferenced("active").
    Exec(ctx)

// Invalidates:
// - users:active:hash...
// - users:active:premium:hash...
// - orders:active:hash...
```

**Multiple keys:**
```go
// Invalidate queries matching ANY of the keys
norm.Table("users").
    Update("...").
    InvalidateCacheReferenced("users", "active").
    Exec(ctx)

// Invalidates anything with "users" OR "active"
```

### 2. Strict Invalidation (Targeted)

**Method:** `InvalidateCache(keys...)`  
**Pattern:** `*<table>*<key1>:<key2>*`

Use this for **precise control** when you know exactly which cache subset to clear. Requires exact key sequence.

```go
// Only invalidates caches with BOTH "active" AND "premium"
norm.Table("users").
    Update("tier", "basic").
    Where("id = $1", userId).
    InvalidateCache("active", "premium").
    Exec(ctx)

// Invalidates:
// - users:active:premium:hash... ✓
// 
// Does NOT invalidate:
// - users:active:hash... ✗ (missing "premium")
// - users:premium:hash... ✗ (missing "active")
// - users:active:trial:hash... ✗ (wrong second key)
```

---

## Best Practices

1. **Use Referenced Invalidation by Default**
   ```go
   // Safe: Clears all user-related caches
   InvalidateCacheReferenced("users")
   ```

2. **Tag Expensive Queries**
   ```go
   // Tag dashboard queries for easy invalidation
   Cache(time.Hour, "dashboard", "analytics")
   ```

3. **Set Appropriate TTLs**
   - Frequently changing data: `30 * time.Second` to `5 * time.Minute`
   - Moderately stable data: `15 * time.Minute` to `1 * time.Hour`
   - Rarely changing data: `1 * time.Hour` to `24 * time.Hour`

4. **Use Consistent Key Naming**
   ```go
   // Good: Consistent naming scheme
   "active", "inactive", "pending"
   "admin", "user", "guest"
   "premium", "trial", "free"
   
   // Avoid: Inconsistent naming
   "active", "not_active", "isActive"
   ```

5. **Monitor Cache Performance**
   ```go
   // Check logs for cache hits/misses
   // [CACHE] Status: HIT   <- Good!
   // [CACHE] Status: MISS  <- Expected on first run
   ```

6. **Invalidate on Writes**
   ```go
   // Always invalidate after INSERT/UPDATE/DELETE
   norm.Table("users").
       Update("...").
       InvalidateCacheReferenced("users").
       Exec(ctx)
   ```

---

## Performance Tips

1. **Cache Read-Heavy Queries**
   - Dashboard statistics
   - User profiles
   - Product catalogs
   - Configuration data

2. **Don't Cache Write-Heavy Data**
   - Real-time analytics
   - Frequently updated counters
   - Session data with short TTLs

3. **Use Longer TTLs for Stable Data**
   ```go
   // Rarely changes
   Cache(24 * time.Hour, "config")
   ```

4. **Batch Invalidations**
   ```go
   // Invalidate multiple related caches at once
   InvalidateCacheReferenced("users", "profiles", "settings")
   ```

5. **Monitor Memory Usage**
   - Memory cache: Limited by RAM
   - Redis: Monitor Redis memory usage
   - Set appropriate TTLs to prevent unbounded growth

---

## Complete Example

```go
package main

import (
    "context"
    "log"
    "time"
    "github.com/skssmd/norm"
)

func main() {
    // 1. Initialize cache
    err := norm.RegisterRedis("localhost:6379", "", 0)
    if err != nil {
        log.Printf("Redis unavailable, using memory cache")
        norm.EnableMemoryCache()
    }
    
    ctx := context.Background()
    
    // 2. Cache a simple query
    var activeUsers []User
    norm.Table("users").
        Select().
        Where("status = $1", "active").
        Cache(5 * time.Minute, "active").
        All(ctx, &activeUsers)
    
    // 3. Cache a JOIN query
    var userOrders []UserOrder
    norm.Table("users", "id", "orders", "user_id").
        Select("users.name", "orders.total").
        Where("users.status = $1", "active").
        Cache(10 * time.Minute, "active", "orders").
        All(ctx, &userOrders)
    
    // 4. Update and invalidate
    norm.Table("users").
        Update("status", "inactive").
        Where("id = $1", 123).
        InvalidateCacheReferenced("active"). // Clears all "active" caches
        Exec(ctx)
    
    // 5. Next query will be a cache MISS (data was invalidated)
    norm.Table("users").
        Select().
        Where("status = $1", "active").
        Cache(5 * time.Minute, "active").
        All(ctx, &activeUsers)
}
```
