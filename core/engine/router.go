package engine

import (
	"context"
	"fmt"
	"reflect"
	"sort"

	"github.com/skssmd/norm/core/driver"
	"github.com/skssmd/norm/core/registry"
)

// Query represents an executable query with routing
type Query struct {
	builder *QueryBuilder
	table   string
	model   interface{}
}

// From creates a new query with routing from model
func (q *Query) From(model interface{}) *Query {
	q.model = model
	q.builder = From(model)
	q.table = q.builder.tableName
	return q
}

// Table sets the table name or model for the query
// Accepts either string (table name) or struct (model with data)
func (q *Query) Table(tableNameOrModel interface{}) *Query {
	switch v := tableNameOrModel.(type) {
	case string:
		// String-based: just set table name
		q.table = v
		q.builder = &QueryBuilder{
			tableName:    v,
			columns:      []string{},
			whereArgs:    []interface{}{},
			updateFields: make(map[string]interface{}),
			insertFields: make(map[string]interface{}),
		}
	default:
		// Struct-based: extract table name and non-zero fields
		tableName := getTableNameFromModel(v)
		q.table = tableName
		q.builder = &QueryBuilder{
			tableName:    tableName,
			model:        v,
			columns:      []string{},
			whereArgs:    []interface{}{},
			updateFields: make(map[string]interface{}),
			insertFields: make(map[string]interface{}),
		}
	}
	return q
}

// Select specifies columns to select
func (q *Query) Select(fields ...interface{}) *Query {
	q.builder.Select(fields...)
	return q
}

// Where adds WHERE clause
func (q *Query) Where(condition string, args ...interface{}) *Query {
	q.builder.Where(condition, args...)
	return q
}

// Update sets fields to update
// Can be used in two ways:
// 1. Pair-based: Update("name", "John", "age", 30)
// 2. Struct-based: Table(User{Name: "John"}).Update().Where(...)
func (q *Query) Update(args ...interface{}) *Query {
	if len(args) == 0 {
		// Struct-based update from Table()
		if q.builder.model != nil {
			q.builder.queryType = "update"
			q.builder.updateFields = q.builder.extractFieldsFromModel(q.builder.model)
		}
		return q
	}

	// Pair-based API: Update("name", "John", "age", 30)
	q.builder.queryType = "update"
	q.builder.updateFields = make(map[string]interface{})

	for i := 0; i < len(args)-1; i += 2 {
		if key, ok := args[i].(string); ok {
			q.builder.updateFields[key] = args[i+1]
		}
	}

	return q
}

// Delete marks as delete query
func (q *Query) Delete() *Query {
	q.builder.Delete()
	return q
}

// Insert sets up an insert operation
// If model is provided, use it; otherwise use the model from Table()
func (q *Query) Insert(model ...interface{}) *Query {
	if len(model) > 0 {
		q.builder.Insert(model[0])
	} else if q.builder.model != nil {
		// Use model from Table() and extract non-zero fields
		q.builder.InsertNonZero(q.builder.model)
	}
	return q
}

// OrderBy adds ORDER BY clause
func (q *Query) OrderBy(order string) *Query {
	q.builder.OrderBy(order)
	return q
}

// Limit sets LIMIT
func (q *Query) Limit(limit int) *Query {
	q.builder.Limit(limit)
	return q
}

// Offset sets OFFSET
func (q *Query) Offset(offset int) *Query {
	q.builder.Offset(offset)
	return q
}

// Pagination sets limit and offset
func (q *Query) Pagination(limit, offset int) *Query {
	q.builder.Pagination(limit, offset)
	return q
}

// BulkInsert sets up bulk insert
// Can accept either:
// 1. Slice of structs: BulkInsert([]User{user1, user2, user3})
// 2. Manual columns and rows: BulkInsert([]string{"name", "email"}, [][]interface{}{...})
func (q *Query) BulkInsert(args ...interface{}) *Query {
	q.builder.queryType = "bulkinsert"

	if len(args) == 0 {
		return q
	}

	// Check if first arg is []string (manual mode)
	if columns, ok := args[0].([]string); ok && len(args) > 1 {
		// Manual mode: BulkInsert([]string{"name", "email"}, [][]interface{}{...})
		if rows, ok := args[1].([][]interface{}); ok {
			q.builder.bulkColumns = columns
			q.builder.bulkRows = rows
		}
		return q
	}

	// Struct-based mode: extract from slice of structs
	q.extractBulkFromModels(args[0])
	return q
}

