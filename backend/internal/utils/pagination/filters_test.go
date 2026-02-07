package pagination

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSearchOrderAndPaginate_IgnoresEmptyFilterValues(t *testing.T) {
	type item struct {
		Name   string
		Status string
	}

	items := []item{
		{Name: "a", Status: "running"},
		{Name: "b", Status: "stopped"},
	}

	result := SearchOrderAndPaginate(items, QueryParams{
		Filters: map[string]string{
			"status": "",
		},
	}, Config[item]{
		FilterAccessors: []FilterAccessor[item]{
			{
				Key: "status",
				Fn: func(i item, filterValue string) bool {
					return i.Status == filterValue
				},
			},
		},
	})

	assert.Len(t, result.Items, 2)
	assert.Equal(t, int64(2), result.TotalCount)
	assert.Equal(t, int64(2), result.TotalAvailable)
}

func TestSearchOrderAndPaginate_IgnoresEmptyFilterTokens(t *testing.T) {
	type item struct {
		Name   string
		Status string
	}

	items := []item{
		{Name: "a", Status: "running"},
		{Name: "b", Status: "stopped"},
	}

	result := SearchOrderAndPaginate(items, QueryParams{
		Filters: map[string]string{
			"status": "running, ,",
		},
	}, Config[item]{
		FilterAccessors: []FilterAccessor[item]{
			{
				Key: "status",
				Fn: func(i item, filterValue string) bool {
					return i.Status == filterValue
				},
			},
		},
	})

	assert.Len(t, result.Items, 1)
	assert.Equal(t, "running", result.Items[0].Status)
	assert.Equal(t, int64(1), result.TotalCount)
	assert.Equal(t, int64(2), result.TotalAvailable)
}
