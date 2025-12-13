# Norm ORM Features

Norm is a lightweight, sharding-aware ORM for Go, designed for high-scale distributed database architectures.

## ✅ Implemented Features

### 1. Advanced Connection Management
Support for complex database topologies out of the box.

- **Global Mode**:
    - **Monolith**: Single primary connection.
    - **Read/Write Split**: Dedicated write pool and load-balanced read pools.
    - **Primary/Replica**: Automatic fallback to replicas.
- **Sharding Mode**:
    - **Key-Based Sharding**: Distribute data across multiple shards.
    - **Role-Based Sharding**: Define specific roles (primary, read, write) per shard.
    - **Standalone Tables**: Pin specific tables to specific shards.

```go
// Example: Registering a sharded setup
norm.Register(norm.Config{
    Mode: "shard",
    Shards: map[string]norm.ShardConfig{
        "shard1": {
            Primary:  "postgres://user:pass@host1:5432/db1",
            Replicas: []string{"postgres://..."},
        },
        "shard2": {
            Primary: "postgres://user:pass@host2:5432/db2",
        },
    },
    Mappings: map[string]string{
        "users":    "shard1", // Users live on shard1
        "orders":   "shard2", // Orders live on shard2
    },
})
```

### 2. Schema Management & Auto-Migration
Define your schema using Go structs and let Norm handle the database creation.

- **Struct Tags**:
    - `pk`: Primary Key.
    - `auto`: Auto-increment (Identity column).
    - `notnull`: NOT NULL constraint.
    - `unique`: UNIQUE constraint.
    - `index`: Create an index.
    - `skey`: Soft Key (Virtual Foreign Key) for distributed joins.
- **Auto-Migration**:
    - Automatically creates tables, indexes, and constraints.
    - **Distributed-Aware**: Skips foreign keys that reference tables on different shards to prevent errors.

```go
type User struct {
    ID    uint   `norm:"pk;auto"`
    Name  string `norm:"notnull;index"`
    Email string `norm:"unique"`
}

// Auto-migrate all registered models
norm.AutoMigrate(ctx)
```

### 3. CRUD Operations
Fluent API for standard database operations.

- **Insert**: Single or Bulk inserts.
- **Select**: Chainable builder with `Where`, `OrderBy`, `Limit`, `Offset`.
- **Update**: Update single fields, multiple fields, or entire structs.
- **Delete**: Delete with conditions.
- **Count**: Efficient counting.

```go
// Select
var users []User
norm.Table("users").Select("name", "email").Where("id > $1", 100).All(ctx, &users)

// Update
norm.Table("users").Update("status", "active").Where("id = $1", 1).Exec(ctx)
```

### 4. Robust Join Support
Seamlessly join tables regardless of where they live.

- **Native Joins**: Uses SQL `JOIN` when tables are co-located on the same database.
- **App-Side Joins**: Automatically detects distributed tables or `skey` relationships and performs efficient application-side fetching and merging.
- **Soft Keys (`skey`)**: Explicitly define relationships across shards.

```go
type Profile struct {
    UserID uint `norm:"skey:users.id"` // References users.id (potentially on another shard)
    Bio    string
}

// Norm automatically chooses Native vs App-Side join
norm.Table("users", "id", "profiles", "user_id").
    Select("users.name", "profiles.bio").
    All(ctx, &results)
```

### 5. Type Safety & Utilities
- **Registry Reset**: Helper for testing to clear pools.
- **Table Output**: Query results can be printed in a formatted table for debugging.
- **Type-Safe Merging**: App-Side joins handle type mismatches (e.g., `int` vs `int64`) gracefully.

---

## 🚀 Future Features (Roadmap)

### 1. Transactions & Sagas
Support for distributed transactions.
- **Local Transactions**: `tx := norm.Begin()` for single-shard operations.
- **Sagas**: Orchestration for multi-shard transactions (compensating actions).

### 2. Hooks / Callbacks
Lifecycle hooks for models.
- `BeforeCreate`, `AfterCreate`
- `BeforeSave`, `AfterSave`
- `BeforeDelete`, `AfterDelete`

### 3. Soft Deletes
Built-in support for non-destructive deletes.
- Automatically filter out records with `deleted_at` set.
- `Unscoped()` mode to query deleted records.

### 4. Relationships (Preloading)
ORM-style relationship loading.
- `HasMany`, `BelongsTo`, `ManyToMany`.
- `Preload("Orders")` to automatically fetch related data.

### 5. Raw SQL Support
Escape hatch for complex queries.
- `norm.Raw("SELECT * FROM complex_view").Scan(&result)`

### 6. Caching Layer
Integrated caching for read-heavy workloads.
- Redis/Memcached integration.
- Query result caching with TTL.

### 7. Observability
Built-in tracing and metrics.
- OpenTelemetry integration.
- Slow query logging.
