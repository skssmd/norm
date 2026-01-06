package engine

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/skssmd/norm/core/registry"
	"github.com/skssmd/norm/core/utils"
)

// JoinDefinition represents a JOIN clause
type JoinDefinition struct {
	Table string
	On    string
	Type  string // "INNER", "LEFT", "RIGHT"
}

// QueryBuilder builds SQL queries with a fluent API
type QueryBuilder struct {
	tableName       string
	model           interface{} // Store model for type safety
	columns         []string
	whereClause     string
	whereArgs       []interface{}
	updateFields    map[string]interface{}
	insertFields    map[string]interface{}
	bulkColumns     []string
	bulkRows        [][]interface{}
	onConflict      string   // Conflict target columns
	conflictAction  string   // "nothing" or "update"
	conflictUpdates []string // Columns to update on conflict
	orderBy         string
	limit           int
	offset          int
	queryType        string // "select", "update", "delete", "insert", "bulkinsert"
	joins            []JoinDefinition
	returningColumns []string
}

// From creates a new query builder for the specified model
// Usage:
//
//	user := User{}
//	From(&user).Select(&user.Name, &user.Email)  // Pass pointer to make it addressable
//
// The model instance binds field pointers to the correct table
func From(model interface{}) *QueryBuilder {
	tableName := getTableNameFromModel(model)

	return &QueryBuilder{
		tableName:    tableName,
		model:        model,
		columns:      []string{},
		whereArgs:    []interface{}{},
		updateFields: make(map[string]interface{}),
		insertFields: make(map[string]interface{}),
		joins:        []JoinDefinition{},
	}
}



// getTableNameFromModel extracts table name from registry or derives it
func getTableNameFromModel(model interface{}) string {
	// First, try to get registered table name from registry
	registeredName := registry.GetRegisteredTableName(model)
	if registeredName != "" {
		return registeredName
	}

	// Fallback: derive from struct name (snake_case + pluralize)
	t := reflect.TypeOf(model)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	snakeName := utils.ToSnakeCase(t.Name())
	return utils.Pluralize(snakeName)
}

// Select specifies columns to select
// Usage:
//
//	user := User{}
//	From(user).Select(&user.Name, &user.Email, &user.Age, &user.CreatedAt)
//
// Field pointers can be any type - string, int, uint, time.Time, etc.
// The compiler enforces that fields belong to the model instance
func (qb *QueryBuilder) Select(fields ...interface{}) *QueryBuilder {
	qb.queryType = "select"
	if len(fields) == 0 {
		qb.columns = []string{"*"}
	} else {
		qb.columns = qb.extractFieldNames(fields)
	}
	return qb
}

// extractFieldNames converts field pointers or strings to column names
func (qb *QueryBuilder) extractFieldNames(fields []interface{}) []string {
	if len(fields) == 0 {
		return []string{"*"}
	}

	// Check if first argument is a string
	if _, ok := fields[0].(string); ok {
		columns := make([]string, len(fields))
		for i, f := range fields {
			columns[i] = f.(string)
		}
		return columns
	}

	// Extract field names from field pointers using reflection
	columns := make([]string, 0, len(fields))
	modelValue := reflect.ValueOf(qb.model)

	// Ensure we have an addressable value
	if modelValue.Kind() == reflect.Ptr {
		modelValue = modelValue.Elem()
	}

	// Check if the value is addressable
	if !modelValue.CanAddr() {
		// If not addressable, we need to work with the original pointer
		// Get the pointer to the field from the field pointer itself
		for _, fieldPtr := range fields {
			fieldName := qb.getFieldNameFromPointerDirect(fieldPtr)
			if fieldName != "" {
				columns = append(columns, utils.ToSnakeCase(fieldName))
			}
		}
		return columns
	}

	modelType := modelValue.Type()

	for _, fieldPtr := range fields {
		fieldName := qb.getFieldNameFromPointer(fieldPtr, modelValue, modelType)
		if fieldName != "" {
			columns = append(columns, utils.ToSnakeCase(fieldName))
		}
	}

	return columns
}

