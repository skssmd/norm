# INSERT Operations

Complete guide to INSERT operations in Norm ORM.

## Table of Contents
- [Overview](#overview)
- [Single Row Insert](#single-row-insert)
- [Bulk Insert](#bulk-insert)
- [Upsert (ON CONFLICT)](#upsert-on-conflict)
- [Best Practices](#best-practices)
- [Complete Examples](#complete-examples)

---

## Overview

Norm provides flexible INSERT operations with:
- ✅ **Struct-based inserts** - Type-safe with automatic field extraction
- ✅ **Bulk inserts** - Efficient multi-row insertion
- ✅ **Upsert support** - Handle conflicts gracefully
- ✅ **Context support** - Timeouts and cancellation

---

## Single Row Insert

### Struct-Based Insert (Recommended)

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

### Insert with Context

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

rowsAffected, err := norm.Table(user).Insert().Exec(ctx)
if err != nil {
    log.Fatal(err)
}
```

---

## Bulk Insert

### Struct-Based Bulk Insert (Recommended)

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

if err != nil {
    log.Fatal(err)
}
fmt.Printf("Inserted %d users\n", rowsAffected)
```

**Generated SQL:**
```sql
INSERT INTO users (email, name, username) 
VALUES ($1, $2, $3), ($4, $5, $6), ($7, $8, $9)
```

### Generate Bulk Data in Loop

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

### Manual Bulk Insert (Legacy)

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

## Upsert (ON CONFLICT)

### Insert or Keep Existing

Insert or keep old value if conflict occurs:

```go
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

### Insert or Update

Update specific columns on conflict:

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

### Example 2: Bulk Import Users

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

### Example 3: Insert with Validation

```go
func CreateUser(user User) error {
    // Validate before insert
    if user.Email == "" {
        return errors.New("email is required")
    }
    
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    rowsAffected, err := norm.Table(user).
        Insert().
        Exec(ctx)
    
    if err != nil {
        return fmt.Errorf("failed to create user: %w", err)
    }
    
    if rowsAffected == 0 {
        return errors.New("user not created")
    }
    
    return nil
}
```

---

## Quick Reference

### Single Insert
```go
norm.Table(user).Insert().Exec()
norm.Table(user).Insert().Exec(ctx)
```

### Bulk Insert
```go
norm.Table("users").BulkInsert(users).Exec()
```

### Upsert
```go
norm.Table(user).Insert().OnConflict("email", "nothing").Exec()
norm.Table(user).Insert().OnConflict("email", "update", "name").Exec()
```

---

## Next Steps

- Learn about [SELECT Operations](./06-select.md) for querying data
- Explore [UPDATE Operations](./07-update.md) for modifying records
- See [Model Definition](./02-model-definition.md) for struct tags
