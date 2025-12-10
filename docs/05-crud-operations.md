# CRUD Operations

Complete guide to Create, Read, Update, and Delete operations in Norm ORM.

## Table of Contents
- [Overview](#overview)
- [INSERT Operations](#insert-operations)
- [SELECT Operations](#select-operations)
- [UPDATE Operations](#update-operations)
- [DELETE Operations](#delete-operations)
- [Bulk Operations](#bulk-operations)
- [Advanced Features](#advanced-features)

---

## Overview

Norm provides a fluent, type-safe API for database operations. All operations support:
- ✅ **Optional Context** - Pass `ctx` for timeouts/cancellation, or omit for simplicity
- ✅ **String-based** - Explicit column names
- ✅ **Struct-based** - Type-safe with automatic field extraction
- ✅ **Pair-based** - Clean key-value syntax

---

## INSERT Operations

### 1. Struct-Based Insert (Non-Zero Values Only)

Inserts only non-zero fields, ignoring empty strings, nil, 0, false:

```go
age := uint(29)
user := User{
    Name:  "John Doe",
    Email: "john@example.com",
    Age:   &age,
    // Username is empty, will be ignored
}

rowsAffected, err := norm.Table(user).Insert().Exec()
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Inserted %d user(s)\n", rowsAffected)
```

**Generated SQL:**
```sql
INSERT INTO users (name, email, age) VALUES ($1, $2, $3)
-- Only non-zero fields are included
```

### 2. Insert with Context

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

rowsAffected, err := norm.Table(user).Insert().Exec(ctx)
```

### 3. Insert with Specific Table Name

```go
user := User{Name: "Jane", Email: "jane@example.com"}
rowsAffected, err := norm.Table("users").Insert(user).Exec()
```

---

## SELECT Operations

### 1. Simple SELECT

```go
count, err := norm.Table("users").
    Select("id", "name", "email").
    Count()
```

### 2. SELECT with WHERE

```go
count, err := norm.Table("users").
    Select("name", "email", "created_at").
    Where("created_at > $1", time.Now().AddDate(0, -1, 0)).
    Count()
```

### 3. SELECT All Fields

```go
count, err := norm.Table("users").
    Select().  // Selects all fields (*)
    Count()
```

### 4. SELECT with Pagination

```go
count, err := norm.Table("users").
    Select("id", "name", "email").
    OrderBy("created_at DESC").
    Pagination(10, 0).  // limit, offset
    Count()
```

**Generated SQL:**
```sql
SELECT id, name, email FROM users 
ORDER BY created_at DESC 
LIMIT 10 OFFSET 0
```

---

## UPDATE Operations

Norm provides **three ways** to update data:

### 1. Pair-Based Update (Recommended)

Clean key-value syntax for explicit updates:

```go
// Update single field
rowsAffected, err := norm.Table("users").
    Update("name", "John Updated").
    Where("username = $1", "johndoe").
    Exec()

// Update multiple fields
rowsAffected, err := norm.Table("users").
    Update("name", "Jane Updated", "email", "jane.new@example.com").
    Where("id = $1", 123).
    Exec()
```

**Generated SQL:**
```sql
UPDATE users SET name = $1, email = $2 WHERE id = $3
```

**Key Feature:** Can explicitly set values to zero (`""`, `0`, `false`)

### 2. Struct-Based Update (Partial Updates)

Updates only non-zero fields, keeping old values for zero fields:

```go
user := User{
    Name:  "John Updated",
    Email: "new@example.com",
    // Age is 0, will be ignored (keeps old value)
    // Username is "", will be ignored (keeps old value)
}

rowsAffected, err := norm.Table(user).
    Update().
    Where("id = $1", 123).
    Exec()
```

**Generated SQL:**
```sql
UPDATE users SET name = $1, email = $2 WHERE id = $3
-- Only non-zero fields are updated
```

**Use Case:** Partial updates where you want to keep existing values for unset fields

### 3. Update with Context

```go
ctx := context.Background()
rowsAffected, err := norm.Table("users").
    Update("status", "active").
    Where("id = $1", 123).
    Exec(ctx)
```

### Comparison: When to Use Each

| Method | Zero Values | Use Case |
|--------|-------------|----------|
| **Pair-based** | Always included | Explicit updates, can set to zero |
| **Struct-based** | Ignored | Partial updates, keep old values |

**Examples:**

```go
// ✅ Pair-based: Set age to 0 explicitly
norm.Table("users").Update("age", 0).Where("id = $1", 123).Exec()
// SQL: UPDATE users SET age = $1 WHERE id = $2
// Values: [0, 123]

// ✅ Struct-based: Update only name, keep everything else
norm.Table(User{Name: "John"}).Update().Where("id = $1", 123).Exec()
// SQL: UPDATE users SET name = $1 WHERE id = $2
// Values: ["John", 123]
```

---

## DELETE Operations

### 1. Delete with Condition

```go
rowsAffected, err := norm.Table("users").
    Delete().
    Where("email = $1", "test@example.com").
    Exec()
```

**Generated SQL:**
```sql
DELETE FROM users WHERE email = $1
```

### 2. Delete with Multiple Conditions

```go
rowsAffected, err := norm.Table("users").
    Delete().
    Where("status = $1 AND created_at < $2", "inactive", time.Now().AddDate(-1, 0, 0)).
    Exec()
```

**⚠️ Warning:** Always use WHERE clause to avoid deleting all rows!

---

## Bulk Operations

### 1. Struct-Based Bulk Insert (Recommended)

Create a slice of structs and insert all at once:

```go
// Create slice of users
bulkUsers := []User{
    {Name: "Alice", Email: "alice@example.com", Username: "alicew"},
    {Name: "Bob", Email: "bob@example.com", Username: "bobb"},
    {Name: "Charlie", Email: "charlie@example.com", Username: "charlied"},
}

// Insert all at once
rowsAffected, err := norm.Table("users").
    BulkInsert(bulkUsers).
    Exec()
```

**Generated SQL:**
```sql
INSERT INTO users (email, name, username) 
VALUES ($1, $2, $3), ($4, $5, $6), ($7, $8, $9)
```

### 2. Generate Bulk Data in Loop

```go
generatedUsers := make([]User, 0)
for i := 1; i <= 100; i++ {
    generatedUsers = append(generatedUsers, User{
        Name:     fmt.Sprintf("User %d", i),
        Email:    fmt.Sprintf("user%d@example.com", i),
        Username: fmt.Sprintf("user%d", i),
    })
}

rowsAffected, err := norm.Table("users").
    BulkInsert(generatedUsers).
    Exec()
```

### 3. Manual Bulk Insert (Legacy)

```go
rowsAffected, err := norm.Table("users").
    BulkInsert(
        []string{"name", "email", "username"},
        [][]interface{}{
            {"Alice", "alice@example.com", "alicew"},
            {"Bob", "bob@example.com", "bobb"},
        },
    ).
    Exec()
```

**Recommendation:** Use struct-based bulk insert for type safety and cleaner code.

---

## Advanced Features

### 1. Upsert (ON CONFLICT)

Insert or update on conflict:

```go
// Insert or keep old value if email exists
user := User{
    Name:  "John",
    Email: "john@example.com",
    Age:   &age,
}

rowsAffected, err := norm.Table(user).
    Insert().
    OnConflict("email", "nothing").  // Keep old record
    Exec()
```

**Update specific columns on conflict:**

```go
rowsAffected, err := norm.Table(user).
    Insert().
    OnConflict("email", "update", "name", "updated_at").  // Update these columns
    Exec()
```

**Generated SQL:**
```sql
INSERT INTO users (name, email, age) VALUES ($1, $2, $3)
ON CONFLICT (email) DO UPDATE SET name = EXCLUDED.name, updated_at = EXCLUDED.updated_at
```

### 2. Transactions (Coming Soon)

```go
// Future API
tx, err := norm.Begin()
defer tx.Rollback()

tx.Table("users").Insert(user).Exec()
tx.Table("orders").Insert(order).Exec()

tx.Commit()
```

### 3. Query Chaining

```go
// Build complex queries
query := norm.Table("users").
    Select("id", "name", "email").
    Where("status = $1", "active").
    OrderBy("created_at DESC").
    Pagination(20, 0)

count, err := query.Count()
```

---

## Complete Examples

### Example 1: User Registration

```go
func RegisterUser(name, email, username string) error {
    user := User{
        Name:     name,
        Email:    email,
        Username: username,
    }
    
    _, err := norm.Table(user).
        Insert().
        OnConflict("email", "nothing").  // Ignore if exists
        Exec()
    
    return err
}
```

### Example 2: Update User Profile

```go
func UpdateProfile(userID uint, updates User) error {
    // Only updates non-zero fields
    _, err := norm.Table(updates).
        Update().
        Where("id = $1", userID).
        Exec()
    
    return err
}
```

### Example 3: Bulk Import Users

```go
func ImportUsers(csvData [][]string) error {
    users := make([]User, 0, len(csvData))
    
    for _, row := range csvData {
        users = append(users, User{
            Name:     row[0],
            Email:    row[1],
            Username: row[2],
        })
    }
    
    _, err := norm.Table("users").
        BulkInsert(users).
        Exec()
    
    return err
}
```

### Example 4: Soft Delete

```go
func SoftDeleteUser(userID uint) error {
    _, err := norm.Table("users").
        Update("deleted_at", time.Now(), "status", "deleted").
        Where("id = $1", userID).
        Exec()
    
    return err
}
```

### Example 5: Search with Filters

```go
func SearchUsers(query string, limit, offset int) (int64, error) {
    return norm.Table("users").
        Select("id", "name", "email").
        Where("name ILIKE $1 OR email ILIKE $1", "%"+query+"%").
        OrderBy("created_at DESC").
        Pagination(limit, offset).
        Count()
}
```

---

## Best Practices

### 1. Always Use WHERE for Updates/Deletes

```go
// ❌ Bad - Updates all rows
norm.Table("users").Update("status", "inactive").Exec()

// ✅ Good - Updates specific rows
norm.Table("users").Update("status", "inactive").Where("id = $1", 123).Exec()
```

### 2. Use Struct-Based for Type Safety

```go
// ✅ Good - Compiler checks types
user := User{Name: "John", Email: "john@example.com"}
norm.Table(user).Insert().Exec()

// ❌ Less safe - No compile-time checks
norm.Table("users").Insert(map[string]interface{}{
    "name": "John",
    "email": "john@example.com",
}).Exec()
```

### 3. Use Bulk Insert for Multiple Rows

```go
// ❌ Bad - Multiple queries
for _, user := range users {
    norm.Table(user).Insert().Exec()
}

// ✅ Good - Single query
norm.Table("users").BulkInsert(users).Exec()
```

### 4. Handle Errors Properly

```go
rowsAffected, err := norm.Table("users").
    Update("status", "active").
    Where("id = $1", 123).
    Exec()

if err != nil {
    return fmt.Errorf("failed to update user: %w", err)
}

if rowsAffected == 0 {
    return errors.New("user not found")
}
```

### 5. Use Context for Timeouts

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

_, err := norm.Table("users").
    Select().
    Where("status = $1", "active").
    Count(ctx)
```

---

## Summary

### INSERT
- **Struct-based**: `Table(user).Insert().Exec()` - Non-zero values only
- **With table name**: `Table("users").Insert(user).Exec()`
- **Bulk**: `Table("users").BulkInsert(users).Exec()`
- **Upsert**: `Insert().OnConflict("email", "update").Exec()`

### SELECT
- **Simple**: `Table("users").Select("id", "name").Count()`
- **With WHERE**: `Select().Where("status = $1", "active").Count()`
- **Pagination**: `Select().OrderBy("id").Pagination(10, 0).Count()`

### UPDATE
- **Pair-based**: `Update("name", "John").Where("id = $1", 123).Exec()`
- **Struct-based**: `Table(user).Update().Where("id = $1", 123).Exec()`

### DELETE
- **With condition**: `Delete().Where("id = $1", 123).Exec()`

### Context
- **With context**: `.Exec(ctx)` or `.Count(ctx)`
- **Without context**: `.Exec()` or `.Count()` - Uses `context.Background()`

Happy coding! 🚀