// extractBulkFromModels extracts columns and rows from a slice of structs
func (q *Query) extractBulkFromModels(models interface{}) {
	modelsValue := reflect.ValueOf(models)

	if modelsValue.Kind() != reflect.Slice {
		return
	}

	if modelsValue.Len() == 0 {
		return
	}

	// Get first model to determine columns
	firstModel := modelsValue.Index(0).Interface()
	firstFields := q.builder.extractFieldsFromModel(firstModel)

	// Extract column names (sorted for consistency)
	columns := make([]string, 0, len(firstFields))
	for col := range firstFields {
		columns = append(columns, col)
	}
	sort.Strings(columns)

	q.builder.bulkColumns = columns

	// Extract rows
	rows := make([][]interface{}, 0, modelsValue.Len())
	for i := 0; i < modelsValue.Len(); i++ {
		model := modelsValue.Index(i).Interface()
		fields := q.builder.extractFieldsFromModel(model)

		row := make([]interface{}, len(columns))
		for j, col := range columns {
			row[j] = fields[col]
		}
		rows = append(rows, row)
	}

	q.builder.bulkRows = rows
}

// OnConflict specifies what to do on conflict
// action can be "nothing" (keep old value) or "update" (replace with new value)
// Example: OnConflict("email", "nothing") - keep old value if email exists
// Example: OnConflict("email", "update", "name", "updated_at") - update specific columns on conflict
func (q *Query) OnConflict(conflictColumn string, action string, updateColumns ...string) *Query {
	q.builder.onConflict = conflictColumn
	q.builder.conflictAction = action
	q.builder.conflictUpdates = updateColumns
	return q
}

// getPool determines which pool to use based on query type and table
func (q *Query) getPool() (*driver.PGPool, error) {
	// Get registry info
	info := registry.GetRegistryInfo()
	mode := info["mode"].(string)

	switch mode {
	case "global":
		return q.getGlobalPool(info)
	case "shard":
		return q.getShardPool(info)
	default:
		return nil, fmt.Errorf("unknown registry mode: %s", mode)
	}
}

// getGlobalPool gets pool for global mode
func (q *Query) getGlobalPool(info map[string]interface{}) (*driver.PGPool, error) {
	pools := info["pools"].(map[string]*driver.PGPool)
	queryType := q.builder.queryType

	// Detect scenario based on available pools
	hasReadWrite := pools["read"] != nil || pools["write"] != nil
	hasPrimaryReplica := pools["primary"] != nil && pools["replica"] != nil

	if hasPrimaryReplica && !hasReadWrite {
		// Scenario: Global Primary/Replica
		// Primary for all writes, fallback to replica when primary down
		switch queryType {
		case "select":
			// Try primary first, fallback to replica when primary down
			if pool, ok := pools["primary"]; ok {
				return pool, nil
			}
			if pool, ok := pools["replica"]; ok {
				return pool, nil
			}
		case "insert", "update", "delete", "bulkinsert":
			// Always use primary for writes
			if pool, ok := pools["primary"]; ok {
				return pool, nil
			}
		}
	} else if hasReadWrite {
		// Scenario: Global Read/Write Split
		// Use role-based routing, fallback read to write when down
		switch queryType {
		case "select":
			// Try read pool first, fallback to write when read down
			if pool, ok := pools["read"]; ok {
				return pool, nil
			}
			if pool, ok := pools["write"]; ok {
				return pool, nil
			}
			// Last resort: primary
			if pool, ok := pools["primary"]; ok {
				return pool, nil
			}
		case "insert", "update", "delete", "bulkinsert":
			// Try write pool first, fallback to primary
			if pool, ok := pools["write"]; ok {
				return pool, nil
			}
			if pool, ok := pools["primary"]; ok {
				return pool, nil
			}
		}
	} else {
		// Single pool scenario
		if pool, ok := pools["primary"]; ok {
			return pool, nil
		}
	}

	return nil, fmt.Errorf("no suitable pool found for query type: %s", queryType)
}

