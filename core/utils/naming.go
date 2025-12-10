package utils

import "strings"

// ToSnakeCase converts CamelCase to snake_case
// Handles consecutive uppercase letters correctly (ID -> id, not i_d)
func ToSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			// Check if previous character was not uppercase (avoid I_D from ID)
			if i > 0 && s[i-1] >= 'a' && s[i-1] <= 'z' {
				result.WriteRune('_')
			}
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// Pluralize adds 's' or 'es' to make plural
func Pluralize(s string) string {
	if strings.HasSuffix(s, "s") || strings.HasSuffix(s, "x") ||
		strings.HasSuffix(s, "z") || strings.HasSuffix(s, "ch") ||
		strings.HasSuffix(s, "sh") {
		return s + "es"
	}
	return s + "s"
}
