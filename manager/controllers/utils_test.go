package controllers

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type filterTestCase struct {
	filter string
	val    string
	out    string
}

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

func TestNestedQueryInvalidKey(t *testing.T) {
	timeout := time.After(5 * time.Second)
	done := make(chan bool)

	go func() {
		q := map[string][]string{
			"filter[abc][efg][eq]": {"a"},
		}
		res := nestedQueryImpl(q, "filte")
		assert.Equal(t, res, QueryMap{})
		done <- true
	}()

	select {
	case <-timeout:
		t.Fatal("Timeout exceeded - probably infinite loop in nested query")
	case <-done:
	}
}

func TestBuildQuery(t *testing.T) {
	cases := []filterTestCase{
		{"sap_system", "True", "h.system_profile::jsonb ? 'sap_version'"},
		{"sap_system", "False", "not h.system_profile::jsonb ? 'sap_version'"},
		// todo: add tests for other cases
	}
	for _, tc := range cases {
		res := buildQuery(tc.filter, tc.val)
		assert.Equal(t, tc.out, res)
	}
}
