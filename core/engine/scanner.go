package engine

import (
	"errors"
	"reflect"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/skssmd/norm/core/utils"
)

// scanRowsToDest scans pgx.Rows into a destination (pointer to slice of structs or pointer to struct)
func scanRowsToDest(rows pgx.Rows, dest interface{}) error {
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr || destValue.IsNil() {
		return errors.New("dest must be a non-nil pointer")
	}

	destElem := destValue.Elem()
	
	// Case 1: Slice of structs (e.g. *[]User)
	if destElem.Kind() == reflect.Slice {
		sliceType := destElem.Type()
		elemType := sliceType.Elem()
		isPtr := false
		if elemType.Kind() == reflect.Ptr {
			elemType = elemType.Elem()
			isPtr = true
		}

		if elemType.Kind() != reflect.Struct {
			return errors.New("dest slice element must be a struct or pointer to struct")
		}

		// Map columns to fields
		fields := rows.FieldDescriptions()
		columnMap := make([]int, len(fields)) // index in row -> index in struct field
		
		for i, fd := range fields {
			colName := string(fd.Name)
			columnMap[i] = -1
			
			// Find matching field
			for j := 0; j < elemType.NumField(); j++ {
				field := elemType.Field(j)
				dbName := utils.ResolveColumnName(field)
				if dbName == colName {
					columnMap[i] = j
					break
				}
			}
		}

		// Iterate rows
		for rows.Next() {
			newElem := reflect.New(elemType).Elem()
			scanArgs := make([]interface{}, len(fields))
			
			for i := range fields {
				fieldIdx := columnMap[i]
				if fieldIdx != -1 {
					scanArgs[i] = newElem.Field(fieldIdx).Addr().Interface()
				} else {
					var ignored interface{}
					scanArgs[i] = &ignored
				}
			}

			if err := rows.Scan(scanArgs...); err != nil {
				return err
			}

			if isPtr {
				destElem.Set(reflect.Append(destElem, newElem.Addr()))
			} else {
				destElem.Set(reflect.Append(destElem, newElem))
			}
		}
		
		return rows.Err()
	}

	// Case 2: Single struct (e.g. *User)
	if destElem.Kind() == reflect.Struct {
		elemType := destElem.Type()
		
		if !rows.Next() {
			if err := rows.Err(); err != nil {
				return err
			}
			return errors.New("no rows in result set")
		}

		fields := rows.FieldDescriptions()
		scanArgs := make([]interface{}, len(fields))

		for i, fd := range fields {
			colName := string(fd.Name)
			found := false
			
			for j := 0; j < elemType.NumField(); j++ {
				field := elemType.Field(j)
				dbName := utils.ResolveColumnName(field)
				if dbName == colName {
					scanArgs[i] = destElem.Field(j).Addr().Interface()
					found = true
					break
				}
			}
			
			if !found {
				var ignored interface{}
				scanArgs[i] = &ignored
			}
		}

		return rows.Scan(scanArgs...)
	}

	return errors.New("dest must be a pointer to a struct or a slice of structs")
}

// scanMapsToDest scans []map[string]interface{} into dest (for App-Side Joins)
func scanMapsToDest(results []map[string]interface{}, dest interface{}) error {
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr || destValue.IsNil() {
		return errors.New("dest must be a non-nil pointer")
	}

	destElem := destValue.Elem()

	if destElem.Kind() != reflect.Slice {
		return errors.New("dest must be a pointer to a slice for join results")
	}

	sliceType := destElem.Type()
	elemType := sliceType.Elem()
	isPtr := false
	if elemType.Kind() == reflect.Ptr {
		elemType = elemType.Elem()
		isPtr = true
	}

	if elemType.Kind() != reflect.Struct {
		return errors.New("dest slice element must be a struct or pointer to struct")
	}

	for _, row := range results {
		newElem := reflect.New(elemType).Elem()

		for j := 0; j < elemType.NumField(); j++ {
			field := elemType.Field(j)
			dbName := utils.ResolveColumnName(field)

			// Try to find value in map
			// 1. Exact match
			if val, ok := row[dbName]; ok {
				setField(newElem.Field(j), val)
				continue
			}

			// 2. Tablename prefix match (e.g. "users.fullname" matches "fullname")
			for k, v := range row {
				if strings.HasSuffix(k, "."+dbName) {
					setField(newElem.Field(j), v)
					break
				}
			}
		}

		if isPtr {
			destElem.Set(reflect.Append(destElem, newElem.Addr()))
		} else {
			destElem.Set(reflect.Append(destElem, newElem))
		}
	}

	return nil
}

func setField(field reflect.Value, value interface{}) {
	if value == nil {
		return
	}
	
	val := reflect.ValueOf(value)
	
	// Handle pointers in struct field
	if field.Kind() == reflect.Ptr {
		// Create new pointer
		newPtr := reflect.New(field.Type().Elem())
		// Recursively set the value to the element
		setField(newPtr.Elem(), value)
		field.Set(newPtr)
		return
	}

	// Simple type conversion if needed (e.g. int64 to int)
	if field.Type() != val.Type() {
		if val.Type().ConvertibleTo(field.Type()) {
			field.Set(val.Convert(field.Type()))
		} else {
			// Fallback: fmt.Scan? Or just ignore for now to avoid panic
			// For basic types, ConvertibleTo handles int/float/string
		}
	} else {
		field.Set(val)
	}
}
