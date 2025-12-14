# Table Registration

This guide covers how to register tables and map them to database connections in Norm ORM.

## Table of Contents
- [Overview](#overview)
- [Registration Modes](#registration-modes)
- [Registration Methods](#registration-methods)
- [Scenarios](#scenarios)
- [Best Practices](#best-practices)

---

## Overview

Table registration tells Norm:
1. Which models should be migrated
2. Which database pool(s) to use for each table
3. What role each table has (primary, read, write, standalone)

Tables are registered using `norm.RegisterTable(model, "tablename")` which returns a builder for configuration.

---

## Registration Modes

### Global Mode (Automatic)

In global mode, tables are **automatically registered as global** when you call `norm.RegisterTable()`:

```go
// No need to call .Global() - it's automatic!
norm.RegisterTable(User{}, "users")
norm.RegisterTable(Order{}, "orders")
```

### Shard Mode (Explicit)

In shard mode, you **must explicitly assign** tables to shards:

```go
// Must specify shard and role
norm.RegisterTable(User{}, "users").Primary("shard1")
norm.RegisterTable(Order{}, "orders").Primary("shard1")
```

---

## Registration Methods

### Global Mode Methods

| Method | Description | When to Use |
|--------|-------------|-------------|
| `RegisterTable(model, "name")` | Auto-registers as global | Global mode only |

### Shard Mode Methods

| Method | Description | When to Use |
|--------|-------------|-------------|
| `RegisterTable(model, "name").Primary("shard")` | Assign to primary shard | Transactional tables |
| `RegisterTable(model, "name").Standalone("shard")` | Assign to standalone shard | Isolated tables |
| `RegisterTable(model, "name").Read("shard")` | Assign read role | Read-only tables |
| `RegisterTable(model, "name").Write("shard")` | Assign write role | Write-heavy tables |

---

## Scenarios

### Scenario 1: Simple Global Registration

**Use Case:** Single database application with all tables in one place.

```go
package main

import (
    "time"
    "github.com/skssmd/norm"
)

// Define your models
type User struct {
    ID        uint      `norm:"index;notnull;pk;auto"`
    Email     string    `norm:"name:useremail;unique;notnull"`
    Name      string    `norm:"name:fullname;notnull"`
    CreatedAt time.Time `norm:"notnull;default:NOW()"`
}

type Order struct {
    ID        uint      `norm:"index;notnull;pk;auto"`
    UserID    uint      `norm:"index;notnull;fkey:users.id;ondelete:cascade"`
    Total     float64   `norm:"notnull"`
    CreatedAt time.Time `norm:"notnull;default:NOW()"`
}

func main() {
    // Assumes database is already registered via norm.Register(dsn).Primary()
    // See Database Connections documentation
    
    // Register tables - automatically global
    norm.RegisterTable(User{}, "users")
    norm.RegisterTable(Order{}, "orders")
    
    // Run migrations
    norm.Norm()
}
```

**What happens:**
- Both tables created in the primary database
- No explicit `.Global()` call needed
- Simple and clean API

---

### Scenario 2: Multi-Shard Table Distribution

**Use Case:** Distribute tables across multiple shards based on data type.

```go
package main

import (
    "time"
    "github.com/skssmd/norm"
)

// User data models
type User struct {
    ID        uint      `norm:"index;notnull;pk;auto"`
    Email     string    `norm:"name:useremail;unique;notnull"`
    Name      string    `norm:"name:fullname;notnull"`
    CreatedAt time.Time `norm:"notnull;default:NOW()"`
}

type Order struct {
    ID        uint      `norm:"index;notnull;pk;auto"`
    UserID    uint      `norm:"index;notnull;fkey:users.id;ondelete:cascade"`
    Total     float64   `norm:"notnull"`
    CreatedAt time.Time `norm:"notnull;default:NOW()"`
}

// Analytics models
type Analytics struct {
    ID        uint      `norm:"index;notnull;pk;auto"`
    UserID    *uint     `norm:"skey:users.id;ondelete:setnull"`
    EventType string    `norm:"index;notnull;max:100"`
    EventData string    `norm:"type:JSONB"`
    CreatedAt time.Time `norm:"notnull;default:NOW()"`
}

type Log struct {
    ID        uint      `norm:"index;notnull;pk;auto"`
    Level     string    `norm:"index;notnull;max:20"`
    Message   string    `norm:"text;notnull"`
    CreatedAt time.Time `norm:"notnull;default:NOW()"`
}

func main() {
    // Assumes shards are already registered:
    // norm.Register(dsn1).Shard("transactional").Primary()
    // norm.Register(dsn2).Shard("analytics").Standalone()
    // See Database Connections documentation
    
    // Register transactional tables to shard1
    err := norm.RegisterTable(User{}, "users").Primary("transactional")
    if err != nil {
        log.Fatal("Failed to register User:", err)
    }
    
    err = norm.RegisterTable(Order{}, "orders").Primary("transactional")
    if err != nil {
        log.Fatal("Failed to register Order:", err)
    }
    
    // Register analytics tables to shard2
    err = norm.RegisterTable(Analytics{}, "analytics").Standalone("analytics")
    if err != nil {
        log.Fatal("Failed to register Analytics:", err)
    }
    
    err = norm.RegisterTable(Log{}, "logs").Standalone("analytics")
    if err != nil {
        log.Fatal("Failed to register Log:", err)
    }
    
    // Run migrations
    norm.Norm()
}
```

**What happens:**
- User and Order tables → `transactional` shard
- Analytics and Log tables → `analytics` shard
- Each shard migrated independently
- Tables isolated by workload type

**Benefits:**
- ✅ Separate OLTP and OLAP workloads
- ✅ Independent scaling
- ✅ Optimized for specific use cases
- ✅ Better resource utilization

---

### Scenario 3: Tenant-Based Sharding

**Use Case:** Multi-tenant SaaS with tenant data isolation.

```go
package main

import (
    "fmt"
    "github.com/skssmd/norm"
)

// Shared models across all tenants
type User struct {
    ID        uint   `norm:"index;notnull;pk;auto"`
    TenantID  string `norm:"index;notnull;max:50"`
    Email     string `norm:"name:useremail;unique;notnull"`
    Name      string `norm:"name:fullname;notnull"`
}

type Order struct {
    ID        uint   `norm:"index;notnull;pk;auto"`
    TenantID  string `norm:"index;notnull;max:50"`
    UserID    uint   `norm:"index;notnull;fkey:users.id;ondelete:cascade"`
    Total     float64 `norm:"notnull"`
}

func main() {
    // Assumes shards are already registered:
    // norm.Register(dsn).Shard("shard_us_east").Primary()
    // norm.Register(dsn).Shard("shard_us_west").Primary()
    // norm.Register(dsn).Shard("shard_eu").Primary()
    // See Database Connections documentation
    
    tenantShards := []string{"shard_us_east", "shard_us_west", "shard_eu"}
    
    // Register tables to all shards
    // (In practice, you'd route to the correct shard based on tenant)
    for _, shardName := range tenantShards {
        err := norm.RegisterTable(User{}, "users").Primary(shardName)
        if err != nil {
            log.Fatal(err)
        }
        
        err = norm.RegisterTable(Order{}, "orders").Primary(shardName)
        if err != nil {
            log.Fatal(err)
        }
    }
    
    // Run migrations on all shards
    norm.Norm()
}
```

**What happens:**
- Same schema replicated across all shards
- Each tenant's data lives in one shard
- Application routes queries based on tenant
- Data isolation and compliance

**Benefits:**
- ✅ Data residency compliance (EU data in EU)
- ✅ Tenant isolation
- ✅ Horizontal scaling
- ✅ Independent failure domains

---

### Scenario 4: Read/Write Role Assignment

**Use Case:** Optimize table access patterns with role-based routing.

```go
package main

import (
    "github.com/skssmd/norm"
)

// Heavy write table
type Order struct {
    ID        uint      `norm:"index;notnull;pk;auto"`
    UserID    uint      `norm:"index;notnull;fkey:users.id"`
    Total     float64   `norm:"notnull"`
    Status    string    `norm:"max:20;default:'pending'"`
    CreatedAt time.Time `norm:"notnull;default:NOW()"`
}

// Heavy read table
type Analytics struct {
    ID        uint      `norm:"index;notnull;pk;auto"`
    EventType string    `norm:"index;notnull;max:100"`
    EventData string    `norm:"type:JSONB"`
    CreatedAt time.Time `norm:"notnull;default:NOW()"`
}

// Balanced read/write
type User struct {
    ID        uint      `norm:"index;notnull;pk;auto"`
    Email     string    `norm:"name:useremail;unique;notnull"`
    Name      string    `norm:"name:fullname;notnull"`
    CreatedAt time.Time `norm:"notnull;default:NOW()"`
}

func main() {
    // Assumes shard is already registered:
    // norm.Register(dsnPrimary).Shard("main").Primary()
    // See Database Connections documentation
    
    // Assign tables based on access patterns
    // User: balanced access → primary pool
    err := norm.RegisterTable(User{}, "users").Primary("main")
    if err != nil {
        log.Fatal(err)
    }
    
    // Order: write-heavy → write role
    err = norm.RegisterTable(Order{}, "orders").Write("main")
    if err != nil {
        log.Fatal(err)
    }
    
    // Analytics: read-heavy → read role
    err = norm.RegisterTable(Analytics{}, "analytics").Read("main")
    if err != nil {
        log.Fatal(err)
    }
    
    // Run migrations
    norm.Norm()
}
```

**What happens:**
- User queries → primary pool (balanced)
- Order queries → write pool (optimized for writes)
- Analytics queries → read pool (optimized for reads)
- Query router uses role metadata

**Benefits:**
- ✅ Optimized query routing
- ✅ Better resource utilization
- ✅ Reduced contention
- ✅ Improved performance

---

### Scenario 5: Dynamic Table Registration

**Use Case:** Register tables based on runtime configuration.

```go
package main

import (
    "encoding/json"
    "io/ioutil"
    "github.com/skssmd/norm"
)

// Configuration structure
type TableConfig struct {
    Model     string `json:"model"`
    Shard     string `json:"shard"`
    Role      string `json:"role"`
}

type Config struct {
    Tables []TableConfig `json:"tables"`
}

// Model registry
var modelRegistry = map[string]interface{}{
    "User":      User{},
    "Order":     Order{},
    "Analytics": Analytics{},
    "Log":       Log{},
}

func main() {
    // Load configuration from file
    configData, err := ioutil.ReadFile("table_config.json")
    if err != nil {
        log.Fatal("Failed to read config:", err)
    }
    
    var config Config
    err = json.Unmarshal(configData, &config)
    if err != nil {
        log.Fatal("Failed to parse config:", err)
    }
    
    // Assumes shards are already registered:
    // norm.Register(dsn1).Shard("shard1").Primary()
    // norm.Register(dsn2).Shard("shard2").Standalone()
    // See Database Connections documentation
    
    // Register tables based on configuration
    for _, tableConfig := range config.Tables {
        model, exists := modelRegistry[tableConfig.Model]
        if !exists {
            log.Printf("Warning: Unknown model %s", tableConfig.Model)
            continue
        }
        
        // Register based on role
        // Note: You'd need to get table name from config or model
        tableName := strings.ToLower(tableConfig.Model) + "s" // Simple pluralization
        
        switch tableConfig.Role {
        case "primary":
            err = norm.RegisterTable(model, tableName).Primary(tableConfig.Shard)
        case "standalone":
            err = norm.RegisterTable(model, tableName).Standalone(tableConfig.Shard)
        case "read":
            err = norm.RegisterTable(model, tableName).Read(tableConfig.Shard)
        case "write":
            err = norm.RegisterTable(model, tableName).Write(tableConfig.Shard)
        default:
            log.Printf("Warning: Unknown role %s", tableConfig.Role)
            continue
        }
        
        if err != nil {
            log.Fatal(fmt.Sprintf("Failed to register %s:", tableConfig.Model), err)
        }
        
        log.Printf("✓ Registered %s to %s (%s)", 
            tableConfig.Model, tableConfig.Shard, tableConfig.Role)
    }
    
    // Run migrations
    norm.Norm()
}
```

**Example `table_config.json`:**

```json
{
  "tables": [
    {
      "model": "User",
      "shard": "shard1",
      "role": "primary"
    },
    {
      "model": "Order",
      "shard": "shard1",
      "role": "primary"
    },
    {
      "model": "Analytics",
      "shard": "shard2",
      "role": "standalone"
    },
    {
      "model": "Log",
      "shard": "shard2",
      "role": "standalone"
    }
  ]
}
```

**What happens:**
- Table registration driven by configuration
- Easy to change without code changes
- Supports dynamic environments
- Centralized table management

**Benefits:**
- ✅ Configuration-driven deployment
- ✅ Easy to modify table placement
- ✅ Supports multiple environments
- ✅ No code changes needed

---

## Best Practices

### 1. Always Check Errors

```go
// ❌ Bad - ignoring errors
norm.RegisterTable(User{}, "users").Primary("shard1")

// ✅ Good - handling errors
err := norm.RegisterTable(User{}, "users").Primary("shard1")
if err != nil {
    log.Fatal("Failed to register User:", err)
}
```

### 2. Consistent Naming

```go
// ✅ Good - consistent shard names
norm.RegisterTable(User{}, "users").Primary("transactional")
norm.RegisterTable(Order{}, "orders").Primary("transactional")

// ❌ Bad - inconsistent names
norm.RegisterTable(User{}, "users").Primary("transactional")
norm.RegisterTable(Order{}, "orders").Primary("trans") // Typo!
```

### 3. Group Related Tables

```go
// ✅ Good - related tables in same shard
norm.RegisterTable(User{}, "users").Primary("users")
norm.RegisterTable(UserProfile{}, "user_profiles").Primary("users")
norm.RegisterTable(UserSettings{}, "user_settings").Primary("users")

// Analytics in separate shard
norm.RegisterTable(Analytics{}, "analytics").Standalone("analytics")
norm.RegisterTable(Log{}, "logs").Standalone("analytics")
```

### 4. Document Table Placement

```go
// Document why tables are placed in specific shards
// User and Order: High transaction volume, need ACID guarantees
err := norm.RegisterTable(User{}, "users").Primary("transactional")
err = norm.RegisterTable(Order{}, "orders").Primary("transactional")

// Analytics: Heavy reads, eventual consistency OK
err = norm.RegisterTable(Analytics{}, "analytics").Standalone("analytics")
```

### 5. Validation

```go
// Validate shard exists before registering tables
registeredShards := []string{"shard1", "shard2"}

func registerTable(model interface{}, shard string, role string) error {
    // Check if shard is registered
    if !contains(registeredShards, shard) {
        return fmt.Errorf("shard %s not registered", shard)
    }
    
    // Register based on role
    // Note: You'd need table name as parameter
    tableName := "table_name" // Should be passed as parameter
    
    switch role {
    case "primary":
        return norm.RegisterTable(model, tableName).Primary(shard)
    case "standalone":
        return norm.RegisterTable(model, tableName).Standalone(shard)
    default:
        return fmt.Errorf("unknown role: %s", role)
    }
}
```

---

## Summary

| Mode | Registration | Use Case |
|------|-------------|----------|
| **Global** | `norm.RegisterTable(model, "name")` | Single database |
| **Shard Primary** | `norm.RegisterTable(model, "name").Primary("shard")` | Transactional data |
| **Shard Standalone** | `norm.RegisterTable(model, "name").Standalone("shard")` | Isolated data |
| **Shard Read** | `norm.RegisterTable(model, "name").Read("shard")` | Read-heavy tables |
| **Shard Write** | `norm.RegisterTable(model, "name").Write("shard")` | Write-heavy tables |

Choose the registration method that matches your database architecture and access patterns.
