package pagination

import (
	"reflect"
	"strconv"

	"github.com/getarcaneapp/arcane/backend/internal/utils/stringutils"
)

// BuildSortClause returns a safe ORDER BY clause based on sortable struct tags.
// It returns an empty string if the requested sort is not allowed.
func BuildSortClause(params QueryParams, result interface{}) string {
	sortColumn := params.Sort
	if sortColumn == "" {
		return ""
	}
	sortDirection := string(params.Order)
	if sortDirection == "" {
		sortDirection = "asc"
	}
	sortDirection = normalizeSortDirection(sortDirection)

	capitalizedSortColumn := stringutils.CapitalizeFirstLetter(sortColumn)
	sortField, sortFieldFound := reflect.TypeOf(result).Elem().Elem().FieldByName(capitalizedSortColumn)
	isSortable, _ := strconv.ParseBool(sortField.Tag.Get("sortable"))
	if !sortFieldFound || !isSortable {
		return ""
	}

	columnName := stringutils.CamelCaseToSnakeCase(sortColumn)
	return "ORDER BY " + columnName + " " + sortDirection
}

// BuildLimitOffset computes limit/offset values for SQL queries.
// If limit is -1, it returns ok=false to indicate no pagination should be applied.
func BuildLimitOffset(params QueryParams) (limit int, offset int, ok bool) {
	limit = params.Limit
	if limit == -1 {
		return 0, 0, false
	}
	if limit <= 0 {
		limit = 20
	} else if limit > 100 {
		limit = 100
	}
	offset = params.Start
	if offset < 0 {
		offset = 0
	}
	return limit, offset, true
}

func normalizeSortDirection(direction string) string {
	if direction != "asc" && direction != "desc" {
		return "asc"
	}
	return direction
}