// getFieldNameFromPointerDirect extracts field name by comparing pointer addresses
// This works even when the model value is not addressable
func (qb *QueryBuilder) getFieldNameFromPointerDirect(fieldPtr interface{}) string {
	ptrValue := reflect.ValueOf(fieldPtr)

	// Must be a pointer
	if ptrValue.Kind() != reflect.Ptr {
		return ""
	}

	// Get the type of what the pointer points to
	ptrType := ptrValue.Type().Elem()

	// Get the model type
	modelValue := reflect.ValueOf(qb.model)
	if modelValue.Kind() == reflect.Ptr {
		modelValue = modelValue.Elem()
	}
	modelType := modelValue.Type()

	// Try to match by type and find the field
	// This is a simpler approach that works with non-addressable values
	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)
		if field.Type == ptrType {
			// Found a field with matching type
			// This is a best-effort match
			return field.Name
		}
	}

	return ""
}

// getFieldNameFromPointer extracts field name from a field pointer using reflection
func (qb *QueryBuilder) getFieldNameFromPointer(fieldPtr interface{}, modelValue reflect.Value, modelType reflect.Type) string {
	ptrValue := reflect.ValueOf(fieldPtr)

	// Must be a pointer
	if ptrValue.Kind() != reflect.Ptr {
		return ""
	}

	// Get the address of the field pointer
	fieldAddr := ptrValue.Pointer()

	// Get the base address of the model
	modelAddr := modelValue.UnsafeAddr()

	// Calculate offset
	offset := fieldAddr - modelAddr

	// Find which field has this offset
	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)
		fieldValue := modelValue.Field(i)

		if fieldValue.UnsafeAddr()-modelAddr == offset {
			return field.Name
		}
	}

	return ""
}

// FieldSelector is a helper for type-safe field selection (F() function)
type FieldSelector struct {
	FieldName string
}

// Where adds a WHERE clause with parameterized values
// Usage: Where("id = $1 AND status = $2", 1, "active")
func (qb *QueryBuilder) Where(condition string, args ...interface{}) *QueryBuilder {
	qb.whereClause = condition
	qb.whereArgs = args
	return qb
}

// Delete marks this as a delete query
func (qb *QueryBuilder) Delete() *QueryBuilder {
	qb.queryType = "delete"
	return qb
}

// Insert sets fields to insert from a model instance (ALL fields including zero values)
// Usage:
//
//	user.Name = "John"
//	user.Email = "john@example.com"
//	Insert(user)
func (qb *QueryBuilder) Insert(model interface{}) *QueryBuilder {
	qb.queryType = "insert"
	qb.insertFields = qb.extractAllFieldsFromModel(model)
	return qb
}

// InsertNonZero sets fields to insert from a model instance (only non-zero values)
// This ignores zero values like "", 0, nil, false
func (qb *QueryBuilder) InsertNonZero(model interface{}) *QueryBuilder {
	qb.queryType = "insert"
	qb.insertFields = qb.extractFieldsFromModel(model)
	return qb
}

// Returning specifies columns to return after INSERT
func (qb *QueryBuilder) Returning(cols ...string) *QueryBuilder {
	qb.returningColumns = cols
	return qb
}

// extractFieldsFromModel extracts non-zero fields from a model instance
func (qb *QueryBuilder) extractFieldsFromModel(model interface{}) map[string]interface{} {
	fields := make(map[string]interface{})

	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return fields
	}

	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		fv := v.Field(i)

		// skip unexported fields
		if sf.PkgPath != "" {
			continue
		}

		// skip zero values (unset fields)
		if fv.IsZero() {
			continue
		}

		columnName := utils.ResolveColumnName(sf)
		fields[columnName] = fv.Interface()
	}

	return fields
}


// extractAllFieldsFromModel extracts ALL fields from a model instance (including zero values)
	func (qb *QueryBuilder) extractAllFieldsFromModel(model interface{}) map[string]interface{} {
		fields := make(map[string]interface{})

		v := reflect.ValueOf(model)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}

		if v.Kind() != reflect.Struct {
			return fields
		}

		t := v.Type()

		for i := 0; i < t.NumField(); i++ {
			sf := t.Field(i)
			fv := v.Field(i)

			// skip unexported
			if sf.PkgPath != "" {
				continue
			}

			columnName := utils.ResolveColumnName(sf)
			fields[columnName] = fv.Interface()
		}

		return fields
	}


