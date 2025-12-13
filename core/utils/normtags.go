package utils

import (
	"fmt"
	"reflect"
	"strings"
)

// parseNormTags parses norm struct tags
func ParseNormTags(tag string) map[string]interface{} {
	tags := make(map[string]interface{})
	if tag == "" {
		return tags
	}

	parts := strings.Split(tag, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if strings.Contains(part, ":") {
			kv := strings.SplitN(part, ":", 2)
			tags[kv[0]] = kv[1]
		} else {
			tags[part] = true
		}
	}

	return tags
}


func GetPostgresType(field reflect.StructField) string {
	tags := ParseNormTags(field.Tag.Get("norm"))

	// Explicit type from tag
	if sqlType, ok := tags["type"]; ok {
		return sqlType.(string)
	}

	fieldType := field.Type
	if fieldType.Kind() == reflect.Ptr {
		fieldType = fieldType.Elem()
	}

	switch fieldType.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		return "INTEGER"
	case reflect.Int64:
		return "BIGINT"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "BIGINT"
	case reflect.Float32:
		return "REAL"
	case reflect.Float64:
		return "DOUBLE PRECISION"
	case reflect.Bool:
		return "BOOLEAN"
	case reflect.String:
		if _, ok := tags["text"]; ok {
			return "TEXT"
		}
		if maxLen, ok := tags["max"]; ok {
			return fmt.Sprintf("VARCHAR(%s)", maxLen.(string))
		}
		return "VARCHAR(255)"
	case reflect.Struct:
		if fieldType.String() == "time.Time" {
			return "TIMESTAMP"
		}
		return "JSONB"
	case reflect.Slice:
		if fieldType.Elem().Kind() == reflect.Uint8 {
			return "BYTEA"
		}
		switch fieldType.Elem().Kind() {
		case reflect.String:
			return "TEXT[]"
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
			return "INTEGER[]"
		case reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return "BIGINT[]"
		case reflect.Float32:
			return "REAL[]"
		case reflect.Float64:
			return "DOUBLE PRECISION[]"
		case reflect.Bool:
			return "BOOLEAN[]"
		default:
			return "JSONB"
		}
	case reflect.Map:
		return "JSONB"
	default:
		return "TEXT"
	}
}