// getShardPool gets pool for shard mode
func (q *Query) getShardPool(info map[string]interface{}) (*driver.PGPool, error) {
	// Get table mapping
	tableMapping, err := registry.GetTableMapping(q.table)
	if err != nil {
		return nil, fmt.Errorf("table '%s' not registered: %w", q.table, err)
	}

	shardName := tableMapping.ShardName()
	role := tableMapping.Role()

	shards := info["shards"].(map[string]interface{})
	shardInfo, ok := shards[shardName]
	if !ok {
		return nil, fmt.Errorf("shard '%s' not found", shardName)
	}

	shardData := shardInfo.(map[string]interface{})

	// Get primary pool
	var primaryPool *driver.PGPool
	if pp, ok := shardData["primary_pool"]; ok && pp != nil {
		primaryPool = pp.(*driver.PGPool)
	}

	// Get standalone pools
	var standalonePools map[string]*driver.PGPool
	if sp, ok := shardData["standalone_pools"]; ok && sp != nil {
		standalonePools = sp.(map[string]*driver.PGPool)
	}

	queryType := q.builder.queryType

	// Scenario: Shards - Use table-specific shard or primary
	switch queryType {
	case "insert", "update", "delete", "bulkinsert":
		// Writes: Use table's registered role pool, fallback to primary
		if role == "standalone" && standalonePools != nil {
			// For standalone tables, use the first standalone pool
			for _, pool := range standalonePools {
				return pool, nil
			}
		}
		// Fallback to primary for writes
		if primaryPool != nil {
			return primaryPool, nil
		}

	case "select":
		// Reads: Use table's registered role pool, fallback to primary
		if role == "standalone" && standalonePools != nil {
			// For standalone tables, use the first standalone pool
			for _, pool := range standalonePools {
				return pool, nil
			}
		}
		// Fallback to primary
		if primaryPool != nil {
			return primaryPool, nil
		}
	}

	return nil, fmt.Errorf("no suitable pool found for table '%s' in shard '%s'", q.table, shardName)
}

// Exec executes the query (for INSERT, UPDATE, DELETE)
// Context is optional - if not provided, uses context.Background()
func (q *Query) Exec(ctx ...context.Context) (int64, error) {
	// Use provided context or default to Background
	execCtx := context.Background()
	if len(ctx) > 0 {
		execCtx = ctx[0]
	}

	pool, err := q.getPool()
	if err != nil {
		return 0, err
	}

	sql, args, err := q.builder.Build()
	if err != nil {
		return 0, err
	}

	// Debug: print SQL and args (uncomment for debugging)
	// fmt.Printf("DEBUG SQL: %s\nDEBUG ARGS: %v\n", sql, args)

	result, err := pool.Pool.Exec(execCtx, sql, args...)
	if err != nil {
		return 0, fmt.Errorf("query execution failed: %w", err)
	}

	return result.RowsAffected(), nil
}

// First executes query and returns first row
func (q *Query) First(ctx context.Context, dest interface{}) error {
	pool, err := q.getPool()
	if err != nil {
		return err
	}

	// Set limit to 1 for First()
	q.builder.Limit(1)

	sql, args, err := q.builder.Build()
	if err != nil {
		return err
	}

	row := pool.Pool.QueryRow(ctx, sql, args...)
	// Scanning will be handled by the caller
	// For now, return the row interface
	_ = row

	// TODO: Implement proper scanning based on dest type
	return fmt.Errorf("First() scanning not yet implemented")
}

// All executes query and returns all rows
func (q *Query) All(ctx context.Context, dest interface{}) error {
	pool, err := q.getPool()
	if err != nil {
		return err
	}

	sql, args, err := q.builder.Build()
	if err != nil {
		return err
	}

	rows, err := pool.Pool.Query(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	// TODO: Implement proper scanning based on dest type
	_ = rows
	return fmt.Errorf("All() scanning not yet implemented")
}

// Count executes a COUNT query
// Context is optional - if not provided, uses context.Background()
func (q *Query) Count(ctx ...context.Context) (int64, error) {
	// Use provided context or default to Background
	execCtx := context.Background()
	if len(ctx) > 0 {
		execCtx = ctx[0]
	}

	pool, err := q.getPool()
	if err != nil {
		return 0, err
	}

	// Store original query state
	originalQueryType := q.builder.queryType
	originalOrderBy := q.builder.orderBy
	originalLimit := q.builder.limit
	originalOffset := q.builder.offset

	// Modify for COUNT
	q.builder.queryType = "select"
	q.builder.columns = []string{"COUNT(*)"}
	q.builder.orderBy = ""
	q.builder.limit = 0
	q.builder.offset = 0

	sql, args, err := q.builder.Build()
	if err != nil {
		return 0, err
	}

	// Restore original state
	q.builder.queryType = originalQueryType
	q.builder.orderBy = originalOrderBy
	q.builder.limit = originalLimit
	q.builder.offset = originalOffset

	var count int64
	err = pool.Pool.QueryRow(execCtx, sql, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count query failed: %w", err)
	}

	return count, nil
}
