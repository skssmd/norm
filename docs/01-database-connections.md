# Database Connection Registration

This guide covers how to register and manage database connections in Norm ORM.

## Table of Contents
- [Overview](#overview)
- [Connection Types](#connection-types)
- [Registration Syntax](#registration-syntax)
- [Scenarios](#scenarios)
- [Best Practices](#best-practices)

---

## Overview

Norm supports three connection modes:
1. **Global Mode** - Single database with optional replicas
2. **Read/Write Split Mode** - Separate read and write pools
3. **Shard Mode** - Multiple independent database shards

All connections are registered through the `norm.Register(dsn)` function, which returns a builder for fluent API chaining.

---

## Connection Types

### Global Mode Connections

| Method | Description | Use Case |
|--------|-------------|----------|
| `Primary()` | Main database connection | All reads and writes |
| `Replica()` | Replica connection for failover | High availability fallback |

### Read/Write Split Connections

| Method | Description | Use Case |
|--------|-------------|----------|
| `Write()` | Write-only pool | All INSERT/UPDATE/DELETE |
| `Read()` | Read-only pool | All SELECT queries, load balancing reads |

### Shard Mode Connections

| Method | Description | Use Case |
|--------|-------------|----------|
| `Shard(name).Primary()` | Primary shard pool | Transactional data |
| `Shard(name).Standalone(tables...)` | Standalone shard pool | Isolated data (analytics, logs) |

---

## Registration Syntax

### Basic Syntax

```go
import "github.com/skssmd/norm"

// Register a connection
err := norm.Register(dsn).Primary()
if err != nil {
    log.Fatal(err)
}
```

### Connection String Format

```go
dsn := "postgresql://username:password@host:port/database"
```

---

## Scenarios

### Scenario 1: Single Database (Monolith)

**Use Case:** Small to medium applications with a single PostgreSQL database.

```go
package main

import (
    "log"
    "github.com/skssmd/norm"
)

func main() {
    // Connection string
    dsn := "postgresql://postgres:password@localhost:5432/myapp"
    
    // Register primary database
    err := norm.Register(dsn).Primary()
    if err != nil {
        log.Fatal("Failed to register primary:", err)
    }
    
    log.Println("✅ Database connected successfully")
}
```

**What happens:**
- Single connection pool created
- All queries use this connection
- Simple and straightforward setup

---

### Scenario 2: Primary with Replicas (High Availability)

**Use Case:** Applications that need high availability and automatic failover.

```go
package main

import (
    "log"
    "github.com/skssmd/norm"
)

func main() {
    // Primary database (for all operations)
    dsnPrimary := "postgresql://postgres:password@primary-db:5432/myapp"
    
    // Replicas (for failover/high availability)
    dsnReplica1 := "postgresql://postgres:password@replica1-db:5432/myapp"
    dsnReplica2 := "postgresql://postgres:password@replica2-db:5432/myapp"
    
    // Register primary connection (handles all operations)
    err := norm.Register(dsnPrimary).Primary()
    if err != nil {
        log.Fatal("Failed to register primary:", err)
    }
    
    // Register replicas (for automatic failover)
    err = norm.Register(dsnReplica1).Replica()
    if err != nil {
        log.Fatal("Failed to register replica 1:", err)
    }
    
    err = norm.Register(dsnReplica2).Replica()
    if err != nil {
        log.Fatal("Failed to register replica 2:", err)
    }
    
    log.Println("✅ Primary and replicas connected successfully")
}
```

**What happens:**
- Primary handles ALL operations (reads and writes)
- Replicas are used only when primary is unavailable
- Automatic failover to replicas if primary goes down

**Benefits:**
- ✅ High availability and fault tolerance
- ✅ Automatic failover on primary failure
- ✅ Zero downtime during primary maintenance
- ✅ Data consistency (no replication lag issues)

---

### Scenario 3: Read/Write Split

**Use Case:** Separate databases for read and write operations (CQRS pattern).

```go
package main

import (
    "log"
    "github.com/skssmd/norm"
)

func main() {
    // Write database (master)
    dsnWrite := "postgresql://postgres:password@write-db:5432/myapp"
    
    // Read databases (can be different servers)
    dsnRead1 := "postgresql://postgres:password@read-db1:5432/myapp"
    dsnRead2 := "postgresql://postgres:password@read-db2:5432/myapp"
    
    // Register write pool (only one allowed)
    err := norm.Register(dsnWrite).Write()
    if err != nil {
        log.Fatal("Failed to register write pool:", err)
    }
    
    // Register multiple read pools
    err = norm.Register(dsnRead1).Read()
    if err != nil {
        log.Fatal("Failed to register read pool 1:", err)
    }
    
    err = norm.Register(dsnRead2).Read()
    if err != nil {
        log.Fatal("Failed to register read pool 2:", err)
    }
    
    log.Println("✅ Read/Write pools connected successfully")
}
```

**What happens:**
- All INSERT/UPDATE/DELETE → Write pool
- All SELECT → Read pools (load balanced)
- Read pools can be eventually consistent

**Benefits:**
- ✅ Complete separation of concerns
- ✅ Optimize each database for its workload
- ✅ Scale reads and writes independently
- ✅ Supports CQRS architecture

---

### Scenario 4: Multi-Tenant Sharding

**Use Case:** SaaS applications with tenant-based data isolation.

```go
package main

import (
    "log"
    "github.com/skssmd/norm"
)

func main() {
    // Shard 1: Tenants A-M
    dsn1 := "postgresql://postgres:password@shard1-db:5432/tenants_am"
    
    // Shard 2: Tenants N-Z
    dsn2 := "postgresql://postgres:password@shard2-db:5432/tenants_nz"
    
    // Register shard 1 with primary role
    err := norm.Register(dsn1).Shard("shard1").Primary()
    if err != nil {
        log.Fatal("Failed to register shard1:", err)
    }
    
    // Register shard 2 with primary role
    err = norm.Register(dsn2).Shard("shard2").Primary()
    if err != nil {
        log.Fatal("Failed to register shard2:", err)
    }
    
    log.Println("✅ Shards connected successfully")
}
```

**What happens:**
- Each shard is an independent database
- Tables are assigned to shards via table registration (see Table Registration docs)
- Application routes queries to correct shard automatically

**Benefits:**
- ✅ Data isolation per tenant/region
- ✅ Horizontal scaling
- ✅ Independent failure domains
- ✅ Compliance with data residency requirements

---

### Scenario 5: Hybrid Sharding with Standalone Pools

**Use Case:** Primary shards for transactional data + standalone shards for analytics/logs.

```go
package main

import (
    "log"
    "github.com/skssmd/norm"
)

func main() {
    // Primary shard for transactional data
    dsnPrimary := "postgresql://postgres:password@primary-db:5432/transactions"
    
    // Standalone shard for analytics (separate database)
    dsnAnalytics := "postgresql://postgres:password@analytics-db:5432/analytics"
    
    // Standalone shard for logs (separate database)
    dsnLogs := "postgresql://postgres:password@logs-db:5432/logs"
    
    // Register primary shard
    err := norm.Register(dsnPrimary).Shard("primary").Primary()
    if err != nil {
        log.Fatal("Failed to register primary shard:", err)
    }
    
    // Register standalone analytics shard
    // Specify table names for automatic routing
    err = norm.Register(dsnAnalytics).Shard("analytics").Standalone("analytics")
    if err != nil {
        log.Fatal("Failed to register analytics shard:", err)
    }
    
    // Register standalone logs shard
    err = norm.Register(dsnLogs).Shard("logs").Standalone("logs")
    if err != nil {
        log.Fatal("Failed to register logs shard:", err)
    }
    
    log.Println("✅ All shards connected successfully")
}
```

**What happens:**
- Primary shard handles OLTP (transactional) data
- Standalone shards handle OLAP (analytical) data
- Each shard can be optimized for its workload
- Standalone pools are registered with table names for automatic routing

**Benefits:**
- ✅ Separate transactional and analytical workloads
- ✅ Optimize each database independently
- ✅ Prevent analytics queries from impacting transactions
- ✅ Different retention policies per shard

---

## Best Practices

### 1. Environment Variables for DSNs

```go
import "os"

func main() {
    // Load from environment
    dsn := os.Getenv("DATABASE_DSN")
    if dsn == "" {
        log.Fatal("DATABASE_DSN not set")
    }
    
    err := norm.Register(dsn).Primary()
    if err != nil {
        log.Fatal(err)
    }
}
```

### 2. Error Handling

```go
// Always check for errors
err := norm.Register(dsn).Primary()
if err != nil {
    log.Fatal("Database connection failed:", err)
}
```

### 3. Connection Pooling

Norm uses `pgxpool` internally, which handles connection pooling automatically:

```go
// Connection pool is managed automatically
// No need to manually open/close connections
err := norm.Register(dsn).Primary()
// Pool is ready to use
```

### 4. Mode Validation

Norm prevents mixing incompatible connection types:

```go
// ❌ This will fail - can't mix global and shard modes
norm.Register(dsn1).Primary()
norm.Register(dsn2).Shard("shard1").Primary() // ERROR

// ✅ This works - consistent mode
norm.Register(dsn1).Shard("shard1").Primary()
norm.Register(dsn2).Shard("shard2").Primary()
```

### 5. Shard Validation

A shard cannot be both primary and standalone:

```go
// ❌ This will fail
norm.Register(dsn).Shard("shard1").Primary()
norm.Register(dsn).Shard("shard1").Standalone() // ERROR

// ✅ This works - different shards
norm.Register(dsn1).Shard("shard1").Primary()
norm.Register(dsn2).Shard("shard2").Standalone()
```


---

## Summary

| Mode | Use Case | Complexity | Scalability |
|------|----------|------------|-------------|
| **Global** | Small apps, prototypes | Low | Limited |
| **Primary + Replicas** | Read-heavy apps | Medium | High (reads) |
| **Read/Write Split** | CQRS, separate workloads | Medium | High (both) |
| **Sharding** | Multi-tenant, large scale | High | Very High |

Choose the mode that best fits your application's needs and scale requirements.

---

## Next Steps

- Learn about [Model Definition](./02-model-definition.md) for struct tags and field mapping
- Explore [Table Registration](./03-table-registration.md) to assign tables to shards
- See [Migrations](./04-migrations.md) for schema management
