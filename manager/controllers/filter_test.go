package controllers

import (
	"app/base/core"
	"app/base/database"
	"testing"

	"github.com/stretchr/testify/assert"
)

var testFilters = []string{
	"eq:abc",
	"in:a,b,c",
	"gt:13",
	"gte:13",
	"lt:12",
	"lte:12",
	"between:12,13",
	"null:",
	"notnull:",
}

func dummyParser(v string) (interface{}, error) {
	return v, nil
}

func TestFilterParse(t *testing.T) {
	operators := []string{
		"eq", "in", "gt", "gte", "lt", "lte", "between", "null", "notnull",
	}

	values := [][]string{
		{"abc"},
		{"a", "b", "c"},
		{"13"},
		{"13"},
		{"12"},
		{"12"},
		{"12", "13"},
		{""},
		{""},
	}

	for i, f := range testFilters {
		filter := ParseFilterValue(ColumnFilter, f)
		assert.Equal(t, operators[i], filter.Operator)
		assert.Equal(t, values[i], filter.Values)
	}
}

func TestFilterToSql(t *testing.T) {
	queries := []string{
		"test = ? ",
		"test IN (?) ",
		"test > ? ",
		"test >= ? ",
		"test < ? ",
		"test <= ? ",
		"test BETWEEN ? AND ? ",
		"test IS NULL ",
		"test IS NOT NULL ",
	}

	for i, f := range testFilters {
		filter := ParseFilterValue(ColumnFilter, f)

		attrMap := database.AttrMap{"test": {DataQuery: "test", OrderQuery: "test", Parser: dummyParser}}
		query, _, err := filter.ToWhere("test", attrMap)
		assert.Equal(t, nil, err)
		assert.Equal(t, queries[i], query)
	}
}

func TestFilterToSqlAdvanced(t *testing.T) {
	queries := []string{
		"(NOT test) = ? ",
		"(NOT test) IN (?) ",
		"(NOT test) > ? ",
		"(NOT test) >= ? ",
		"(NOT test) < ? ",
		"(NOT test) <= ? ",
		"(NOT test) BETWEEN ? AND ? ",
		"(NOT test) IS NULL ",
		"(NOT test) IS NOT NULL ",
	}

	for i, f := range testFilters {
		filter := ParseFilterValue(ColumnFilter, f)
		attrMap := database.AttrMap{"test": {DataQuery: "(NOT test)", OrderQuery: "(NOT test)", Parser: dummyParser}}
		query, _, err := filter.ToWhere("test", attrMap)
		assert.Equal(t, nil, err)
		assert.Equal(t, queries[i], query)
	}
}

// Filter out null characters
func TestFilterInvalidValue(t *testing.T) {
	filter := ParseFilterValue(ColumnFilter, "eq:aa\u0000aa")
	attrMap, _, err := database.GetQueryAttrs(struct{ V string }{""})
	assert.NoError(t, err)
	_, value, err := filter.ToWhere("v", attrMap)
	assert.NoError(t, err)
	assert.Equal(t, []interface{}{"aaaa"}, value)
}

func TestFilterOtherAdvisoryTypes(t *testing.T) {
	core.SetupTest(t)
	// Check the list is loaded from database correctly
	assert.Equal(t, []string{"unknown", "unspecified"}, database.OtherAdvisoryTypes)
}

func TestFilterSeverityInWithNull(t *testing.T) {
	filter := ParseFilterValue(ColumnFilter, "in:2,null")
	attrMap := database.AttrMap{"severity": {DataQuery: "severity", OrderQuery: "severity", Parser: dummyParser}}
	query, args, err := filter.ToWhere("severity", attrMap)

	assert.NoError(t, err)
	assert.Equal(t, "(severity IN ? OR severity IS NULL) ", query)
	assert.Equal(t, 1, len(args))
	assert.IsType(t, []any{}, args[0])
	severityValues := args[0].([]any)
	assert.Equal(t, 1, len(severityValues))
	assert.Equal(t, "2", severityValues[0])
}

func TestFilterSeverityNotInWithNull(t *testing.T) {
	filter := ParseFilterValue(ColumnFilter, "notin:2,3,null")
	attrMap := database.AttrMap{"severity": {DataQuery: "severity", OrderQuery: "severity", Parser: dummyParser}}
	query, args, err := filter.ToWhere("severity", attrMap)

	assert.NoError(t, err)
	assert.Equal(t, "(severity NOT IN ? AND severity IS NOT NULL) ", query)
	assert.Equal(t, 1, len(args))
	assert.IsType(t, []any{}, args[0])
	severityValues := args[0].([]any)
	assert.Equal(t, 2, len(severityValues))
	assert.Equal(t, "2", severityValues[0])
	assert.Equal(t, "3", severityValues[1])
}