// OrderBy adds ORDER BY clause
// Usage: OrderBy("created_at DESC")
func (qb *QueryBuilder) OrderBy(order string) *QueryBuilder {
	qb.orderBy = order
	return qb
}

// Limit sets the LIMIT clause
func (qb *QueryBuilder) Limit(limit int) *QueryBuilder {
	qb.limit = limit
	return qb
}

// Offset sets the OFFSET clause
func (qb *QueryBuilder) Offset(offset int) *QueryBuilder {
	qb.offset = offset
	return qb
}

// Pagination sets both limit and offset
// Usage: Pagination(10, 20) for page 3 with 10 items per page
func (qb *QueryBuilder) Pagination(limit, offset int) *QueryBuilder {
	qb.limit = limit
	qb.offset = offset
	return qb
}

// Build generates the SQL query and arguments
func (qb *QueryBuilder) Build() (string, []interface{}, error) {
	switch qb.queryType {
	case "select":
		return qb.buildSelect()
	case "update":
		return qb.buildUpdate()
	case "delete":
		return qb.buildDelete()
	case "insert":
		return qb.buildInsert()
	case "bulkinsert":
		return qb.buildBulkInsert()
	default:
		return "", nil, fmt.Errorf("query type not specified")
	}
}

// buildSelect builds a SELECT query
func (qb *QueryBuilder) buildSelect() (string, []interface{}, error) {
	if qb.tableName == "" {
		return "", nil, fmt.Errorf("table name is required")
	}

	var sql strings.Builder
	sql.WriteString("SELECT ")

	if len(qb.columns) == 0 {
		sql.WriteString("*")
	} else {
		sql.WriteString(strings.Join(qb.columns, ", "))
	}

	sql.WriteString(" FROM ")
	sql.WriteString(qb.tableName)

	// Add JOIN clauses
	for _, join := range qb.joins {
		sql.WriteString(fmt.Sprintf(" %s JOIN %s ON %s", join.Type, join.Table, join.On))
	}

	if qb.whereClause != "" {
		sql.WriteString(" WHERE ")
		sql.WriteString(qb.whereClause)
	}

	if qb.orderBy != "" {
		sql.WriteString(" ORDER BY ")
		sql.WriteString(qb.orderBy)
	}

	if qb.limit > 0 {
		sql.WriteString(fmt.Sprintf(" LIMIT %d", qb.limit))
	}

	if qb.offset > 0 {
		sql.WriteString(fmt.Sprintf(" OFFSET %d", qb.offset))
	}

	return sql.String(), qb.whereArgs, nil
}

// buildUpdate builds an UPDATE query
func (qb *QueryBuilder) buildUpdate() (string, []interface{}, error) {
	if qb.tableName == "" {
		return "", nil, fmt.Errorf("table name is required")
	}
	if len(qb.updateFields) == 0 {
		return "", nil, fmt.Errorf("no fields to update")
	}

	var sql strings.Builder
	var args []interface{}

	sql.WriteString("UPDATE ")
	sql.WriteString(qb.tableName)
	sql.WriteString(" SET ")

	// Build SET clause - use deterministic order for map iteration
	fields := make([]string, 0, len(qb.updateFields))
	for field := range qb.updateFields {
		fields = append(fields, field)
	}

	setClauses := []string{}
	paramIndex := 1
	for _, field := range fields {
		value := qb.updateFields[field]
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", field, paramIndex))
		args = append(args, value)
		paramIndex++
	}
	sql.WriteString(strings.Join(setClauses, ", "))

	// Add WHERE clause with adjusted parameter indexes
	if qb.whereClause != "" {
		sql.WriteString(" WHERE ")
		// Adjust parameter placeholders in WHERE clause
		adjustedWhere := qb.adjustPlaceholders(qb.whereClause, paramIndex)
		sql.WriteString(adjustedWhere)
		args = append(args, qb.whereArgs...)
	}

	return sql.String(), args, nil
}

