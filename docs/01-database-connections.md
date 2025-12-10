# Database Connection Registration

This guide covers how to register and manage database connections in Norm ORM.

## Table of Contents
- [Overview](#overview)
- [Connection Types](#connection-types)
- [Registration Methods](#registration-methods)
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
| `Primary()` | Main database connection | Single database setup |
| `Replica()` | Read replica connection | Downtime fallback |

### Read/Write Split Connections

| Method | Description | Use Case |
|--------|-------------|----------|
| `Write()` | Write-only pool | All INSERT/UPDATE/DELETE |
| `Read()` | Read-only pool | All SELECT queries,Load balancing reads |

### Shard Mode Connections

| Method | Description | Use Case |
|--------|-------------|----------|
| `Shard(name).Primary()` | Primary shard pool | Transactional data |
| `Shard(name).Standalone()` | Standalone shard pool | Isolated data |

---

## Registration Methods

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
    
    // Tables are automatically registered as global
    norm.Table(User{})
    norm.Table(Order{})
    
    // Run migrations
    norm.Norm()
}
```

**What happens:**
- Single connection pool created
- All queries use this connection
- Simple and straightforward setup

---

### Scenario 2: Primary with Read Replicas

**Use Case:** Applications with high read traffic that need to scale reads.

```go
package main

import (
    "log"
    "github.com/skssmd/norm"
)

func main() {
    // Primary database (for writes)
    dsnPrimary := "postgresql://postgres:password@primary-db:5432/myapp"
    
    // Read replicas (for reads)
    dsnReplica1 := "postgresql://postgres:password@replica1-db:5432/myapp"
    dsnReplica2 := "postgresql://postgres:password@replica2-db:5432/myapp"
    
    // Register primary connection
    err := norm.Register(dsnPrimary).Primary()
    if err != nil {
        log.Fatal("Failed to register primary:", err)
    }
    
    // Register read replicas
    err = norm.Register(dsnReplica1).Replica()
    if err != nil {
        log.Fatal("Failed to register replica 1:", err)
    }
    
    err = norm.Register(dsnReplica2).Replica()
    if err != nil {
        log.Fatal("Failed to register replica 2:", err)
    }
    
    // Register tables
    norm.Table(User{})
    norm.Table(Order{})
    norm.Table(Product{})
    
    // Run migrations (only on primary)
    norm.Norm()
}
```

**What happens:**
- Writes go to primary database
- Reads are load-balanced across replicas
- Migrations run only on primary
- Automatic failover to primary if replicas unavailable

**Benefits:**
- ✅ Scales read traffic horizontally
- ✅ Reduces load on primary database
- ✅ Improves query performance
- ✅ High availability for reads

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
    
    // Read databases (can be different servers or even different DB engines)
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
    
    // Register tables
    norm.Table(User{})
    norm.Table(Order{})
    norm.Table(Analytics{})
    
    // Run migrations
    norm.Norm()
}
```

**What happens:**
- All INSERT/UPDATE/DELETE → Write pool
- All SELECT → Read pools (load balanced)
- Migrations run on write pool
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
    
    // Register tables to specific shards
    // User and Order tables go to shard1
    err = norm.Table(User{}).Shard("shard1").Primary()
    if err != nil {
        log.Fatal("Failed to register User to shard1:", err)
    }
    
    err = norm.Table(Order{}).Shard("shard1").Primary()
    if err != nil {
        log.Fatal("Failed to register Order to shard1:", err)
    }
    
    // Analytics table goes to shard2
    err = norm.Table(Analytics{}).Shard("shard2").Primary()
    if err != nil {
        log.Fatal("Failed to register Analytics to shard2:", err)
    }
    
    // Run migrations (runs on all shards)
    norm.Norm()
}
```

**What happens:**
- Each shard is an independent database
- Tables are explicitly assigned to shards
- Migrations run on each shard independently
- Application routes queries to correct shard

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
    err = norm.Register(dsnAnalytics).Shard("analytics").Standalone()
    if err != nil {
        log.Fatal("Failed to register analytics shard:", err)
    }
    
    // Register standalone logs shard
    err = norm.Register(dsnLogs).Shard("logs").Standalone()
    if err != nil {
        log.Fatal("Failed to register logs shard:", err)
    }
    
    // Register transactional tables to primary shard
    err = norm.Table(User{}).Shard("primary").Primary()
    if err != nil {
        log.Fatal(err)
    }
    
    err = norm.Table(Order{}).Shard("primary").Primary()
    if err != nil {
        log.Fatal(err)
    }
    
    // Register analytics tables to analytics shard
    err = norm.Table(Analytics{}).Shard("analytics").Standalone()
    if err != nil {
        log.Fatal(err)
    }
    
    // Register log tables to logs shard
    err = norm.Table(Log{}).Shard("logs").Standalone()
    if err != nil {
        log.Fatal(err)
    }
    
    // Run migrations
    norm.Norm()
}
```

**What happens:**
- Primary shard handles OLTP (transactional) data
- Standalone shards handle OLAP (analytical) data
- Each shard can be optimized for its workload
- Migrations run on all shards

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
