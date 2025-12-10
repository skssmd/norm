package engine

import (
	"reflect"
	"strings"
)

// This file is kept for backward compatibility
// The recommended approaches are:
// 1. Field pointers: Select(&user.Name, &user.Email)
// 2. String literals: Select("name", "email")

// Field creates a type-safe field selector
// Usage: Field("Name") or use field pointers/selectors directly
func Field(name string) FieldSelector {
	return FieldSelector{FieldName: name}
}

// Fields creates multiple field selectors from a model
// This uses reflection to extract all field names
func Fields(model interface{}, fieldNames ...string) []interface{} {
	result := make([]interface{}, len(fieldNames))
	for i, name := range fieldNames {
		result[i] = Field(name)
	}
	return result
}

// AllFields extracts all field names from a model struct
func AllFields(model interface{}) []interface{} {
	t := reflect.TypeOf(model)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return []interface{}{"*"}
	}

	fields := make([]interface{}, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		// Skip unexported fields
		if field.PkgPath != "" {
			continue
		}
		fields = append(fields, Field(field.Name))
	}

	return fields
}

// Cols is a helper to create field selectors from column names (snake_case)
// This is useful when you want to specify database column names directly
func Cols(names ...string) []interface{} {
	result := make([]interface{}, len(names))
	for i, name := range names {
		// Convert snake_case to PascalCase for field name
		result[i] = FieldSelector{FieldName: toPascalCase(name)}
	}
	return result
}

// toPascalCase converts snake_case to PascalCase
func toPascalCase(s string) string {
	parts := strings.Split(s, "_")
	var result strings.Builder

	for _, part := range parts {
		if len(part) > 0 {
			result.WriteString(strings.ToUpper(part[0:1]))
			if len(part) > 1 {
				result.WriteString(part[1:])
			}
		}
	}

	return result.String()
}
