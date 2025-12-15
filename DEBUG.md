# Debug Mode

Norm supports an optional debug mode that can be enabled via environment variable to show detailed logging information during development.

## Enabling Debug Mode

Set the `NORM_DEBUG` environment variable before running your application:

### Linux/Mac
```bash
export NORM_DEBUG=true
go run main.go
```

Or inline:
```bash
NORM_DEBUG=true go run main.go
```

### Windows (PowerShell)
```powershell
$env:NORM_DEBUG="true"
go run main.go
```

### Windows (CMD)
```cmd
set NORM_DEBUG=true
go run main.go
```

## What Debug Mode Shows

When debug mode is enabled, you'll see additional logging for:

### Cache Operations
```
[CACHE] Key: users:active:abc123hash...
[CACHE] Status: MISS (Pulling from DB)
```

```
[CACHE] Key: users:active:abc123hash...
[CACHE] Status: HIT
```

### Query Routing
```
[DEBUG] getShardPool table=users shard=shard1 role=primary queryType=select hasStandalone=false
[DEBUG] Looking for table 'users' in standalone pools of shard 'shard1'. Available: map[]
```

### Join Operations
```
[DEBUG] Executing App-Side Join (Distributed/Skey)
```

### Query Results
When `dest` is `nil` in query methods, debug mode will print formatted results:
```
Query Results (3 rows) [CACHE]:
fullname             | useremail            
------------------------------------------------------------
Alice Williams       | alice@example.com    
Bob Brown            | bob@example.com      
Charlie Davis        | charlie@example.com  
```

## Production Mode (Default)

When `NORM_DEBUG` is not set or set to any value other than `true`, `1`, or `on`:

- **No debug logs** are printed
- **Only errors** are shown with detailed context
- **Silent operation** for successful queries
- **Optimal performance** (no logging overhead)

### Error Logging (Always Shown)

Even without debug mode, errors are always logged with context:

```
[ERROR] Failed to execute query: connection refused
  Details:
    table: users
    query_type: select
    shard: shard1
```

## Best Practices

1. **Development:** Enable debug mode to understand query routing and caching behavior
   ```bash
   NORM_DEBUG=true go run main.go
   ```

2. **Testing:** Enable for integration tests to verify cache hits/misses
   ```bash
   NORM_DEBUG=true go test ./...
   ```

3. **Production:** Never enable debug mode (performance and log volume)
   ```bash
   # NORM_DEBUG not set
   go run main.go
   ```

4. **Debugging Issues:** Temporarily enable to diagnose problems
   ```bash
   NORM_DEBUG=true ./your-app
   ```

## Programmatic Check

You can check if debug mode is enabled in your code:

```go
import "github.com/skssmd/norm/core/engine"

if engine.IsDebugMode() {
    // Debug mode is enabled
    fmt.Println("Running in debug mode")
}
```

## Environment Variable Values

The following values enable debug mode (case-insensitive):
- `true`
- `1`
- `on`

Any other value (or unset) disables debug mode:
- `false`
- `0`
- `off`
- `` (empty/unset)
