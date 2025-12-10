# Database Migrations

This guide covers how automatic migrations work in Norm ORM and how to manage database schema changes.

## Table of Contents
- [Overview](#overview)
- [How Migrations Work](#how-migrations-work)
- [Migration Operations](#migration-operations)
- [Running Migrations](#running-migrations)
- [Scenarios](#scenarios)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

---

## Overview

Norm provides **automatic schema migrations** that:
- ✅ Create tables from struct definitions
- ✅ Add new columns automatically
- ✅ Create indexes and foreign keys
- ✅ Handle multiple databases/shards
- ✅ Run migrations concurrently
- ❌ **Never** drop columns (safety first)
- ❌ **Never** change column types automatically

---

## How Migrations Work

### Migration Flow

```
1. Register databases → norm.Register(dsn).Primary()
2. Register tables    → norm.RegisterTable(Model{}, "table_name")
3. Run migrations     → norm.Norm()
```

### What Happens During Migration

```go
norm.Norm() // Triggers:
```

1. **Analyze Models**: Parse struct tags and generate SQL
2. **Check Tables**: Query database for existing tables
3. **Create New Tables**: Execute CREATE TABLE for missing tables
4. **Update Existing Tables**: Add missing columns
5. **Create Indexes**: Add missing indexes
6. **Create Foreign Keys**: Add missing constraints
7. **Print Summary**: Show registry and migration results

---

## Migration Operations

### Automatic Operations

| Operation | When | Example |
|-----------|------|---------|
| **CREATE TABLE** | Table doesn't exist | New model added |
| **ADD COLUMN** | Column missing | New field in struct |
| **CREATE INDEX** | Index missing | Added `index` tag |
| **ADD FOREIGN KEY** | FK missing | Added `fkey` tag |

### Manual Operations (Not Automatic)

| Operation | Why | How to Handle |
|-----------|-----|---------------|
| **DROP COLUMN** | Safety | Manual SQL |
| **ALTER COLUMN TYPE** | Data loss risk | Manual SQL + data migration |
| **RENAME COLUMN** | Breaking change | Manual SQL |
| **DROP TABLE** | Data loss | Use `norm.DropTables()` |

---

## Running Migrations

### Basic Migration

```go
package main

import (
    "log"
    "github.com/skssmd/norm"
)

type User struct {
    ID    uint   `norm:"index;notnull;pk"`
    Email string `norm:"unique;notnull"`
    Name  string `norm:"notnull"`
}

func main() {
    // 1. Register database
    dsn := "postgresql://user:pass@localhost:5432/mydb"
    err := norm.Register(dsn).Primary()
    if err != nil {
        log.Fatal(err)
    }
    
    // 2. Register tables with explicit table names
    norm.RegisterTable(User{}, "users")
    
    // 3. Run migrations
    norm.Norm()
}
```

**Output:**
```
============================================================
RUNNING AUTO MIGRATIONS
============================================================

🔄 Auto-migrating Global:primary...
  ✓ Created table 'users' in Global:primary
✅ Auto-migration completed for Global:primary

✅ All auto migrations completed successfully!

============================================================
REGISTRY SUMMARY
============================================================

📊 Database Connection Registry:
  Mode: global
  Total Connection Pools: 1

📋 Table Registry:
  Total Tables Registered: 1

  Table Mappings:
    • User → Global (mode: global)
```

### Drop Tables (Development)

```go
func main() {
    // Register database and tables
    norm.Register(dsn).Primary()
    norm.RegisterTable(User{}, "users")
    norm.RegisterTable(Order{}, "orders")
    
    // Drop all tables (useful for fresh start)
    if err := norm.DropTables(); err != nil {
        log.Fatal("Failed to drop tables:", err)
    }
    
    // Run migrations (recreate tables)
    norm.Norm()
}
```

**⚠️ Warning:** `DropTables()` deletes all data! Only use in development.

---

## Scenarios

### Scenario 1: Initial Schema Creation

**Use Case:** First time running the application.

```go
package main

import (
    "time"
    "github.com/skssmd/norm"
)

// Define models
type User struct {
    ID        uint      `norm:"index;notnull;pk"`
    Email     string    `norm:"unique;notnull"`
    Name      string    `norm:"notnull"`
    CreatedAt time.Time `norm:"notnull;default:NOW()"`
}

type Order struct {
    ID        uint      `norm:"index;notnull;pk"`
    UserID    uint      `norm:"fkey:users.id;ondelete:cascade;notnull"`
    Total     float64   `norm:"notnull"`
    Status    string    `norm:"max:20;default:'pending'"`
    CreatedAt time.Time `norm:"notnull;default:NOW()"`
}

func main() {
    // Setup
    norm.Register(dsn).Primary()
    norm.Table(User{})
    norm.Table(Order{})
    
    // Run migrations
    norm.Norm()
}
```

**What happens:**
1. Creates `users` table with all columns
2. Creates `orders` table with all columns
3. Creates foreign key from `orders.user_id` → `users.id`
4. Creates indexes on primary keys
5. Prints success message

**Generated SQL:**
```sql
-- Users table
CREATE TABLE IF NOT EXISTS users (
    id BIGINT PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_users_id ON users(id);

-- Orders table
CREATE TABLE IF NOT EXISTS orders (
    id BIGINT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    total DOUBLE PRECISION NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_orders_id ON orders(id);
CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders(user_id);
```

---

### Scenario 2: Adding New Columns

**Use Case:** Add new fields to existing models.

**Before:**
```go
type User struct {
    ID        uint      `norm:"index;notnull;pk"`
    Email     string    `norm:"unique;notnull"`
    Name      string    `norm:"notnull"`
    CreatedAt time.Time `norm:"notnull;default:NOW()"`
}
```

**After:**
```go
type User struct {
    ID        uint       `norm:"index;notnull;pk"`
    Email     string     `norm:"unique;notnull"`
    Name      string     `norm:"notnull"`
    Bio       *string    `norm:"text"`           // NEW: nullable bio
    Age       *uint      `norm:""`               // NEW: nullable age
    CreatedAt time.Time  `norm:"notnull;default:NOW()"`
    UpdatedAt *time.Time `norm:"default:NOW()"` // NEW: nullable updated_at
}
```

**Run migrations:**
```go
func main() {
    norm.Register(dsn).Primary()
    norm.RegisterTable(User{}, "users")
    norm.Norm()
}
```

**Output:**
```
🔄 Auto-migrating Global:primary...
  ✓ Updated table 'users' in Global:primary
    + Added column 'bio' to 'users'
    + Added column 'age' to 'users'
    + Added column 'updated_at' to 'users'
✅ Auto-migration completed for Global:primary
```

**Generated SQL:**
```sql
ALTER TABLE users ADD COLUMN bio TEXT;
ALTER TABLE users ADD COLUMN age BIGINT;
ALTER TABLE users ADD COLUMN updated_at TIMESTAMP DEFAULT NOW();
```

---

### Scenario 3: Adding Indexes

**Use Case:** Optimize queries by adding indexes.

**Before:**
```go
type User struct {
    ID    uint   `norm:"index;notnull;pk"`
    Email string `norm:"unique;notnull"`
    Name  string `norm:"notnull"`
}
```

**After:**
```go
type User struct {
    ID    uint   `norm:"index;notnull;pk"`
    Email string `norm:"index;unique;notnull"` // Added index tag
    Name  string `norm:"index;notnull"`        // Added index tag
}
```

**Run migrations:**
```go
norm.Norm()
```

**What happens:**
- Indexes created on `email` and `name` columns
- Existing data preserved
- No downtime

**Generated SQL:**
```sql
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_name ON users(name);
```

---

### Scenario 4: Adding Foreign Keys

**Use Case:** Add relationships between existing tables.

**Before:**
```go
type Order struct {
    ID        uint    `norm:"index;notnull;pk"`
    UserID    uint    `norm:"notnull"` // Just a column
    Total     float64 `norm:"notnull"`
}
```

**After:**
```go
type Order struct {
    ID        uint    `norm:"index;notnull;pk"`
    UserID    uint    `norm:"fkey:users.id;ondelete:cascade;notnull"` // Now a FK
    Total     float64 `norm:"notnull"`
}
```

**Run migrations:**
```go
norm.Norm()
```

**What happens:**
- Foreign key constraint added
- Index created on `user_id`
- Existing data validated (must have valid user_ids)

**Generated SQL:**
```sql
ALTER TABLE orders 
ADD CONSTRAINT fk_orders_user_id 
FOREIGN KEY (user_id) REFERENCES users(id) 
ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders(user_id);
```

---

### Scenario 5: Multi-Shard Migrations

**Use Case:** Migrate multiple independent shards.

```go
package main

import (
    "github.com/skssmd/norm"
)

type User struct {
    ID    uint   `norm:"index;notnull;pk"`
    Email string `norm:"unique;notnull"`
    Name  string `norm:"notnull"`
}

type Analytics struct {
    ID        uint   `norm:"index;notnull;pk"`
    EventType string `norm:"max:100"`
    EventData string `norm:"type:JSONB"`
}

func main() {
    // Register shards
    norm.Register(dsn1).Shard("shard1").Primary()
    norm.Register(dsn2).Shard("shard2").Standalone()
    
    // Register tables to shards
    norm.RegisterTable(User{}, "users").Shard("shard1").Primary()
    norm.RegisterTable(Analytics{}, "analytics").Shard("shard2").Standalone()
    
    // Run migrations (parallel)
    norm.Norm()
}
```

**Output:**
```
============================================================
RUNNING AUTO MIGRATIONS
============================================================

🔄 Auto-migrating Shard:shard1:primary...
  ✓ Created table 'users' in Shard:shard1:primary
✅ Auto-migration completed for Shard:shard1:primary

🔄 Auto-migrating Shard:shard2:standalone1...
  ✓ Created table 'analyticses' in Shard:shard2:standalone1
✅ Auto-migration completed for Shard:shard2:standalone1

✅ All auto migrations completed successfully!
```

**What happens:**
- Migrations run **concurrently** on both shards
- Each shard gets only its assigned tables
- Independent failure domains
- Faster migration time

---

### Scenario 6: Development Workflow

**Use Case:** Iterative development with frequent schema changes.

```go
package main

import (
    "log"
    "os"
    "github.com/skssmd/norm"
)

func main() {
    dsn := os.Getenv("DATABASE_DSN")
    
    // Setup
    norm.Register(dsn).Primary()
    norm.RegisterTable(User{}, "users")
    norm.RegisterTable(Order{}, "orders")
    norm.RegisterTable(Product{}, "products")
    
    // Development: Drop and recreate
    if os.Getenv("ENV") == "development" {
        log.Println("🗑️  Dropping all tables (development mode)")
        if err := norm.DropTables(); err != nil {
            log.Fatal("Failed to drop tables:", err)
        }
    }
    
    // Run migrations
    norm.Norm()
}
```

**Workflow:**
1. Modify struct definitions
2. Run application
3. Tables dropped and recreated
4. Fresh schema every time

**⚠️ Only for development!**

---

### Scenario 7: Production Deployment

**Use Case:** Safe schema updates in production.

```go
package main

import (
    "log"
    "github.com/skssmd/norm"
)

func main() {
    // Production setup
    dsn := os.Getenv("DATABASE_DSN")
    if dsn == "" {
        log.Fatal("DATABASE_DSN not set")
    }
    
    // Register database
    err := norm.Register(dsn).Primary()
    if err != nil {
        log.Fatal("Database connection failed:", err)
    }
    
    // Register all tables
    norm.RegisterTable(User{}, "users")
    norm.RegisterTable(Order{}, "orders")
    norm.RegisterTable(Product{}, "products")
    norm.RegisterTable(Analytics{}, "analytics")
    
    // Run migrations (safe - only adds, never drops)
    log.Println("Running database migrations...")
    norm.Norm()
    log.Println("Migrations completed successfully")
    
    // Start application
    startServer()
}
```

**Production Safety:**
- ✅ Only adds new tables/columns
- ✅ Never drops existing data
- ✅ Idempotent (safe to run multiple times)
- ✅ Concurrent-safe
- ❌ Won't change column types
- ❌ Won't drop columns

---

## Best Practices

### 1. Version Control Your Models

```go
// models/user.go
package models

import "time"

// User represents a user account
// Version: 1.2.0
// Added: bio, age fields
type User struct {
    ID        uint       `norm:"index;notnull;pk"`
    Email     string     `norm:"unique;notnull"`
    Name      string     `norm:"notnull"`
    Bio       *string    `norm:"text"`      // v1.2.0
    Age       *uint      `norm:""`          // v1.2.0
    CreatedAt time.Time  `norm:"notnull;default:NOW()"`
    UpdatedAt *time.Time `norm:"default:NOW()"`
}
```

### 2. Test Migrations Locally First

```bash
# Development
ENV=development go run main.go

# Staging
ENV=staging DATABASE_DSN=$STAGING_DSN go run main.go

# Production (after testing)
ENV=production DATABASE_DSN=$PROD_DSN go run main.go
```

### 3. Backup Before Major Changes

```bash
# Backup database
pg_dump -h localhost -U postgres mydb > backup_$(date +%Y%m%d).sql

# Run migrations
go run main.go

# If issues, restore
psql -h localhost -U postgres mydb < backup_20241209.sql
```

### 4. Use Nullable Fields for New Columns

```go
// ✅ Good - new column is nullable
type User struct {
    ID    uint    `norm:"index;notnull;pk"`
    Email string  `norm:"unique;notnull"`
    Bio   *string `norm:"text"` // NEW: nullable
}

// ❌ Bad - new column is NOT NULL (fails if data exists)
type User struct {
    ID    uint   `norm:"index;notnull;pk"`
    Email string `norm:"unique;notnull"`
    Bio   string `norm:"text;notnull"` // ERROR: existing rows have no value
}
```

### 5. Add Indexes Strategically

```go
// ✅ Good - index on frequently queried columns
type User struct {
    ID    uint   `norm:"index;notnull;pk"`
    Email string `norm:"index;unique;notnull"` // Searched often
    Name  string `norm:"notnull"`              // Not indexed (rarely searched)
}

// ❌ Bad - too many indexes (slows writes)
type User struct {
    ID    uint   `norm:"index;notnull;pk"`
    Email string `norm:"index;unique;notnull"`
    Name  string `norm:"index;notnull"`        // Unnecessary
    Bio   string `norm:"index;text"`           // Unnecessary
}
```

### 6. Monitor Migration Performance

```go
import "time"

func main() {
    start := time.Now()
    
    // Setup
    norm.Register(dsn).Primary()
    norm.Table(User{})
    norm.Table(Order{})
    
    // Run migrations
    norm.Norm()
    
    duration := time.Since(start)
    log.Printf("Migrations completed in %v", duration)
}
```

### 7. Handle Migration Errors

```go
func main() {
    // Register database
    err := norm.Register(dsn).Primary()
    if err != nil {
        log.Fatal("Database connection failed:", err)
    }
    
    // Register tables
    norm.RegisterTable(User{}, "users")
    norm.RegisterTable(Order{}, "orders")
    
    // Run migrations with error handling
    // Note: norm.Norm() calls log.Fatal internally on error
    // For custom error handling, you'd need to use the migrator directly
    norm.Norm()
    
    log.Println("Application started successfully")
}
```

---

## Troubleshooting

### Issue 1: Foreign Key Constraint Fails

**Error:**
```
Failed to create foreign key 'fk_orders_user_id': 
violates foreign key constraint
```

**Cause:** Existing data has invalid foreign key values.

**Solution:**
```sql
-- Find orphaned records
SELECT * FROM orders WHERE user_id NOT IN (SELECT id FROM users);

-- Fix data
DELETE FROM orders WHERE user_id NOT IN (SELECT id FROM users);
-- OR
UPDATE orders SET user_id = NULL WHERE user_id NOT IN (SELECT id FROM users);

-- Then run migrations again
```

### Issue 2: Column Already Exists

**Error:**
```
Failed to add column 'bio': column already exists
```

**Cause:** Migration ran twice or manual column added.

**Solution:**
- This is usually safe to ignore
- Migration will skip existing columns
- Check logs for actual errors

### Issue 3: Type Mismatch

**Error:**
```
column "age" is of type integer but expression is of type bigint
```

**Cause:** Changed Go type (e.g., `int` → `uint`).

**Solution:**
```sql
-- Manual migration required
ALTER TABLE users ALTER COLUMN age TYPE BIGINT;
```

### Issue 4: Index Creation Fails

**Error:**
```
Failed to create index 'idx_users_email': index already exists
```

**Cause:** Index exists with different name.

**Solution:**
- Safe to ignore (index exists)
- Or drop old index:
```sql
DROP INDEX old_index_name;
```

### Issue 5: Shard Not Found

**Error:**
```
No tables registered for Shard:shard1:primary
```

**Cause:** Tables not assigned to shard.

**Solution:**
```go
// Make sure to assign tables to shards
norm.RegisterTable(User{}, "users").Shard("shard1").Primary()
norm.RegisterTable(Order{}, "orders").Shard("shard1").Primary()
```

---

## Summary

### Migration Lifecycle

```
1. Define Models → Go structs with norm tags
2. Register DB   → norm.Register(dsn)
3. Register Tables → norm.Table(Model{})
4. Run Migrations → norm.Norm()
5. Schema Updated → Tables/columns/indexes created
```

### Safety Guarantees

| Operation | Automatic | Safe |
|-----------|-----------|------|
| CREATE TABLE | ✅ Yes | ✅ Yes |
| ADD COLUMN | ✅ Yes | ✅ Yes |
| CREATE INDEX | ✅ Yes | ✅ Yes |
| ADD FOREIGN KEY | ✅ Yes | ⚠️ Check data |
| DROP COLUMN | ❌ No | Manual only |
| ALTER TYPE | ❌ No | Manual only |
| DROP TABLE | ❌ No | Use DropTables() |

### Key Points

- ✅ Migrations are **idempotent** (safe to run multiple times)
- ✅ Migrations are **concurrent** (multiple shards in parallel)
- ✅ Migrations are **safe** (never drop data automatically)
- ✅ Migrations are **automatic** (no SQL files needed)
- ⚠️ Always **test locally** before production
- ⚠️ Always **backup** before major changes

Happy migrating! 🚀
