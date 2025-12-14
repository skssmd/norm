```
   _  _   ___   ___  __  __ 
  | \| | / _ \ | _ \|  \/  |  
  | .` || (_) ||   /| |\/| |  
  |_|\_| \___/ |_|_\|_|  |_|  
                              
        .-"""-.
       /   _   \   [ norm: The Scalable Go ORM and Database Orchestrator For Postgres]
      |  (o)_(o)|  
      |   (_)   |   + Automatic Routing
     / \   U   / \  + R/W Separation
    |   \ `-' /   | + Table Sharding    
     \___\___/___/  + Multi-Shard Relations with Integrated Joins and Cascade
   
```

# Norm ORM

[![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.24-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

**Norm** is a powerful, flexible ORM for Go that supports multiple database architectures including monolithic, read/write split, and sharded configurations. Built on PostgreSQL with pgx/v5, Norm provides automatic query routing, intelligent join handling, and seamless struct scanning.

## ✨ Features

- 🚀 **Fluent Query Builder** - Intuitive, chainable API for all CRUD operations
- 🔄 **Multiple Database Modes** - Monolithic, Read/Write Split, Sharding
- 🔗 **Smart JOIN Support** - Native, App-Side (skey), and Distributed joins
- 📦 **Struct Scanning** - Automatic mapping of query results to Go structs
- 🎯 **Auto Migrations** - Automatic table creation from struct definitions
- ⚡ **Bulk Operations** - Efficient bulk inserts with struct support
- 🔑 **Foreign & Soft Keys** - Full support for fkey and skey relationships
- 🌐 **Sharding Support** - Automatic routing across multiple database shards
- 📊 **Type Safety** - Compile-time type checking with struct-based operations

## 📦 Installation

```bash
go get github.com/skssmd/norm
```

## 🚀 Quick Start

### 1. Define Your Models

```go
package main

import (
    "time"
    "github.com/skssmd/norm"
)

type User struct {
    ID        uint       `norm:"index;notnull;pk;auto"`
    Email     string     `norm:"name:useremail;unique;notnull"`
    Name      string     `norm:"name:fullname;notnull"`
    Username  string     `norm:"name:uname;notnull;unique"`
    Age       *uint      `norm:""`
    CreatedAt time.Time  `norm:"notnull;default:NOW()"`
}

type Order struct {
    ID        uint      `norm:"index;notnull;pk;auto"`
    UserID    uint      `norm:"index;notnull;fkey:users.id;ondelete:cascade"`
    Total     float64   `norm:"notnull"`
    Status    string    `norm:"max:20;default:'pending'"`
    CreatedAt time.Time `norm:"notnull;default:NOW()"`
}
```

### 2. Setup Database Connection

```go
func main() {
    // Register primary database
    err := norm.Register("postgres://user:pass@localhost:5432/mydb").Primary()
    if err != nil {
        log.Fatal(err)
    }

    // Register tables
    norm.RegisterTable(User{}, "users")
    norm.RegisterTable(Order{}, "orders")

    // Run auto migrations
    norm.Norm()
}
```

### 3. Perform CRUD Operations

```go
ctx := context.Background()

// INSERT
user := User{
    Name:     "Alice Williams",
    Email:    "alice@example.com",
    Username: "alicew",
}
rowsAffected, err := norm.Table(user).Insert().Exec(ctx)

// SELECT with scanning
var users []User
err = norm.Table("users").
    Select().
    Where("created_at > $1", time.Now().AddDate(0, -1, 0)).
    OrderBy("created_at DESC").
    All(ctx, &users)

// UPDATE
rowsAffected, err = norm.Table("users").
    Update("name", "Alice Updated").
    Where("username = $1", "alicew").
    Exec(ctx)

// DELETE
rowsAffected, err = norm.Table("users").
    Delete().
    Where("id = $1", 123).
    Exec(ctx)
```

### 4. JOIN Queries with Scanning

```go
type UserOrder struct {
    UserName   string  `norm:"name:fullname"`
    OrderTotal float64 `norm:"name:total"`
    Status     string  `norm:"name:status"`
}

var userOrders []UserOrder

err := norm.Table("users", "id", "orders", "user_id").
    Select("users.fullname", "orders.total", "orders.status").
    Where("users.username = $1", "alice").
    OrderBy("orders.created_at DESC").
    All(ctx, &userOrders)

for _, order := range userOrders {
    fmt.Printf("%s: $%.2f (%s)\n", order.UserName, order.OrderTotal, order.Status)
}
```

## 📚 Documentation

Comprehensive documentation is available in the [`docs/`](docs/) directory:

### Setup & Configuration
- [01 - Database Connections](docs/01-database-connections.md) - Connection modes and shard setup
- [02 - Model Definition](docs/02-model-definition.md) - Struct tags and field mapping
- [03 - Table Registration](docs/03-table-registration.md) - Table-to-shard mapping
- [04 - Migrations](docs/04-migrations.md) - Auto migrations and schema management

### Operations
- [05 - INSERT Operations](docs/05-insert.md) - Single and bulk inserts, upserts
- [06 - SELECT Operations](docs/06-select.md) - Queries and struct scanning
- [07 - UPDATE Operations](docs/07-update.md) - Pair-based and struct-based updates
- [08 - DELETE Operations](docs/08-delete.md) - Deletes and soft delete patterns
- [09 - JOIN Operations](docs/09-joins.md) - Native, app-side, and distributed joins
- [10 - Raw SQL](docs/10-raw-sql.md) - Raw SQL queries with routing

## 🎯 Key Concepts

### Database Modes

Norm supports multiple database architectures:

1. **Global Monolith** - Single primary database with optional replicas
2. **Read/Write Split** - Separate pools for read and write operations
3. **Sharding** - Distribute tables across multiple database shards

### JOIN Types

Norm automatically selects the optimal join strategy:

- **Native JOIN** - Standard SQL JOIN when tables are co-located
- **App-Side JOIN** - Application-level join for soft-key (skey) relationships
- **Distributed JOIN** - Cross-shard joins handled automatically

### Struct Scanning

Map query results directly to Go structs with automatic field mapping:

```go
var user User
err := norm.Table("users").
    Select().
    Where("id = $1", 123).
    First(ctx, &user)
```

## 🔧 Advanced Features

### Bulk Insert

```go
users := []User{
    {Name: "Alice", Email: "alice@example.com", Username: "alice"},
    {Name: "Bob", Email: "bob@example.com", Username: "bob"},
}

rowsAffected, err := norm.Table("users").
    BulkInsert(users).
    Exec(ctx)
```

### Upsert (ON CONFLICT)

```go
user := User{Name: "John", Email: "john@example.com"}

rowsAffected, err := norm.Table(user).
    Insert().
    OnConflict("email", "update", "name", "updated_at").
    Exec(ctx)
```

### Sharding Setup

```go
// Register shards
norm.Register(dsn1).Shard("shard1").Primary()
norm.Register(dsn2).Shard("shard2").Primary()

// Assign tables to shards
norm.RegisterTable(User{}, "users").Primary("shard1")
norm.RegisterTable(Order{}, "orders").Primary("shard1")
norm.RegisterTable(Analytics{}, "analytics").Standalone("shard2")
```

## 🧪 Examples

Check out the [`examples/`](examples/) directory for complete working examples:

- Basic CRUD operations
- JOIN queries with scanning
- Sharding configurations
- Bulk operations
- Migration examples

Or run the comprehensive test scenarios:

```bash
cd examples
go run test_all_scenarios.go query.go db.go
```

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Built with [pgx/v5](https://github.com/jackc/pgx) - High-performance PostgreSQL driver
- Inspired by modern ORM patterns and best practices

## 📞 Support

- 📖 [Documentation](docs/)
- 🐛 [Issue Tracker](https://github.com/skssmd/norm/issues)
- 💬 [Discussions](https://github.com/skssmd/norm/discussions)

---

Made with ❤️ by the Norm team