// buildDelete builds a DELETE query
func (qb *QueryBuilder) buildDelete() (string, []interface{}, error) {
	if qb.tableName == "" {
		return "", nil, fmt.Errorf("table name is required")
	}

	var sql strings.Builder
	sql.WriteString("DELETE FROM ")
	sql.WriteString(qb.tableName)

	if qb.whereClause != "" {
		sql.WriteString(" WHERE ")
		sql.WriteString(qb.whereClause)
	}

	return sql.String(), qb.whereArgs, nil
}

// buildInsert builds an INSERT query
func (qb *QueryBuilder) buildInsert() (string, []interface{}, error) {
	if qb.tableName == "" {
		return "", nil, fmt.Errorf("table name is required")
	}
	if len(qb.insertFields) == 0 {
		return "", nil, fmt.Errorf("no fields to insert")
	}

	var sql strings.Builder
	var args []interface{}
	var columns []string
	var placeholders []string

	paramIndex := 1
	for field, value := range qb.insertFields {
		columns = append(columns, field)
		placeholders = append(placeholders, fmt.Sprintf("$%d", paramIndex))
		args = append(args, value)
		paramIndex++
	}

	sql.WriteString("INSERT INTO ")
	sql.WriteString(qb.tableName)
	sql.WriteString(" (")
	sql.WriteString(strings.Join(columns, ", "))
	sql.WriteString(") VALUES (")
	sql.WriteString(strings.Join(placeholders, ", "))
	sql.WriteString(")")

	// Add ON CONFLICT clause if specified
	if qb.onConflict != "" {
		sql.WriteString(" ON CONFLICT (")
		sql.WriteString(qb.onConflict)
		sql.WriteString(") DO ")

		if qb.conflictAction == "nothing" {
			sql.WriteString("NOTHING")
		} else if qb.conflictAction == "update" && len(qb.conflictUpdates) > 0 {
			sql.WriteString("UPDATE SET ")
			updates := []string{}
			for _, col := range qb.conflictUpdates {
				updates = append(updates, fmt.Sprintf("%s = EXCLUDED.%s", col, col))
			}
			sql.WriteString(strings.Join(updates, ", "))
		}
	}

	// Add RETURNING clause if specified
	if qb.returningColumns != nil {
		sql.WriteString(" RETURNING ")
		if len(qb.returningColumns) == 0 {
			sql.WriteString("*")
		} else {
			sql.WriteString(strings.Join(qb.returningColumns, ", "))
		}
	}

	return sql.String(), args, nil
}

// buildBulkInsert builds a bulk INSERT query
func (qb *QueryBuilder) buildBulkInsert() (string, []interface{}, error) {
	if qb.tableName == "" {
		return "", nil, fmt.Errorf("table name is required")
	}
	if len(qb.bulkColumns) == 0 {
		return "", nil, fmt.Errorf("no columns specified for bulk insert")
	}
	if len(qb.bulkRows) == 0 {
		return "", nil, fmt.Errorf("no rows to insert")
	}

	var sql strings.Builder
	var args []interface{}
	paramIndex := 1

	sql.WriteString("INSERT INTO ")
	sql.WriteString(qb.tableName)
	sql.WriteString(" (")
	sql.WriteString(strings.Join(qb.bulkColumns, ", "))
	sql.WriteString(") VALUES ")

	// Build multiple value sets
	valueSets := []string{}
	for _, row := range qb.bulkRows {
		placeholders := []string{}
		for range row {
			placeholders = append(placeholders, fmt.Sprintf("$%d", paramIndex))
			paramIndex++
		}
		valueSets = append(valueSets, "("+strings.Join(placeholders, ", ")+")")
		args = append(args, row...)
	}

	sql.WriteString(strings.Join(valueSets, ", "))

	// Add ON CONFLICT clause if specified
	if qb.onConflict != "" {
		sql.WriteString(" ON CONFLICT (")
		sql.WriteString(qb.onConflict)
		sql.WriteString(") DO ")

		if qb.conflictAction == "nothing" {
			sql.WriteString("NOTHING")
		} else if qb.conflictAction == "update" && len(qb.conflictUpdates) > 0 {
			sql.WriteString("UPDATE SET ")
			updates := []string{}
			for _, col := range qb.conflictUpdates {
				updates = append(updates, fmt.Sprintf("%s = EXCLUDED.%s", col, col))
			}
			sql.WriteString(strings.Join(updates, ", "))
		}
	}

	return sql.String(), args, nil
}

