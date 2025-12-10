# Model Definition

This guide covers how to define database models (structs) with Norm tags for automatic schema generation.

## Table of Contents
- [Overview](#overview)
- [Basic Structure](#basic-structure)
- [Norm Tags](#norm-tags)
- [Field Types](#field-types)
- [Relationships](#relationships)
- [Advanced Features](#advanced-features)
- [Examples](#examples)
- [Best Practices](#best-practices)

---

## Overview

Models in Norm are Go structs with special `norm` tags that define:
- Column types and constraints
- Indexes and primary keys
- Foreign key relationships
- Default values
- Validation rules

---

## Basic Structure

```go
type ModelName struct {
    FieldName FieldType `norm:"tag1;tag2;tag3"`
}
```

### Example

```go
type User struct {
    ID        uint      `norm:"index;notnull;pk"`
    Email     string    `norm:"unique;notnull"`
    Name      string    `norm:"notnull"`
    CreatedAt time.Time `norm:"notnull;default:NOW()"`
}
```

---

## Norm Tags

### Constraint Tags

| Tag | Description | SQL Effect | Example |
|-----|-------------|------------|---------|
| `pk` | Primary key | `PRIMARY KEY` | `norm:"pk"` |
| `notnull` | Not null constraint | `NOT NULL` | `norm:"notnull"` |
| `unique` | Unique constraint | `UNIQUE` | `norm:"unique"` |
| `index` | Create index | `CREATE INDEX` | `norm:"index"` |
| `default:value` | Default value | `DEFAULT value` | `norm:"default:NOW()"` |

### Type Tags

| Tag | Description | SQL Type | Example |
|-----|-------------|----------|---------|
| `max:N` | VARCHAR length | `VARCHAR(N)` | `norm:"max:255"` |
| `text` | Unlimited text | `TEXT` | `norm:"text"` |
| `type:TYPE` | Custom SQL type | `TYPE` | `norm:"type:JSONB"` |

### Relationship Tags

| Tag | Description | SQL Effect | Example |
|-----|-------------|------------|---------|
| `fkey:table.column` | Foreign key (hard) | `FOREIGN KEY` | `norm:"fkey:users.id"` |
| `skey:table.column` | Soft key (logical) | Index only | `norm:"skey:users.id"` |
| `ondelete:action` | Delete action | `ON DELETE action` | `norm:"ondelete:cascade"` |
| `onupdate:action` | Update action | `ON UPDATE action` | `norm:"onupdate:cascade"` |

---

## Field Types

### Go Type → PostgreSQL Type Mapping

| Go Type | PostgreSQL Type | Notes |
|---------|----------------|-------|
| `int`, `int8`, `int16`, `int32` | `INTEGER` | Signed integers |
| `int64` | `BIGINT` | Large signed integers |
| `uint`, `uint8`, `uint16`, `uint32` | `BIGINT` | Unsigned integers |
| `uint64` | `BIGINT` | Large unsigned integers |
| `float32` | `REAL` | Single precision |
| `float64` | `DOUBLE PRECISION` | Double precision |
| `bool` | `BOOLEAN` | True/false |
| `string` | `VARCHAR(255)` | Default string |
| `string` with `max:N` | `VARCHAR(N)` | Custom length |
| `string` with `text` | `TEXT` | Unlimited length |
| `time.Time` | `TIMESTAMP` | Date and time |
| `[]byte` | `BYTEA` | Binary data |
| Structs | `JSONB` | JSON data |
| Slices | `JSONB` | JSON arrays |

### Pointer Types (Nullable)

| Go Type | PostgreSQL Type | Nullable |
|---------|----------------|----------|
| `*string` | `VARCHAR(255)` | ✅ Yes |
| `*uint` | `BIGINT` | ✅ Yes |
| `*int` | `INTEGER` | ✅ Yes |
| `*time.Time` | `TIMESTAMP` | ✅ Yes |
| `*bool` | `BOOLEAN` | ✅ Yes |

---

## Relationships

### Hard Foreign Keys (`fkey`)

Database-enforced foreign key constraints.

```go
type Order struct {
    ID     uint `norm:"index;notnull;pk"`
    UserID uint `norm:"fkey:users.id;ondelete:cascade;notnull"`
}
```

**Generated SQL:**
```sql
CREATE TABLE orders (
    id BIGINT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
CREATE INDEX idx_orders_user_id ON orders(user_id);
```

**When to use:**
- ✅ Critical relationships (orders must have valid users)
- ✅ Data integrity is essential
- ✅ Tables in same database
- ✅ Want database-enforced constraints

### Soft Foreign Keys (`skey`)

Application-level logical relationships without DB constraints.

```go
type Analytics struct {
    ID     uint  `norm:"index;notnull;pk"`
    UserID *uint `norm:"skey:users.id;ondelete:setnull"`
}
```

**Generated SQL:**
```sql
CREATE TABLE analytics (
    id BIGINT PRIMARY KEY,
    user_id BIGINT  -- nullable, no FK constraint
);
CREATE INDEX idx_analytics_user_id ON analytics(user_id);
```

**When to use:**
- ✅ Optional relationships
- ✅ Analytics/logging data
- ✅ Cross-database relationships
- ✅ Application-level cascade control
- ✅ Historical data preservation

### Cascade Actions

| Action | Hard Key (`fkey`) | Soft Key (`skey`) |
|--------|------------------|------------------|
| `cascade` | DB deletes related rows | App must delete |
| `setnull` | DB sets to NULL | App must set NULL |
| `restrict` | DB prevents delete | N/A |
| `noaction` | DB does nothing | N/A |
| `setdefault` | DB sets default value | N/A |

---

## Advanced Features

### 1. Composite Primary Keys

```go
type UserRole struct {
    UserID uint `norm:"pk;notnull"`
    RoleID uint `norm:"pk;notnull"`
}
```

**Generated SQL:**
```sql
CREATE TABLE user_roles (
    user_id BIGINT NOT NULL,
    role_id BIGINT NOT NULL,
    PRIMARY KEY (user_id, role_id)
);
```

### 2. JSON Fields

```go
type Product struct {
    ID         uint   `norm:"index;notnull;pk"`
    Name       string `norm:"notnull"`
    Attributes string `norm:"type:JSONB"`  // JSON data
}
```

**Usage:**
```go
// Store JSON
product := Product{
    Name: "Laptop",
    Attributes: `{"brand": "Dell", "ram": "16GB"}`,
}
```

### 3. Custom SQL Types

```go
type Location struct {
    ID    uint   `norm:"index;notnull;pk"`
    Point string `norm:"type:GEOMETRY(POINT, 4326)"` // PostGIS
}
```

### 4. Array Fields

```go
type Article struct {
    ID   uint     `norm:"index;notnull;pk"`
    Tags []string `norm:"type:TEXT[]"` // PostgreSQL array
}
```

### 5. Enum-like Fields

```go
type Order struct {
    ID     uint   `norm:"index;notnull;pk"`
    Status string `norm:"max:20;default:'pending'"`
}

// Use constants for type safety
const (
    OrderStatusPending   = "pending"
    OrderStatusProcessing = "processing"
    OrderStatusCompleted  = "completed"
    OrderStatusCancelled  = "cancelled"
)
```

---

## Examples

### Example 1: Basic User Model

```go
package main

import "time"

// User represents a user account
type User struct {
    // Primary key - auto-increment ID
    ID uint `norm:"index;notnull;pk"`
    
    // Unique email for login
    Email string `norm:"unique;notnull"`
    
    // User's full name
    Name string `norm:"notnull"`
    
    // Username for display (optional)
    Username *string `norm:"unique"` // nullable
    
    // Hashed password
    PasswordHash string `norm:"notnull"`
    
    // Profile bio (long text)
    Bio *string `norm:"text"` // nullable, unlimited length
    
    // Age (optional)
    Age *uint `norm:""` // nullable BIGINT
    
    // Account status
    IsActive bool `norm:"notnull;default:true"`
    
    // Timestamps
    CreatedAt time.Time  `norm:"notnull;default:NOW()"`
    UpdatedAt *time.Time `norm:"default:NOW()"` // nullable
}
```

**Generated SQL:**
```sql
CREATE TABLE users (
    id BIGINT PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    username VARCHAR(255) UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    bio TEXT,
    age BIGINT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
CREATE INDEX idx_users_id ON users(id);
```

---

### Example 2: E-commerce Models

```go
package main

import "time"

// Product represents an item for sale
type Product struct {
    ID          uint    `norm:"index;notnull;pk"`
    SKU         string  `norm:"unique;notnull;max:50"`
    Name        string  `norm:"notnull"`
    Description string  `norm:"text"`
    Price       float64 `norm:"notnull"`
    Stock       int     `norm:"notnull;default:0"`
    IsActive    bool    `norm:"notnull;default:true"`
    CreatedAt   time.Time `norm:"notnull;default:NOW()"`
}

// Order represents a customer order
type Order struct {
    ID         uint      `norm:"index;notnull;pk"`
    OrderNumber string   `norm:"unique;notnull;max:50"`
    
    // Hard foreign key - order must have valid user
    UserID     uint      `norm:"fkey:users.id;ondelete:restrict;notnull"`
    
    Total      float64   `norm:"notnull"`
    Status     string    `norm:"max:20;default:'pending'"`
    Notes      *string   `norm:"text"` // nullable
    CreatedAt  time.Time `norm:"notnull;default:NOW()"`
    UpdatedAt  *time.Time `norm:"default:NOW()"`
}

// OrderItem represents a line item in an order
type OrderItem struct {
    ID        uint    `norm:"index;notnull;pk"`
    
    // Hard foreign keys with cascade
    OrderID   uint    `norm:"fkey:orders.id;ondelete:cascade;notnull"`
    ProductID uint    `norm:"fkey:products.id;ondelete:restrict;notnull"`
    
    Quantity  int     `norm:"notnull"`
    Price     float64 `norm:"notnull"` // Price at time of order
}
```

**Relationships:**
- Order → User: `RESTRICT` (can't delete user with orders)
- OrderItem → Order: `CASCADE` (delete items when order deleted)
- OrderItem → Product: `RESTRICT` (can't delete product with order items)

---

### Example 3: Blog Platform Models

```go
package main

import "time"

// User - blog author
type User struct {
    ID        uint      `norm:"index;notnull;pk"`
    Username  string    `norm:"unique;notnull;max:50"`
    Email     string    `norm:"unique;notnull"`
    Bio       *string   `norm:"text"`
    CreatedAt time.Time `norm:"notnull;default:NOW()"`
}

// Post - blog post
type Post struct {
    ID        uint      `norm:"index;notnull;pk"`
    
    // Hard foreign key - post must have author
    AuthorID  uint      `norm:"fkey:users.id;ondelete:cascade;notnull"`
    
    Title     string    `norm:"notnull"`
    Slug      string    `norm:"unique;notnull;max:200"`
    Content   string    `norm:"text;notnull"`
    Excerpt   *string   `norm:"max:500"`
    
    // Status enum
    Status    string    `norm:"max:20;default:'draft'"`
    
    // View count
    Views     uint      `norm:"default:0"`
    
    // Timestamps
    PublishedAt *time.Time `norm:""`
    CreatedAt   time.Time  `norm:"notnull;default:NOW()"`
    UpdatedAt   *time.Time `norm:"default:NOW()"`
}

// Comment - post comment
type Comment struct {
    ID        uint      `norm:"index;notnull;pk"`
    
    // Hard foreign keys
    PostID    uint      `norm:"fkey:posts.id;ondelete:cascade;notnull"`
    AuthorID  uint      `norm:"fkey:users.id;ondelete:cascade;notnull"`
    
    // Optional parent comment for threading
    ParentID  *uint     `norm:"fkey:comments.id;ondelete:cascade"`
    
    Content   string    `norm:"text;notnull"`
    IsApproved bool     `norm:"notnull;default:false"`
    CreatedAt time.Time `norm:"notnull;default:NOW()"`
}

// Tag - post categorization
type Tag struct {
    ID   uint   `norm:"index;notnull;pk"`
    Name string `norm:"unique;notnull;max:50"`
    Slug string `norm:"unique;notnull;max:50"`
}

// PostTag - many-to-many relationship
type PostTag struct {
    PostID uint `norm:"pk;fkey:posts.id;ondelete:cascade;notnull"`
    TagID  uint `norm:"pk;fkey:tags.id;ondelete:cascade;notnull"`
}
```

**Features:**
- Threaded comments (self-referencing foreign key)
- Many-to-many tags (junction table)
- Soft delete support (status field)
- SEO-friendly slugs

---

### Example 4: Analytics & Logging Models

```go
package main

import "time"

// User - main user table
type User struct {
    ID        uint      `norm:"index;notnull;pk"`
    Email     string    `norm:"unique;notnull"`
    Name      string    `norm:"notnull"`
    CreatedAt time.Time `norm:"notnull;default:NOW()"`
}

// Analytics - user activity tracking
type Analytics struct {
    ID        uint      `norm:"index;notnull;pk"`
    
    // Soft foreign key - keep analytics even if user deleted
    UserID    *uint     `norm:"skey:users.id;ondelete:setnull"`
    
    EventType string    `norm:"index;notnull;max:100"`
    EventData string    `norm:"type:JSONB"` // Flexible JSON data
    
    // Session tracking
    SessionID *string   `norm:"max:100"`
    IPAddress *string   `norm:"max:45"` // IPv6 compatible
    UserAgent *string   `norm:"text"`
    
    CreatedAt time.Time `norm:"index;notnull;default:NOW()"`
}

// Log - application logs
type Log struct {
    ID        uint      `norm:"index;notnull;pk"`
    
    // Soft foreign key - logs survive user deletion
    UserID    *uint     `norm:"skey:users.id;ondelete:setnull"`
    
    Level     string    `norm:"index;notnull;max:20"` // INFO, WARN, ERROR
    Message   string    `norm:"text;notnull"`
    Context   *string   `norm:"type:JSONB"` // Additional context
    
    // Stack trace for errors
    StackTrace *string  `norm:"text"`
    
    CreatedAt time.Time `norm:"index;notnull;default:NOW()"`
}

// PageView - website analytics
type PageView struct {
    ID        uint      `norm:"index;notnull;pk"`
    
    // Soft key - anonymous tracking OK
    UserID    *uint     `norm:"skey:users.id;ondelete:setnull"`
    
    URL       string    `norm:"index;notnull;max:500"`
    Referrer  *string   `norm:"max:500"`
    Duration  *int      `norm:""` // Time spent on page (seconds)
    
    CreatedAt time.Time `norm:"index;notnull;default:NOW()"`
}
```

**Why soft keys here:**
- Analytics data is historical
- Should survive user deletion
- No referential integrity needed
- Cross-database analytics possible

---

### Example 5: Multi-Tenant SaaS Models

```go
package main

import "time"

// Tenant - organization/company
type Tenant struct {
    ID        uint      `norm:"index;notnull;pk"`
    Name      string    `norm:"notnull"`
    Slug      string    `norm:"unique;notnull;max:100"`
    Plan      string    `norm:"max:50;default:'free'"`
    IsActive  bool      `norm:"notnull;default:true"`
    CreatedAt time.Time `norm:"notnull;default:NOW()"`
}

// User - tenant user
type User struct {
    ID        uint      `norm:"index;notnull;pk"`
    
    // Hard foreign key - user belongs to tenant
    TenantID  uint      `norm:"index;fkey:tenants.id;ondelete:cascade;notnull"`
    
    Email     string    `norm:"notnull"` // Unique per tenant
    Name      string    `norm:"notnull"`
    Role      string    `norm:"max:50;default:'member'"`
    CreatedAt time.Time `norm:"notnull;default:NOW()"`
}

// Project - tenant project
type Project struct {
    ID        uint      `norm:"index;notnull;pk"`
    
    // Hard foreign key - project belongs to tenant
    TenantID  uint      `norm:"index;fkey:tenants.id;ondelete:cascade;notnull"`
    
    Name      string    `norm:"notnull"`
    Status    string    `norm:"max:20;default:'active'"`
    CreatedAt time.Time `norm:"notnull;default:NOW()"`
}

// Task - project task
type Task struct {
    ID        uint      `norm:"index;notnull;pk"`
    
    // Hard foreign keys
    TenantID  uint      `norm:"index;fkey:tenants.id;ondelete:cascade;notnull"`
    ProjectID uint      `norm:"fkey:projects.id;ondelete:cascade;notnull"`
    AssigneeID *uint    `norm:"fkey:users.id;ondelete:setnull"` // Optional
    
    Title     string    `norm:"notnull"`
    Description *string `norm:"text"`
    Status    string    `norm:"max:20;default:'todo'"`
    Priority  string    `norm:"max:20;default:'medium'"`
    DueDate   *time.Time `norm:""`
    CreatedAt time.Time `norm:"notnull;default:NOW()"`
}
```

**Multi-tenant features:**
- All tables have `TenantID` for isolation
- Cascade delete when tenant deleted
- Composite indexes on `(tenant_id, ...)` for performance

---

## Best Practices

### 1. Always Use Primary Keys

```go
// ✅ Good
type User struct {
    ID uint `norm:"index;notnull;pk"`
    // ...
}

// ❌ Bad - no primary key
type User struct {
    Email string `norm:"unique;notnull"`
    // ...
}
```

### 2. Use Appropriate String Lengths

```go
// ✅ Good - specific lengths
type User struct {
    Email    string `norm:"unique;notnull"`        // VARCHAR(255)
    Username string `norm:"unique;notnull;max:50"` // VARCHAR(50)
    Bio      string `norm:"text"`                  // TEXT
}

// ❌ Bad - TEXT for everything (slower indexes)
type User struct {
    Email    string `norm:"text"`
    Username string `norm:"text"`
    Bio      string `norm:"text"`
}
```

### 3. Use Pointers for Nullable Fields

```go
// ✅ Good - explicit nullability
type User struct {
    Name      string     `norm:"notnull"`
    Bio       *string    `norm:"text"`      // nullable
    Age       *uint      `norm:""`          // nullable
    UpdatedAt *time.Time `norm:"default:NOW()"` // nullable
}

// ❌ Bad - unclear if nullable
type User struct {
    Name string `norm:""`
    Bio  string `norm:""`
}
```

### 4. Index Foreign Keys

```go
// ✅ Good - index is automatic with fkey/skey
type Order struct {
    UserID uint `norm:"fkey:users.id;ondelete:cascade"`
}

// ✅ Also good - explicit index
type Order struct {
    UserID uint `norm:"index;notnull"`
}
```

### 5. Use Timestamps

```go
// ✅ Good - track creation and updates
type Model struct {
    ID        uint       `norm:"index;notnull;pk"`
    CreatedAt time.Time  `norm:"notnull;default:NOW()"`
    UpdatedAt *time.Time `norm:"default:NOW()"`
}
```

### 6. Document Complex Fields

```go
type Product struct {
    ID uint `norm:"index;notnull;pk"`
    
    // JSON field storing product attributes
    // Example: {"color": "red", "size": "large"}
    Attributes string `norm:"type:JSONB"`
    
    // Status: draft, active, archived
    Status string `norm:"max:20;default:'draft'"`
}
```

### 7. Use Constants for Enums

```go
// Define constants
const (
    OrderStatusPending   = "pending"
    OrderStatusProcessing = "processing"
    OrderStatusCompleted  = "completed"
)

type Order struct {
    ID     uint   `norm:"index;notnull;pk"`
    Status string `norm:"max:20;default:'pending'"`
}

// Use in code
order := Order{
    Status: OrderStatusPending,
}
```

---

## Summary

| Feature | Tag | Example |
|---------|-----|---------|
| **Primary Key** | `pk` | `norm:"pk"` |
| **Not Null** | `notnull` | `norm:"notnull"` |
| **Unique** | `unique` | `norm:"unique"` |
| **Index** | `index` | `norm:"index"` |
| **Default** | `default:value` | `norm:"default:NOW()"` |
| **VARCHAR** | `max:N` | `norm:"max:255"` |
| **TEXT** | `text` | `norm:"text"` |
| **Hard FK** | `fkey:table.col` | `norm:"fkey:users.id"` |
| **Soft FK** | `skey:table.col` | `norm:"skey:users.id"` |
| **Cascade** | `ondelete:cascade` | `norm:"ondelete:cascade"` |
| **Custom Type** | `type:TYPE` | `norm:"type:JSONB"` |

Define your models thoughtfully - they are the foundation of your database schema!
