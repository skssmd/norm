# SELECT Operations

Complete guide to SELECT queries and struct scanning in Norm ORM.

## Table of Contents
- [Overview](#overview)
- [Basic SELECT](#basic-select)
- [Struct Scanning](#struct-scanning)
- [Best Practices](#best-practices)

---

## Overview

Norm provides powerful SELECT operations with:
- ✅ **Flexible querying** - WHERE, ORDER BY, LIMIT, OFFSET
- ✅ **Struct scanning** - Automatic mapping to Go structs
- ✅ **Join support** - Native and distributed joins
- ✅ **Type-safe** - Compile-time type checking

---

## Basic SELECT

### Simple SELECT

```go
count, err := norm.Table("users").
    Select("id", "name", "email").
    Count()
```

### SELECT with WHERE

```go
count, err := norm.Table("users").
    Select("name", "email", "created_at").
    Where("created_at > $1", time.Now().AddDate(0, -1, 0)).
    Count()
```

### SELECT All Fields

```go
count, err := norm.Table("users").
    Select().  // Selects all fields (*)
    Count()
```

### SELECT with Pagination

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

## Struct Scanning

### Scan Single Row with First()

```go
type User struct {
    ID       uint   `norm:"index;notnull;pk;auto"`
    Name     string `norm:"name:fullname;notnull"`
    Email    string `norm:"name:useremail;unique;notnull"`
    Username string `norm:"name:uname;notnull;unique"`
}

var user User

err := norm.Table("users").
    Select().
    Where("username = $1", "alice").
    First(ctx, &user)

if err != nil {
    log.Fatal(err)
}

fmt.Printf("User: %s (%s)\n", user.Name, user.Email)
```

### Scan Multiple Rows with All()

```go
var users []User

err := norm.Table("users").
    Select().
    Where("created_at > $1", time.Now().AddDate(0, -1, 0)).
    OrderBy("created_at DESC").
    Limit(10).
    All(ctx, &users)

if err != nil {
    log.Fatal(err)
}

for _, u := range users {
    fmt.Printf("- %s\n", u.Name)
}
```

### Scan with Field Mapping Tags

Use `norm:"name:column_name"` tags to map struct fields to specific columns:

```go
type UserAndProfile struct {
    UserName string `norm:"name:fullname"`        // Maps to "fullname" column
    UserBio  string `norm:"name:bio"`             // Maps to "bio" column
    UID      int    `norm:"name:user_id_alias"`   // Maps to aliased column
}

var userAndProfiles []UserAndProfile

err := norm.Table("users", "id", "profiles", "user_id").
    Select("users.fullname", "profiles.bio", "users.id as user_id_alias").
    Where("users.username = $1", "alice").
    All(ctx, &userAndProfiles)
```

### Field Mapping Priority

Norm maps columns to struct fields in this order:

1. **Exact match with `norm:"name:column_name"` tag**
2. **Table-prefixed match** (e.g., `users.fullname` → field with `name:fullname`)
3. **Case-insensitive field name match**

### Scanning Partial Results

You don't need to include all columns in your struct:

```go
type UserSummary struct {
    Name  string `norm:"name:fullname"`
    Email string `norm:"name:useremail"`
    // Other User fields are ignored
}

var summaries []UserSummary

err := norm.Table("users").
    Select("fullname", "useremail").  // Only select what you need
    All(ctx, &summaries)
```

---

## Best Practices

### 1. Use Struct Scanning

```go
// ✅ Good - Type-safe
var users []User
norm.Table("users").Select().All(ctx, &users)

// ❌ Less safe - No type checking
norm.Table("users").Select().All(ctx, nil)
```

### 2. Select Only What You Need

```go
// ✅ Good - Efficient
norm.Table("users").Select("id", "name").All(ctx, &users)

// ❌ Bad - Wasteful
norm.Table("users").Select().All(ctx, &users)  // Selects all columns
```

### 3. Use Pagination for Large Results

```go
// ✅ Good - Paginated
norm.Table("users").
    Select().
    OrderBy("created_at DESC").
    Pagination(100, 0).
    All(ctx, &users)
```

### 4. Use Context for Timeouts

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

norm.Table("users").Select().All(ctx, &users)
```

---

## Next Steps

- Learn about [JOIN Operations](./09-joins.md) for combining tables
- Explore [UPDATE Operations](./07-update.md) for modifying records
- See [Raw SQL](./10-raw-sql.md) for custom queries