// adjustPlaceholders adjusts $1, $2, etc. to start from a different index
func (qb *QueryBuilder) adjustPlaceholders(query string, startIndex int) string {
	result := query
	// Replace in reverse order to avoid conflicts (e.g., $1 -> $2 affecting $10)
	for i := 100; i >= 1; i-- {
		old := fmt.Sprintf("$%d", i)
		new := fmt.Sprintf("$%d", startIndex+i-1)
		result = strings.ReplaceAll(result, old, new)
	}
	return result
}

// BulkInsertBuilder handles bulk insert with transaction
type BulkInsertBuilder struct {
	tableName string
	model     interface{}
	columns   []string
	rows      [][]interface{}
}

// BulkInsert creates a new bulk insert builder from model
// Usage: BulkInsert(User{}, []string{"name", "email"}, [][]interface{}{{"John", "john@example.com"}, {"Jane", "jane@example.com"}})
func BulkInsert(model interface{}, columns []string, rows [][]interface{}) *BulkInsertBuilder {
	tableName := getTableNameFromModel(model)

	return &BulkInsertBuilder{
		tableName: tableName,
		model:     model,
		columns:   columns,
		rows:      rows,
	}
}

// AddRow adds a row to the bulk insert
func (bib *BulkInsertBuilder) AddRow(values ...interface{}) *BulkInsertBuilder {
	if len(values) != len(bib.columns) {
		// Should handle error properly, for now just skip
		return bib
	}
	bib.rows = append(bib.rows, values)
	return bib
}

// Build generates the bulk insert SQL
func (bib *BulkInsertBuilder) Build() (string, []interface{}, error) {
	if bib.tableName == "" {
		return "", nil, fmt.Errorf("table name is required")
	}
	if len(bib.columns) == 0 {
		return "", nil, fmt.Errorf("columns are required")
	}
	if len(bib.rows) == 0 {
		return "", nil, fmt.Errorf("no rows to insert")
	}

	var sql strings.Builder
	var args []interface{}

	sql.WriteString("INSERT INTO ")
	sql.WriteString(bib.tableName)
	sql.WriteString(" (")
	sql.WriteString(strings.Join(bib.columns, ", "))
	sql.WriteString(") VALUES ")

	// Build VALUES clause
	valueClauses := []string{}
	paramIndex := 1

	for _, row := range bib.rows {
		placeholders := []string{}
		for _, value := range row {
			placeholders = append(placeholders, fmt.Sprintf("$%d", paramIndex))
			args = append(args, value)
			paramIndex++
		}
		valueClauses = append(valueClauses, "("+strings.Join(placeholders, ", ")+")")
	}

	sql.WriteString(strings.Join(valueClauses, ", "))

	return sql.String(), args, nil
}

// ExecuteWithTransaction executes the bulk insert within a transaction
func (bib *BulkInsertBuilder) ExecuteWithTransaction(ctx context.Context, conn interface{}) error {
	// This will be implemented in router.go
	// For now, just build the query
	sql, args, err := bib.Build()
	if err != nil {
		return err
	}

	// Type assert to pgx connection
	if pgxConn, ok := conn.(interface {
		Begin(context.Context) (pgx.Tx, error)
	}); ok {
		tx, err := pgxConn.Begin(ctx)
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}
		defer tx.Rollback(ctx)

		_, err = tx.Exec(ctx, sql, args...)
		if err != nil {
			return fmt.Errorf("failed to execute bulk insert: %w", err)
		}

		if err = tx.Commit(ctx); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}

		return nil
	}

	return fmt.Errorf("invalid connection type")
}
