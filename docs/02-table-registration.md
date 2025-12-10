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

Tables are registered using `norm.Table(model)` which returns a builder for configuration.

---

## Registration Modes

### Global Mode (Automatic)

In global mode, tables are **automatically registered as global** when you call `norm.Table()`:

```go
// No need to call .Global() - it's automatic!
norm.Table(User{})
norm.Table(Order{})
```

### Shard Mode (Explicit)

In shard mode, you **must explicitly assign** tables to shards:

```go
// Must specify shard and role
norm.Table(User{}).Shard("shard1").Primary()
norm.Table(Order{}).Shard("shard1").Primary()
```

---

## Registration Methods

### Global Mode Methods

| Method | Description | When to Use |
|--------|-------------|-------------|
| `Table(model)` | Auto-registers as global | Global mode only |

### Shard Mode Methods

| Method | Description | When to Use |
|--------|-------------|-------------|
| `Table(model).Shard(name).Primary()` | Assign to primary shard | Transactional tables |
| `Table(model).Shard(name).Standalone()` | Assign to standalone shard | Isolated tables |
| `Table(model).Shard(name).Read()` | Assign read role | Read-only tables |
| `Table(model).Shard(name).Write()` | Assign write role | Write-heavy tables |

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
    ID        uint      `norm:"index;notnull;pk"`
    Email     string    `norm:"unique;notnull"`
    Name      string    `norm:"notnull"`
    CreatedAt time.Time `norm:"notnull;default:NOW()"`
}

type Order struct {
    ID        uint      `norm:"index;notnull;pk"`
    UserID    uint      `norm:"fkey:users.id;ondelete:cascade"`
    Total     float64   `norm:"notnull"`
    CreatedAt time.Time `norm:"notnull;default:NOW()"`
}

func main() {
    // Register database
    norm.Register(dsn).Primary()
    
    // Register tables - automatically global
    norm.Table(User{})
    norm.Table(Order{})
    
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
    ID        uint      `norm:"index;notnull;pk"`
    Email     string    `norm:"unique;notnull"`
    Name      string    `norm:"notnull"`
    CreatedAt time.Time `norm:"notnull;default:NOW()"`
}

type Order struct {
    ID        uint      `norm:"index;notnull;pk"`
    UserID    uint      `norm:"fkey:users.id;ondelete:cascade"`
    Total     float64   `norm:"notnull"`
    CreatedAt time.Time `norm:"notnull;default:NOW()"`
}

// Analytics models
type Analytics struct {
    ID        uint      `norm:"index;notnull;pk"`
    UserID    *uint     `norm:"skey:users.id;ondelete:setnull"`
    EventType string    `norm:"max:100"`
    EventData string    `norm:"type:JSONB"`
    CreatedAt time.Time `norm:"notnull;default:NOW()"`
}

type Log struct {
    ID        uint      `norm:"index;notnull;pk"`
    Level     string    `norm:"max:20"`
    Message   string    `norm:"text"`
    CreatedAt time.Time `norm:"notnull;default:NOW()"`
}

func main() {
    // Register shards
    norm.Register(dsn1).Shard("transactional").Primary()
    norm.Register(dsn2).Shard("analytics").Standalone()
    
    // Register transactional tables to shard1
    err := norm.Table(User{}).Shard("transactional").Primary()
    if err != nil {
        log.Fatal("Failed to register User:", err)
    }
    
    err = norm.Table(Order{}).Shard("transactional").Primary()
    if err != nil {
        log.Fatal("Failed to register Order:", err)
    }
    
    // Register analytics tables to shard2
    err = norm.Table(Analytics{}).Shard("analytics").Standalone()
    if err != nil {
        log.Fatal("Failed to register Analytics:", err)
    }
    
    err = norm.Table(Log{}).Shard("analytics").Standalone()
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
    ID        uint   `norm:"index;notnull;pk"`
    TenantID  string `norm:"index;notnull;max:50"`
    Email     string `norm:"unique;notnull"`
    Name      string `norm:"notnull"`
}

type Order struct {
    ID        uint   `norm:"index;notnull;pk"`
    TenantID  string `norm:"index;notnull;max:50"`
    UserID    uint   `norm:"fkey:users.id;ondelete:cascade"`
    Total     float64 `norm:"notnull"`
}

