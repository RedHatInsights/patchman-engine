package controllers

import (
	"app/base/core"
	"github.com/stretchr/testify/assert"
	"testing"
)

var testFilters = []string{
	"eq:abc",
	"in:a,b,c",
	"gt:13",
	"lt:12",
	"between:12,13",
}

func TestFilterParse(t *testing.T) {
	core.SetupTestEnvironment()
	operators := []string{
		"eq", "in", "gt", "lt", "between",
	}

	values := [][]string{
		{"abc"},
		{"a", "b", "c"},
		{"13"},
		{"12"},
		{"12", "13"},
	}

	for i, f := range testFilters {
		filter, err := ParseFilterValue("test", f)
		assert.Equal(t, nil, err)
		assert.Equal(t, operators[i], filter.Operator)
		assert.Equal(t, values[i], filter.Values)
	}
}

func TestFilterFiltering(t *testing.T) {
	core.SetupTestEnvironment()
	filters := Filters{}
	for _, f := range testFilters {
		filter, err := ParseFilterValue("test", f)
		assert.Equal(t, nil, err)
		filters = append(filters, filter)
	}

	filteredFilters, err := filters.FilterFilters(AttrMap{"test": "test"})
	assert.Nil(t, err)
	assert.Equal(t, filters, filteredFilters)
}

func TestFilterToSql(t *testing.T) {
	core.SetupTestEnvironment()
	queries := []string{
		"test = ? ",
		"test IN (?) ",
		"test > ? ",
		"test < ? ",
		"test BETWEEN ? AND ? ",
	}

	for i, f := range testFilters {
		filter, err := ParseFilterValue("test", f)
		assert.Equal(t, nil, err)
		query, _, err := filter.ToWhere(AttrMap{})
		assert.Equal(t, nil, err)
		assert.Equal(t, queries[i], query)
	}
}
