# Join Performance Comparison: Inventory

This benchmark compares three approaches for executing cross-shard JOIN queries on the **Products JOIN inventories** scenario.

## Test Environment
- **Products**: 10 rows (Shard 1)
- **inventories**: 40 rows (Shard 2)
- **Scenario**: Fetch all products with their inventory details with paid filter

## 1. Code Comparison: Setup, Query & Caching

This section compares the actual implementation effort required for each approach, covering initialization, query construction, and caching logic.

### A. Initial Setup

**Single DB Raw SQL**:
Requires setting up a connection pool. Simple, but restricted to one database.
```go
pool, _ := pgxpool.New(ctx, "postgres://user:pass@localhost:5432/db")
```

**Multi-Shard Raw SQL**:
Requires managing multiple connection pools and manually mapping data to shards.
```go
// Manual connection management
shard1, _ := pgxpool.New(ctx, "postgres://.../shard1")
shard2, _ := pgxpool.New(ctx, "postgres://.../shard2")
shard3, _ := pgxpool.New(ctx, "postgres://.../shard3")
// Developer must remember which table is on which shard
```

**Norm ORM**:
Declarative registration. Norm handles the routing automatically.
```go
// One-time registration
norm.Register("postgres://.../shard1").Shard("shard1").Primary()
norm.Register("postgres://.../shard2").Shard("shard2").Standalone("inventories")
```

### B. Execution Logic

This section compares how the query and caching logic is implemented in each approach. Note that for raw SQL, caching must be handled manually.

### Single DB Raw SQL (Simulated)
**Logic**: Manually check cache, if miss, execute SQL JOIN, then set cache.
```go
func ProductsJoininventories(ctx context.Context) ([]Result, error) {
    // 1. Manual Cache Check
    key := "product-inventories"
    // provided you have written a function for caching
    if cached, found := cache.Get(key); found {
        return cached, nil
    }

    // 2. Execute Query
   query := `
		SELECT p.name, i.warehouse, i.quantity
		FROM products p
		INNER JOIN inventories i ON p.id = i.product_id
		WHERE p.status = $1 AND i.quantity > $2 -- Native filtering on both tables
	`
    rows, _ := pool.Query(ctx, query)
    
    // 3. Set Cache
    cache.Set(key, rows, time.Minute)
    return rows, nil
}
```

### Multi-Shard Raw SQL
**Logic**: Manually fetch from Shard 1, extract IDs, fetch from Shard 2, join in memory, handling cache manually.
```go
func ProductsJoininventories(ctx context.Context) ([]Result, error) {
     
    // 1. Manual Cache Check
    key1 := "multi-shard-product"
    key2 := "multi-shard-inventories"
    // provided you have written a function for caching
    if cached1, found := cache.Get(key1); found {
        return cached1, nil 
    }
    if cached2, found := cache.Get(key2); found {
        return cached2, nil
    }

    // 2. Fetch Left Table (Shard 1) with status filter
    // Only fetch products that are 'paid'.
    rows1, _ := shard1.Query(ctx, 
        "SELECT id, name FROM products WHERE status = $1", 
        "paid")

    // 3. Fetch Right Table (Shard 2) using IDs and quantity filter
    ids := extractIDs(products) // IDs are only from the 'paid' products scanned above
    rows2, _ := shard2.Query(ctx, 
        "SELECT ... FROM inventories WHERE product_id = ANY($1) AND quantity > $2", 
        ids, 
        0)
 
    // 5. Manual Cache Set
    cache.Set(key1, rows1, time.Minute) 
    cache.Set(key2, rows2, time.Minute)
    
    return rows1, rows2, nil 
}
```
### Norm ORM
**Logic**: Declarative query with **Caching** (Optimized).

```go

func ProductsJoininventories(ctx context.Context) ([]Result, error) {
    var results []Result
    
    // 1. Declarative Query + Caching

    err := norm.WithCache(time.Minute, "bench", "product-inventories").
        Table("products", "id", "inventories", "product_id").
        Select("products.name", "inventories.warehouse", "inventories.quantity").
        Where("products.status = $1 and inventories.quantity > $2", "paid", 0).
        All(ctx, &results)
        
    return results, err
}
```

## 2. Performance Results

### Products JOIN inventories

| Approach | Avg No Cache | Avg With Cache |
|----------|--------------|----------------|
| Single DB Raw SQL | 566.271ms | 358µs |
| Multi-Shard Raw SQL | 7.077336s | 337µs |
| Norm ORM | 641.169ms | 350µs |



Norm ORM provides a solution that eliminates the severe coding burden of coordinating multiple database transactions, caching, and with **60-70% less code**.

| Complexity Eliminated by Norm ORM | Multi-Shard Raw SQL Burden | Norm ORM Solution |
|:---|:---|:---|
| **Multi-Shard Routing** | Manual hardcoding of connections, explicit calls to specific shards (`shard1.Query`, `shard2.Query`). | **Automated** and context-aware routing through declarative table registration. |
| **Cross-Database Operations** | **Explicit Cross-Database Operations:** The application must request data from multiple databases manually. | **Optimized Distributed Engine:** Handles Database Routing and Data Merging automatically, Without needing to memorize Table Ownership |
| **Caching** | Manual cache management, including setting and getting cache values. | **Automated** and context-aware caching through declarative table registration. |


**In conclusion, Norm ORM transforms the complexity of Multi-Shard Raw SQL into a simple, declarative pattern, while simultaneously providing a native Optimized Raw SQL performance.**
