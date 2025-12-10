# Norm ORM Documentation

Welcome to the Norm ORM documentation! This guide will help you understand and use all features of Norm.

## 📚 Documentation Index

### Getting Started

1. **[Database Connections](01-database-connections.md)**
   - Connection types and modes
   - Global, Read/Write Split, and Sharding
   - Scenario-based examples
   - Best practices

2. **[Table Registration](02-table-registration.md)**
   - How to register tables
   - Global vs Shard mode
   - Role-based table assignment
   - Dynamic registration

3. **[Model Definition](03-model-definition.md)**
   - Struct tags and field types
   - Constraints and indexes
   - Foreign keys (hard and soft)
   - Advanced features

4. **[Migrations](04-migrations.md)**
   - Automatic schema migrations
   - Adding columns and indexes
   - Production deployment
   - Troubleshooting

5. **[CRUD Operations](05-crud-operations.md)**
   - INSERT, SELECT, UPDATE, DELETE
   - Struct-based and pair-based APIs
   - Bulk operations
   - Upsert and advanced features

---

## 🚀 Quick Start

### Installation

```bash
go get github.com/skssmd/norm
```

### Basic Example

```go
package main

import (
    "time"
    "github.com/skssmd/norm"
)

// Define your model
type User struct {
    ID        uint      `norm:"index;notnull;pk"`
    Email     string    `norm:"unique;notnull"`
    Name      string    `norm:"notnull"`
    CreatedAt time.Time `norm:"notnull;default:NOW()"`
}

func main() {
    // 1. Register database
    dsn := "postgresql://user:pass@localhost:5432/mydb"
    norm.Register(dsn).Primary()
    
    // 2. Register table with explicit name
    norm.RegisterTable(User{}, "users")
    
    // 3. Run migrations
    norm.Norm()
    
    // 4. Perform CRUD operations
    user := User{Name: "John", Email: "john@example.com"}
    norm.Table(user).Insert().Exec()
}
```

---

## 📖 Core Concepts

### Connection Modes

| Mode | Use Case | Complexity |
|------|----------|------------|
| **Global** | Single database | Low |
| **Read/Write Split** | Separate read/write | Medium |
| **Sharding** | Multi-tenant, scale | High |

### Model Tags

| Tag | Purpose | Example |
|-----|---------|---------|
| `pk` | Primary key | `norm:"pk"` |
| `notnull` | NOT NULL | `norm:"notnull"` |
| `unique` | UNIQUE | `norm:"unique"` |
| `index` | Create index | `norm:"index"` |
| `fkey` | Foreign key | `norm:"fkey:users.id"` |
| `skey` | Soft key | `norm:"skey:users.id"` |
| `max:N` | VARCHAR(N) | `norm:"max:255"` |
| `text` | TEXT type | `norm:"text"` |
| `default` | Default value | `norm:"default:NOW()"` |

### Type Mapping

| Go Type | PostgreSQL Type |
|---------|----------------|
| `uint` | `BIGINT` |
| `int` | `INTEGER` |
| `string` | `VARCHAR(255)` |
| `string` with `text` | `TEXT` |
| `time.Time` | `TIMESTAMP` |
| `*string` | `VARCHAR(255)` (nullable) |
| `*uint` | `BIGINT` (nullable) |

---

## 🎯 Common Scenarios

### Scenario 1: Single Database

```go
// Simple monolith application
norm.Register(dsn).Primary()
norm.Table(User{})
norm.Table(Order{})
norm.Norm()
```

### Scenario 2: Read Replicas

```go
// Scale reads with replicas
norm.Register(dsnPrimary).Primary()
norm.Register(dsnReplica1).Replica()
norm.Register(dsnReplica2).Replica()

norm.Table(User{})
norm.Table(Order{})
norm.Norm()
```

### Scenario 3: Multi-Tenant Sharding

```go
// Separate tenant data
norm.Register(dsn1).Shard("shard1").Primary()
norm.Register(dsn2).Shard("shard2").Primary()

norm.Table(User{}).Shard("shard1").Primary()
norm.Table(Order{}).Shard("shard1").Primary()

norm.Table(Analytics{}).Shard("shard2").Standalone()
norm.Norm()
```

---

## 🔑 Key Features

### ✅ Automatic Migrations
- Creates tables from structs
- Adds new columns automatically
- Creates indexes and foreign keys
- Never drops data

### ✅ Multiple Connection Modes
- Global (single DB)
- Read/Write split
- Sharding (multi-tenant)
- Hybrid configurations

### ✅ Flexible Relationships
- Hard foreign keys (DB-enforced)
- Soft foreign keys (app-level)
- Cascade delete/set null
- Cross-database relationships

### ✅ Type Safety
- Go types → PostgreSQL types
- Nullable fields with pointers
- Custom types supported
- JSON/JSONB support

