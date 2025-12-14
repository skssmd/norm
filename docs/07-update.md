# UPDATE Operations

Complete guide to UPDATE operations in Norm ORM.

## Table of Contents
- [Overview](#overview)
- [Pair-Based UPDATE](#pair-based-update)
- [Struct-Based UPDATE](#struct-based-update)
- [Best Practices](#best-practices)

---

## Overview

Norm provides two ways to update data:
- ✅ **Pair-based** - Explicit key-value updates (can set to zero)
- ✅ **Struct-based** - Partial updates (ignores zero values)

---

## Pair-Based UPDATE

### Update Single Field

```go
rowsAffected, err := norm.Table("users").
    Update("name", "John Updated").
    Where("username = $1", "johndoe").
    Exec()
```

### Update Multiple Fields

```go
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

---

## Struct-Based UPDATE

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

---

## Comparison

| Method | Zero Values | Use Case |
|--------|-------------|----------|
| **Pair-based** | Always included | Explicit updates, can set to zero |
| **Struct-based** | Ignored | Partial updates, keep old values |

**Examples:**

```go
// ✅ Pair-based: Set age to 0 explicitly
norm.Table("users").Update("age", 0).Where("id = $1", 123).Exec()

// ✅ Struct-based: Update only name, keep everything else
norm.Table(User{Name: "John"}).Update().Where("id = $1", 123).Exec()
```

---

## Best Practices

### 1. Use Pair-Based for Explicit Updates

```go
// ✅ Clear and explicit
norm.Table("users").Update("status", "inactive").Where("id = $1", 123).Exec()
```

### 2. Use Context for Timeouts

```go
ctx := context.Background()
rowsAffected, err := norm.Table("users").
    Update("status", "active").
    Where("id = $1", 123).
    Exec(ctx)
```

### 3. Always Use WHERE Clause

```go
// ⚠️ Warning: Updates ALL rows!
norm.Table("users").Update("status", "active").Exec()

// ✅ Good: Updates specific rows
norm.Table("users").Update("status", "active").Where("id = $1", 123).Exec()
```

---

## Next Steps

- Learn about [DELETE Operations](./08-delete.md)
- Explore [INSERT Operations](./05-insert.md)
- See [SELECT Operations](./06-select.md)
