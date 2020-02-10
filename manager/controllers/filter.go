package controllers

import (
	"fmt"
	"golang.org/x/crypto/openpgp/errors"
	"strings"
)

type FilterData struct {
	Operator string   `json:"op"`
	Values   []string `json:"values"`
}
type Filter struct {
	FieldName string `json:"field"`
	FilterData
}

type Filters []Filter

// Remove filters with field names, which are not in the arguments
func (t Filters) FilterFilters(fields AttrMap) (Filters, error) {
	res := Filters{}
	err := error(nil)

	for _, v := range t {
		if _, has := fields[v.FieldName]; has {
			res = append(res, v)
		}
	}
	return res, err
}

func (t *Filters) ToQueryParams() string {
	parts := make([]string, len(*t))
	for i, v := range *t {
		values := strings.Join(v.Values, ",")
		parts[i] = fmt.Sprintf("filter[%v]=%v:%v", v.FieldName, v.Operator, values)
	}
	return strings.Join(parts, "&")
}

func (t *Filters) ToMetaMap() map[string]FilterData {
	res := make(map[string]FilterData)
	for _, v := range *t {
		res[v.FieldName] = v.FilterData
	}
	return res
}

// Convert a single filter to where clauses
func (t *Filter) ToWhere(attributes AttrMap) (string, []interface{}, error) {
	// Gorm deals with interface{} but for ease of use we only use strings
	var values = make([]interface{}, len(t.Values))
	for i, v := range t.Values {
		values[i] = v
	}
	// We need to look up expression used to create the attribute, because FROM clause can't contain
	// column aliases
	switch t.Operator {
	case "eq":
		return fmt.Sprintf("%v = ? ", attributes[t.FieldName]), values, nil
	case "neq":
		return fmt.Sprintf("%v <> ? ", attributes[t.FieldName]), values, nil
	case "gt":
		return fmt.Sprintf("%v > ? ", attributes[t.FieldName]), values, nil
	case "lt":
		return fmt.Sprintf("%v < ? ", attributes[t.FieldName]), values, nil
	case "geq":
		return fmt.Sprintf("%v >= ? ", attributes[t.FieldName]), values, nil
	case "leq":
		return fmt.Sprintf("%v <= ? ", attributes[t.FieldName]), values, nil
	case "between":
		if len(t.Values) != 2 {
			return "", []interface{}{}, errors.InvalidArgumentError("the `between` filter needs 2 values")
		}
		return fmt.Sprintf("%v BETWEEN ? AND ? ", attributes[t.FieldName]), values, nil
	case "in":
		return fmt.Sprintf("%v IN (?) ", attributes[t.FieldName]), values, nil
	case "notin":
		return fmt.Sprintf("%v NOT IN (?) ", attributes[t.FieldName]), values, nil
	default:
		return "", []interface{}{}, errors.InvalidArgumentError(fmt.Sprintf("Unknown filter : %v", t.Operator))
	}
}

func ParseFilterValue(field string, v string) (Filter, error) {
	idx := strings.Index(v, ":")

	var operator string
	var value string

	if idx < 0 {
		operator = "eq"
		value = v
	} else {
		operator = v[:idx]
		value = v[idx+1:]
	}

	var values []string

	if operator == "between" || operator == "in" || operator == "notin" {
		values = append(values, strings.Split(value, ",")...)
	} else {
		values = []string{value}
	}
	return Filter{
		FieldName: field,
		FilterData: FilterData{
			Operator: operator,
			Values:   values,
		},
	}, nil
}
