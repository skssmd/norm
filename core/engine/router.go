package engine

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/skssmd/norm/core/driver"
	"github.com/skssmd/norm/core/registry"
)

// Query represents an executable query with routing
type Query struct {
	builder     *QueryBuilder
	table       string
	model       interface{}
	joinContext *JoinContext
}

// JoinContext holds information for join operations
type JoinContext struct {
	Tables []string
	Keys   []string
	Models []interface{}
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
// Table sets the table name or model for the query
// Accepts either string (table name) or struct (model with data)
// Supports join syntax: Table("users", "id", "orders", "user_id")
func (q *Query) Table(args ...interface{}) *Query {
	if len(args) == 0 {
		return q
	}

	// Single argument case (standard query)
	if len(args) == 1 {
		tableNameOrModel := args[0]
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
				joins:        []JoinDefinition{},
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
				joins:        []JoinDefinition{},
			}
		}
		return q
	}

	// Join case
	q.joinContext = &JoinContext{
		Tables: make([]string, 0),
		Keys:   make([]string, 0),
		Models: make([]interface{}, 0),
	}

	// Parse args in pairs (Table, Key)
	for i := 0; i < len(args); i += 2 {
		if i+1 >= len(args) {
			break // Should handle error, but for now ignore incomplete pair
		}

		tableArg := args[i]
		keyArg := args[i+1]

		var tableName string
		var model interface{}

		// Handle Table argument
		switch v := tableArg.(type) {
		case string:
			tableName = v
		default:
			tableName = getTableNameFromModel(v)
			model = v
		}

		// Handle Key argument
		var keyName string
		switch v := keyArg.(type) {
		case string:
			keyName = v
		default:
			// Extract field name from pointer if possible, or use struct tag
			// For now assume string or implement pointer extraction later
			// If it's a pointer to a field, we need to resolve it
			// This requires the model to be set
			if model != nil {
				// Try to resolve field name from pointer
				// This is complex without the builder context yet
				// For now, assume string keys for simplicity in this step
				keyName = fmt.Sprintf("%v", v)
			} else {
				keyName = fmt.Sprintf("%v", v)
			}
		}

		q.joinContext.Tables = append(q.joinContext.Tables, tableName)
		q.joinContext.Keys = append(q.joinContext.Keys, keyName)
		q.joinContext.Models = append(q.joinContext.Models, model)
	}

	// Initialize builder with the first table
	if len(q.joinContext.Tables) > 0 {
		firstTable := q.joinContext.Tables[0]
		firstModel := q.joinContext.Models[0]

		q.table = firstTable
		q.builder = &QueryBuilder{
			tableName:    firstTable,
			model:        firstModel,
			columns:      []string{},
			whereArgs:    []interface{}{},
			updateFields: make(map[string]interface{}),
			insertFields: make(map[string]interface{}),
			joins:        []JoinDefinition{},
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
	poolsRaw := info["pools"].(map[string]interface{})
	queryType := q.builder.queryType

	// Convert interface{} map to typed map
	pools := make(map[string]*driver.PGPool)
	for name, poolInterface := range poolsRaw {
		if pool, ok := poolInterface.(*driver.PGPool); ok {
			pools[name] = pool
		}
	}

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
	tableModel, exists := registry.GetModel(q.table)
	if !exists {
		return nil, fmt.Errorf("table '%s' not registered", q.table)
	}

	// Find the shard for the table for any role
	var shardName, role string
	found := false
	for r, shards := range tableModel.Roles {
		for s := range shards {
			shardName = s
			role = r
			found = true
			break
		}
		if found {
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("no shard found for table '%s'", q.table)
	}

	// Lookup shard info in registry
	shards := info["shards"].(map[string]interface{})
	shardInfoRaw, ok := shards[shardName]
	if !ok {
		return nil, fmt.Errorf("shard '%s' not found", shardName)
	}

	shardInfo := shardInfoRaw.(map[string]interface{})

	// Get primary pool
	var primaryPool *driver.PGPool
	if pp, ok := shardInfo["primary_pool"]; ok && pp != nil {
		primaryPool = pp.(*driver.PGPool)
	}

	// Get standalone pool for this table
	var standalonePool *driver.PGPool
	if spRaw, ok := shardInfo["standalone_pools"]; ok && spRaw != nil {
		if spMap, ok := spRaw.(map[string]*driver.PGPool); ok {
			// DEBUG: Print available standalone pools
			fmt.Printf("DEBUG: Looking for table '%s' in standalone pools of shard '%s'. Available: %v\n", q.table, shardName, spMap)
			if pool, ok := spMap[q.table]; ok {
				standalonePool = pool
			}
		}
	}

	queryType := q.builder.queryType
	fmt.Printf("DEBUG: getShardPool table=%s shard=%s role=%s queryType=%s hasStandalone=%v\n", q.table, shardName, role, queryType, standalonePool != nil)
	switch queryType {
	case "insert", "update", "delete", "bulkinsert":
		if role == "standalone" && standalonePool != nil {
			return standalonePool, nil
		}
		if primaryPool != nil {
			return primaryPool, nil
		}
	case "select":
		if role == "standalone" && standalonePool != nil {
			return standalonePool, nil
		}
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
	if q.joinContext != nil {
		return q.executeJoin(ctx, dest, true) // true for single row
	}
	return q.executeStandard(ctx, dest, true)
}

// All executes query and returns all rows
func (q *Query) All(ctx context.Context, dest interface{}) error {
	if q.joinContext != nil {
		return q.executeJoin(ctx, dest, false)
	}
	return q.executeStandard(ctx, dest, false)
}

// executeStandard executes a standard single-table query
func (q *Query) executeStandard(ctx context.Context, dest interface{}, singleRow bool) error {
	pool, err := q.getPool()
	if err != nil {
		return err
	}

	if singleRow {
		q.builder.Limit(1)
	}

	sql, args, err := q.builder.Build()
	if err != nil {
		return err
	}

	// Execute query
	rows, err := pool.Pool.Query(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	if dest != nil {
		return scanRowsToDest(rows, dest)
	}

	// If dest is nil, print results for demo purposes
	if dest == nil {
		results, err := scanRowsToMap(rows)
		if err != nil {
			return fmt.Errorf("failed to scan rows: %w", err)
		}

		fmt.Printf("\nQuery Results (%d rows):\n", len(results))
		if len(results) > 0 {
			// Get headers from first row
			headers := make([]string, 0, len(results[0]))
			for k := range results[0] {
				headers = append(headers, k)
			}
			sort.Strings(headers)

			// Print headers
			for _, h := range headers {
				fmt.Printf("%-20s | ", h)
			}
			fmt.Println()
			fmt.Println(strings.Repeat("-", len(headers)*23))

			// Print rows
			for _, row := range results {
				for _, h := range headers {
					val := fmt.Sprintf("%v", row[h])
					if len(val) > 18 {
						val = val[:15] + "..."
					}
					fmt.Printf("%-20s | ", val)
				}
				fmt.Println()
			}
		}
		return nil
	}
	
	return nil
}

// scanRowsToMap scans rows into a slice of maps
func scanRowsToMap(rows pgx.Rows) ([]map[string]interface{}, error) {
	fields := rows.FieldDescriptions()
	var results []map[string]interface{}

	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, err
		}

		row := make(map[string]interface{})
		for i, field := range fields {
			row[string(field.Name)] = values[i]
		}
		results = append(results, row)
	}
	return results, nil
}

// executeJoin executes a join query (either native or app-side)
func (q *Query) executeJoin(ctx context.Context, dest interface{}, singleRow bool) error {
	if len(q.joinContext.Tables) < 2 {
		return fmt.Errorf("join requires at least 2 tables")
	}

	// 1. Check co-location
	coLocated, err := q.isCoLocated()
	if err != nil {
		return err
	}

	// 2. Decide Native vs App-Side
	if coLocated {
		// Configure Native Join
		for i := 1; i < len(q.joinContext.Tables); i++ {
			t1 := q.joinContext.Tables[i-1]
			k1 := q.joinContext.Keys[i-1]
			t2 := q.joinContext.Tables[i]
			k2 := q.joinContext.Keys[i]

			onClause := fmt.Sprintf("%s.%s = %s.%s", t1, k1, t2, k2)
			
			q.builder.joins = append(q.builder.joins, JoinDefinition{
				Table: t2,
				On:    onClause,
				Type:  "INNER",
			})
		}
		return q.executeStandard(ctx, dest, singleRow)
	}

	// 3. App-Side Join
	return q.executeAppSideJoin(ctx, dest, singleRow)
}

// isCoLocated checks if all tables in the join context are on the same database/shard
func (q *Query) isCoLocated() (bool, error) {
	info := registry.GetRegistryInfo()
	mode := info["mode"].(string)

	if mode == "global" {
		return true, nil
	}

	if mode != "shard" {
		return false, fmt.Errorf("unknown registry mode: %s", mode)
	}

	// Check shards for each table
	var firstShard string
	
	for i, tableName := range q.joinContext.Tables {
		tableModel, exists := registry.GetModel(tableName)
		if !exists {
			return false, fmt.Errorf("table '%s' not registered", tableName)
		}

		// Find shard for this table
		var shardName string
		found := false
		for _, shards := range tableModel.Roles {
			for s := range shards {
				shardName = s
				found = true
				break
			}
			if found {
				break
			}
		}

		if !found {
			return false, fmt.Errorf("no shard found for table '%s'", tableName)
		}

		if i == 0 {
			firstShard = shardName
		} else {
			if shardName != firstShard {
				return false, nil // Different shards
			}
		}
	}

	// Check for Skeys
	for i, tableName := range q.joinContext.Tables {
		key := q.joinContext.Keys[i]
		tableModel, _ := registry.GetModel(tableName)
		if tableModel != nil {
			for _, field := range tableModel.Fields {
				if field.Fieldname == key && field.Skey != "" {
					return false, nil // Force App-Side
				}
			}
		}
	}

	return true, nil
}

// executeAppSideJoin executes a join by fetching data from multiple sources and merging
func (q *Query) executeAppSideJoin(ctx context.Context, dest interface{}, singleRow bool) error {
	fmt.Println("  🔄 Executing App-Side Join (Distributed/Skey)...")

	// 1. Fetch T1
	// We need to filter columns for T1
	t1 := q.joinContext.Tables[0]
	k1 := q.joinContext.Keys[0]
	t2 := q.joinContext.Tables[1]
	k2 := q.joinContext.Keys[1]

	originalCols := q.builder.columns
	var cols1, cols2 []string

	if len(originalCols) > 0 {
		for _, col := range originalCols {
			if strings.HasPrefix(col, t2+".") {
				cols2 = append(cols2, col)
			} else {
				// Assume T1 if T1 prefix or no prefix
				cols1 = append(cols1, col)
			}
		}
	}

	// Ensure K1 is in cols1 (if we have specific columns)
	if len(cols1) > 0 {
		hasK1 := false
		for _, col := range cols1 {
			if col == k1 || col == t1+"."+k1 {
				hasK1 = true
				break
			}
		}
		if !hasK1 {
			cols1 = append(cols1, t1+"."+k1)
		}
		q.builder.columns = cols1
	}

	pool1, err := q.getPool()
	if err != nil {
		return err
	}

	sql1, args1, err := q.builder.Build()
	// Restore original columns just in case
	q.builder.columns = originalCols
	if err != nil {
		return err
	}

	rows1, err := pool1.Pool.Query(ctx, sql1, args1...)
	if err != nil {
		return fmt.Errorf("failed to fetch T1: %w", err)
	}
	defer rows1.Close()

	results1, err := scanRowsToMap(rows1)
	if err != nil {
		return fmt.Errorf("failed to scan T1: %w", err)
	}

	if len(results1) == 0 {
		return nil // No results
	}

	// 2. Extract keys from T1 results
	// We assume T1 joins to T2 using T1.K1 = T2.K2
	// So we need values of K1 from T1 results

	keys := make([]interface{}, 0, len(results1))
	seen := make(map[interface{}]bool)

	for _, row := range results1 {
		if val, ok := row[k1]; ok && val != nil {
			if !seen[val] {
				keys = append(keys, val)
				seen[val] = true
			}
		}
	}

	if len(keys) == 0 {
		return nil // No keys to join
	}

	// 3. Fetch T2
	// We need a new query for T2
	
	// Ensure K2 is in cols2
	if len(cols2) > 0 {
		hasK2 := false
		for _, col := range cols2 {
			if col == k2 || col == t2+"."+k2 {
				hasK2 = true
				break
			}
		}
		if !hasK2 {
			cols2 = append(cols2, t2+"."+k2)
		}
	} else {
		// If no specific columns for T2, select *
		cols2 = []string{"*"}
	}

	q2 := &Query{
		builder: &QueryBuilder{
			tableName: t2,
			columns:   cols2,
			whereArgs: []interface{}{},
			queryType: "select",
		},
		table: t2,
	}

	// Add WHERE K2 IN (...)
	placeholders := make([]string, len(keys))
	for i := range keys {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}
	
	inClause := fmt.Sprintf("%s IN (%s)", k2, strings.Join(placeholders, ", "))
	q2.Where(inClause, keys...)

	pool2, err := q2.getPool()
	if err != nil {
		return fmt.Errorf("failed to get pool for T2: %w", err)
	}

	sql2, args2, err := q2.builder.Build()
	if err != nil {
		return err
	}

	rows2, err := pool2.Pool.Query(ctx, sql2, args2...)
	if err != nil {
		return fmt.Errorf("failed to fetch T2: %w", err)
	}
	defer rows2.Close()

	results2, err := scanRowsToMap(rows2)
	if err != nil {
		return fmt.Errorf("failed to scan T2: %w", err)
	}

	// 4. Merge Results
	// Create a map of T2 results indexed by K2 (normalized to string)
	t2Map := make(map[string][]map[string]interface{})
	for _, row := range results2 {
		if val, ok := row[k2]; ok && val != nil {
			keyStr := fmt.Sprintf("%v", val)
			t2Map[keyStr] = append(t2Map[keyStr], row)
		}
	}

	// Join T1 and T2
	var joinedResults []map[string]interface{}
	
	for _, r1 := range results1 {
		val := r1[k1]
		if val == nil {
			continue
		}
		keyStr := fmt.Sprintf("%v", val)
		
		if r2List, ok := t2Map[keyStr]; ok {
			for _, r2 := range r2List {
				// Merge r1 and r2
				merged := make(map[string]interface{})
				for k, v := range r1 {
					if strings.Contains(k, ".") {
						merged[k] = v
					} else {
						merged[t1+"."+k] = v
					}
				}
				for k, v := range r2 {
					if strings.Contains(k, ".") {
						merged[k] = v
					} else {
						merged[t2+"."+k] = v
					}
				}
				joinedResults = append(joinedResults, merged)
			}
		}
	}

	if dest != nil {
		return scanMapsToDest(joinedResults, dest)
	}

	// Print results (Table format)
	fmt.Printf("\nJoined Results (%d rows):\n", len(joinedResults))
	if len(joinedResults) > 0 {
		// Print header
		// Just print a few columns for demo
		// Use keys from first row to find suitable columns to print
		var keys []string
		for k := range joinedResults[0] {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		
		// Try to find fullname/bio/total/event_type
		col1 := ""
		col2 := ""
		
		for _, k := range keys {
			if strings.Contains(k, "fullname") {
				col1 = k
			} else if strings.Contains(k, "bio") || strings.Contains(k, "total") || strings.Contains(k, "event_type") {
				col2 = k
			}
		}
		
		if col1 == "" && len(keys) > 0 { col1 = keys[0] }
		if col2 == "" && len(keys) > 1 { col2 = keys[1] }
		
		if col1 != "" && col2 != "" {
			fmt.Printf("%-30s | %-30s\n", col1, col2)
			fmt.Println(strings.Repeat("-", 65))
			
			for _, row := range joinedResults {
				v1 := fmt.Sprintf("%v", row[col1])
				v2 := fmt.Sprintf("%v", row[col2])
				if len(v1) > 28 { v1 = v1[:25] + "..." }
				if len(v2) > 28 { v2 = v2[:25] + "..." }
				fmt.Printf("%-30s | %-30s\n", v1, v2)
			}
		}
	}

	return nil
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
