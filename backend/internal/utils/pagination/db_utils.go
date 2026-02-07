package pagination

import (
	"fmt"
	"strings"
)

// PlaceholderFunc returns a placeholder string for the given argument index (1-based).
type PlaceholderFunc func(index int) string

// QuestionMarkPlaceholder returns a "?" placeholder for any index.
func QuestionMarkPlaceholder(_ int) string {
	return "?"
}

// DollarPlaceholder returns a "$<index>" placeholder (for Postgres-style queries).
func DollarPlaceholder(index int) string {
	return fmt.Sprintf("$%d", index)
}

// BuildFilterClause builds a WHERE clause fragment and args for string values.
// It uses question mark placeholders by default.
func BuildFilterClause(column string, value string) (string, []any) {
	return BuildFilterClauseWithPlaceholders(column, value, QuestionMarkPlaceholder)
}

// BuildFilterClauseWithPlaceholders builds a WHERE clause fragment with a custom placeholder style.
func BuildFilterClauseWithPlaceholders(column string, value string, placeholder PlaceholderFunc) (string, []any) {
	if value == "" {
		return "", nil
	}

	if strings.Contains(value, ",") {
		values := strings.Split(value, ",")
		args := make([]any, 0, len(values))
		holders := make([]string, 0, len(values))
		for i, v := range values {
			trimmed := strings.TrimSpace(v)
			if trimmed == "" {
				continue
			}
			args = append(args, trimmed)
			holders = append(holders, placeholder(i+1))
		}
		if len(args) == 0 {
			return "", nil
		}
		return fmt.Sprintf("%s IN (%s)", column, strings.Join(holders, ", ")), args
	}

	return fmt.Sprintf("%s = %s", column, placeholder(1)), []any{value}
}

// BuildBooleanFilterClause builds a WHERE clause fragment and args for boolean values.
// It uses question mark placeholders by default.
func BuildBooleanFilterClause(column string, value string) (string, []any) {
	return BuildBooleanFilterClauseWithPlaceholders(column, value, QuestionMarkPlaceholder)
}

// BuildBooleanFilterClauseWithPlaceholders builds a WHERE clause fragment with a custom placeholder style.
func BuildBooleanFilterClauseWithPlaceholders(column string, value string, placeholder PlaceholderFunc) (string, []any) {
	if value == "" {
		return "", nil
	}

	parts := strings.Split(value, ",")
	var boolValues []bool

	for _, part := range parts {
		switch strings.TrimSpace(part) {
		case "true", "1":
			boolValues = append(boolValues, true)
		case "false", "0":
			boolValues = append(boolValues, false)
		}
	}

	if len(boolValues) == 0 {
		return "", nil
	}
	if len(boolValues) == 1 {
		return fmt.Sprintf("%s = %s", column, placeholder(1)), []any{boolValues[0]}
	}

	holders := make([]string, 0, len(boolValues))
	args := make([]any, 0, len(boolValues))
	for i, v := range boolValues {
		args = append(args, v)
		holders = append(holders, placeholder(i+1))
	}
	return fmt.Sprintf("%s IN (%s)", column, strings.Join(holders, ", ")), args
}