### ✅ Production Ready
- Connection pooling
- Concurrent migrations
- Error handling
- Safety guarantees

---

## 📝 Examples

### Basic User Model

```go
type User struct {
    ID        uint       `norm:"index;notnull;pk"`
    Email     string     `norm:"unique;notnull"`
    Name      string     `norm:"notnull"`
    Bio       *string    `norm:"text"`           // nullable
    Age       *uint      `norm:""`               // nullable
    CreatedAt time.Time  `norm:"notnull;default:NOW()"`
    UpdatedAt *time.Time `norm:"default:NOW()"` // nullable
}
```

### Foreign Key Relationship

```go
type Order struct {
    ID        uint      `norm:"index;notnull;pk"`
    UserID    uint      `norm:"fkey:users.id;ondelete:cascade;notnull"`
    Total     float64   `norm:"notnull"`
    Status    string    `norm:"max:20;default:'pending'"`
    CreatedAt time.Time `norm:"notnull;default:NOW()"`
}
```

### Soft Key (Logical Relationship)

```go
type Analytics struct {
    ID        uint      `norm:"index;notnull;pk"`
    UserID    *uint     `norm:"skey:users.id;ondelete:setnull"` // soft key
    EventType string    `norm:"max:100"`
    EventData string    `norm:"type:JSONB"`
    CreatedAt time.Time `norm:"notnull;default:NOW()"`
}
```

---

## 🛠️ Development Tools

### Drop Tables (Development Only)

```go
// Drop all tables and recreate
if os.Getenv("ENV") == "development" {
    norm.DropTables()
}
norm.Norm()
```

### Environment Variables

```bash
# .env file
DATABASE_DSN=postgresql://user:pass@localhost:5432/mydb
DATABASE_DSN2=postgresql://user:pass@localhost:5432/shard2
ENV=development
```

```go
// Load from environment
dsn := os.Getenv("DATABASE_DSN")
norm.Register(dsn).Primary()
```

---

## 🔍 Troubleshooting

### Common Issues

1. **Foreign Key Constraint Fails**
   - Check existing data has valid foreign keys
   - Clean up orphaned records

2. **Column Type Mismatch**
   - Manual migration required for type changes
   - Use `ALTER TABLE` SQL

3. **Index Already Exists**
   - Safe to ignore if index exists
   - Migration is idempotent

4. **Shard Not Found**
   - Ensure tables are assigned to registered shards
   - Check shard names match

See [Migrations Documentation](04-migrations.md#troubleshooting) for detailed solutions.

---

## 📚 API Reference

### Database Registration

```go
// Global mode
norm.Register(dsn).Primary()
norm.Register(dsn).Replica()

// Read/Write split
norm.Register(dsn).Write()
norm.Register(dsn).Read()

// Sharding
norm.Register(dsn).Shard(name).Primary()
norm.Register(dsn).Shard(name).Standalone()
```

### Table Registration

```go
// Global mode (automatic)
norm.Table(Model{})

// Shard mode (explicit)
norm.Table(Model{}).Shard(name).Primary()
norm.Table(Model{}).Shard(name).Standalone()
norm.Table(Model{}).Shard(name).Read()
norm.Table(Model{}).Shard(name).Write()
```

### Migration Functions

```go
// Run migrations
norm.Norm()

// Drop all tables (development only)
norm.DropTables()
```

---

## 🎓 Learning Path

1. **Start Here**: [Database Connections](01-database-connections.md)
   - Understand connection modes
   - Choose the right architecture

2. **Next**: [Table Registration](02-table-registration.md)
   - Learn how to register tables
   - Understand global vs shard mode

3. **Then**: [Model Definition](03-model-definition.md)
   - Define your data models
   - Use tags effectively

4. **Then**: [Migrations](04-migrations.md)
   - Run your first migration
   - Deploy to production

5. **Finally**: [CRUD Operations](05-crud-operations.md)
   - Perform database operations
   - Learn all API styles
   - Master bulk operations

---

## 💡 Best Practices

### ✅ Do

- Use primary keys on all tables
- Use pointers for nullable fields
- Index foreign keys
- Add timestamps (created_at, updated_at)
- Test migrations locally first
- Backup before major changes
- Use environment variables for DSNs

### ❌ Don't

- Don't use TEXT for short strings
- Don't skip error handling
- Don't run DropTables() in production
- Don't change column types without migration
- Don't mix connection modes
- Don't forget to register tables

---

## 🤝 Contributing

Found an issue or have a suggestion? Please open an issue or submit a pull request!

---

## 📄 License

[Your License Here]

---

## 🔗 Links

- [GitHub Repository](https://github.com/skssmd/norm)
- [Issue Tracker](https://github.com/skssmd/norm/issues)
- [Discussions](https://github.com/skssmd/norm/discussions)

---

**Happy coding with Norm! 🚀**