func main() {
    // Register shards for different tenant groups
    tenantShards := map[string]string{
        "shard_us_east":  "postgresql://user:pass@us-east-db:5432/tenants",
        "shard_us_west":  "postgresql://user:pass@us-west-db:5432/tenants",
        "shard_eu":       "postgresql://user:pass@eu-db:5432/tenants",
    }
    
    // Register all shards
    for shardName, dsn := range tenantShards {
        err := norm.Register(dsn).Shard(shardName).Primary()
        if err != nil {
            log.Fatal(fmt.Sprintf("Failed to register %s:", shardName), err)
        }
    }
    
    // Register tables to all shards
    // (In practice, you'd route to the correct shard based on tenant)
    for shardName := range tenantShards {
        err := norm.Table(User{}).Shard(shardName).Primary()
        if err != nil {
            log.Fatal(err)
        }
        
        err = norm.Table(Order{}).Shard(shardName).Primary()
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
    ID        uint      `norm:"index;notnull;pk"`
    UserID    uint      `norm:"fkey:users.id"`
    Total     float64   `norm:"notnull"`
    Status    string    `norm:"max:20"`
    CreatedAt time.Time `norm:"notnull;default:NOW()"`
}

// Heavy read table
type Analytics struct {
    ID        uint      `norm:"index;notnull;pk"`
    EventType string    `norm:"max:100"`
    EventData string    `norm:"type:JSONB"`
    CreatedAt time.Time `norm:"notnull;default:NOW()"`
}

// Balanced read/write
type User struct {
    ID        uint      `norm:"index;notnull;pk"`
    Email     string    `norm:"unique;notnull"`
    Name      string    `norm:"notnull"`
    CreatedAt time.Time `norm:"notnull;default:NOW()"`
}

func main() {
    // Register shard with different pools
    norm.Register(dsnPrimary).Shard("main").Primary()
    
    // Assign tables based on access patterns
    // User: balanced access → primary pool
    err := norm.Table(User{}).Shard("main").Primary()
    if err != nil {
        log.Fatal(err)
    }
    
    // Order: write-heavy → write role
    err = norm.Table(Order{}).Shard("main").Write()
    if err != nil {
        log.Fatal(err)
    }
    
    // Analytics: read-heavy → read role
    err = norm.Table(Analytics{}).Shard("main").Read()
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
    
    // Register shards (from environment or config)
    norm.Register(dsn1).Shard("shard1").Primary()
    norm.Register(dsn2).Shard("shard2").Standalone()
    
    // Register tables based on configuration
    for _, tableConfig := range config.Tables {
        model, exists := modelRegistry[tableConfig.Model]
        if !exists {
            log.Printf("Warning: Unknown model %s", tableConfig.Model)
            continue
        }
        
        // Register based on role
        switch tableConfig.Role {
        case "primary":
            err = norm.Table(model).Shard(tableConfig.Shard).Primary()
        case "standalone":
            err = norm.Table(model).Shard(tableConfig.Shard).Standalone()
        case "read":
            err = norm.Table(model).Shard(tableConfig.Shard).Read()
        case "write":
            err = norm.Table(model).Shard(tableConfig.Shard).Write()
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
norm.Table(User{}).Shard("shard1").Primary()

// ✅ Good - handling errors
err := norm.Table(User{}).Shard("shard1").Primary()
if err != nil {
    log.Fatal("Failed to register User:", err)
}
```

### 2. Consistent Naming

```go
// ✅ Good - consistent shard names
norm.Table(User{}).Shard("transactional").Primary()
norm.Table(Order{}).Shard("transactional").Primary()

// ❌ Bad - inconsistent names
norm.Table(User{}).Shard("transactional").Primary()
norm.Table(Order{}).Shard("trans").Primary() // Typo!
```

### 3. Group Related Tables

```go
// ✅ Good - related tables in same shard
norm.Table(User{}).Shard("users").Primary()
norm.Table(UserProfile{}).Shard("users").Primary()
norm.Table(UserSettings{}).Shard("users").Primary()

// Analytics in separate shard
norm.Table(Analytics{}).Shard("analytics").Standalone()
norm.Table(Log{}).Shard("analytics").Standalone()
```

### 4. Document Table Placement

```go
// Document why tables are placed in specific shards
// User and Order: High transaction volume, need ACID guarantees
err := norm.Table(User{}).Shard("transactional").Primary()
err = norm.Table(Order{}).Shard("transactional").Primary()

// Analytics: Heavy reads, eventual consistency OK
err = norm.Table(Analytics{}).Shard("analytics").Standalone()
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
    switch role {
    case "primary":
        return norm.Table(model).Shard(shard).Primary()
    case "standalone":
        return norm.Table(model).Shard(shard).Standalone()
    default:
        return fmt.Errorf("unknown role: %s", role)
    }
}
```

---

## Summary

| Mode | Registration | Use Case |
|------|-------------|----------|
| **Global** | `norm.Table(model)` | Single database |
| **Shard Primary** | `norm.Table(model).Shard(name).Primary()` | Transactional data |
| **Shard Standalone** | `norm.Table(model).Shard(name).Standalone()` | Isolated data |
| **Shard Read** | `norm.Table(model).Shard(name).Read()` | Read-heavy tables |
| **Shard Write** | `norm.Table(model).Shard(name).Write()` | Write-heavy tables |

Choose the registration method that matches your database architecture and access patterns.
