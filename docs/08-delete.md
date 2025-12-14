# DELETE Operations

Complete guide to DELETE operations in Norm ORM.

## Table of Contents
- [Overview](#overview)
- [Basic DELETE](#basic-delete)
- [Soft Delete Pattern](#soft-delete-pattern)
- [Best Practices](#best-practices)

---

## Overview

Norm provides simple DELETE operations with:
- ✅ **Conditional deletes** - WHERE clause support
- ✅ **Context support** - Timeouts and cancellation
- ✅ **Soft delete pattern** - Mark as deleted instead of removing

---

## Basic DELETE

### Delete with Condition

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

### Delete with Multiple Conditions

```go
rowsAffected, err := norm.Table("users").
    Delete().
    Where("status = $1 AND created_at < $2", "inactive", time.Now().AddDate(-1, 0, 0)).
    Exec()
```

### Delete with Context

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

rowsAffected, err := norm.Table("users").
    Delete().
    Where("id = $1", 123).
    Exec(ctx)
```

---

## Soft Delete Pattern

Instead of permanently deleting records, mark them as deleted:

```go
func SoftDeleteUser(userID uint) error {
    _, err := norm.Table("users").
        Update("deleted_at", time.Now(), "status", "deleted").
        Where("id = $1", userID).
        Exec()
    
    return err
}
```

### Query Excluding Soft-Deleted Records

```go
var users []User

err := norm.Table("users").
    Select().
    Where("deleted_at IS NULL").
    All(ctx, &users)
```

---

## Best Practices

### 1. Always Use WHERE Clause

```go
// ⚠️ DANGER: Deletes ALL rows!
norm.Table("users").Delete().Exec()

// ✅ Good: Deletes specific rows
norm.Table("users").Delete().Where("id = $1", 123).Exec()
```

### 2. Consider Soft Deletes

```go
// ✅ Soft delete - can be recovered
norm.Table("users").
    Update("deleted_at", time.Now()).
    Where("id = $1", 123).
    Exec()

// ⚠️ Hard delete - permanent
norm.Table("users").
    Delete().
    Where("id = $1", 123).
    Exec()
```

### 3. Check Rows Affected

```go
rowsAffected, err := norm.Table("users").
    Delete().
    Where("id = $1", 123).
    Exec()

if err != nil {
    return fmt.Errorf("delete failed: %w", err)
}

if rowsAffected == 0 {
    return errors.New("user not found")
}
```

---

## Complete Example

```go
func DeleteInactiveUsers(days int) (int64, error) {
    cutoffDate := time.Now().AddDate(0, 0, -days)
    
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    
    rowsAffected, err := norm.Table("users").
        Delete().
        Where("status = $1 AND last_login < $2", "inactive", cutoffDate).
        Exec(ctx)
    
    if err != nil {
        return 0, fmt.Errorf("failed to delete inactive users: %w", err)
    }
    
    return rowsAffected, nil
}
```

---

## Next Steps

- Learn about [UPDATE Operations](./07-update.md)
- Explore [INSERT Operations](./05-insert.md)
- See [SELECT Operations](./06-select.md)
