package controllers

import (
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func TestNestedQueryParse(t *testing.T) {
	q1 := map[string][]string{
		"filter[abc][efg][eq]": {"a"},
		"filter[a][]":          {"b", "c"},
		// Check that we stop after we encountered invalid filter syntax
		"filter[]]]]]": {},
		"filter[[[[[":  {},
		"filter":       {},
	}
	res := nestedQueryImpl(q1, "filter")
	res.Visit(func(keys []string, val string) {
		if reflect.DeepEqual([]string{"abc", "efg", "eq"}, keys) {
			assert.Equal(t, "a", val)
		}
		// We need to be able to parse multi-value elems
		if reflect.DeepEqual([]string{"a"}, keys) {
			assert.Contains(t, []string{"b", "c"}, val)
		}
	})
}
