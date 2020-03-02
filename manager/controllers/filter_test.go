package controllers

import (
	"app/base/database"
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
		filter, err := ParseFilterValue(f)
		assert.Equal(t, nil, err)
		assert.Equal(t, operators[i], filter.Operator)
		assert.Equal(t, values[i], filter.Values)
	}
}

func TestFilterToSql(t *testing.T) {
	queries := []string{
		"test = ? ",
		"test IN (?) ",
		"test > ? ",
		"test < ? ",
		"test BETWEEN ? AND ? ",
	}

	for i, f := range testFilters {
		filter, err := ParseFilterValue(f)
		assert.Equal(t, nil, err)
		query, _, err := filter.ToWhere("test", database.AttrMap{"test": "test"})
		assert.Equal(t, nil, err)
		assert.Equal(t, queries[i], query)
	}
}

func TestFilterToSqlAdvanced(t *testing.T) {
	queries := []string{
		"(NOT test) = ? ",
		"(NOT test) IN (?) ",
		"(NOT test) > ? ",
		"(NOT test) < ? ",
		"(NOT test) BETWEEN ? AND ? ",
	}

	for i, f := range testFilters {
		filter, err := ParseFilterValue(f)
		assert.Equal(t, nil, err)
		query, _, err := filter.ToWhere("test", database.AttrMap{"test": "(NOT test)"})
		assert.Equal(t, nil, err)
		assert.Equal(t, queries[i], query)
	}
}
