package controllers

import (
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
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

// Parse a filter from field name and field value specification
func ParseFilterValue(field string, val string) (Filter, error) {
	idx := strings.Index(val, ":")

	var operator string
	var value string

	if idx < 0 {
		operator = "eq"
		value = val
	} else {
		operator = val[:idx]
		value = val[idx+1:]
	}

	values := strings.Split(value, ",")

	return Filter{
		FieldName: field,
		FilterData: FilterData{
			Operator: operator,
			Values:   values,
		},
	}, nil
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
			return "", []interface{}{}, errors.New("the `between` filter needs 2 values")
		}
		return fmt.Sprintf("%v BETWEEN ? AND ? ", attributes[t.FieldName]), values, nil
	case "in":
		return fmt.Sprintf("%v IN (?) ", attributes[t.FieldName]), values, nil
	case "notin":
		return fmt.Sprintf("%v NOT IN (?) ", attributes[t.FieldName]), values, nil
	default:
		return "", []interface{}{}, errors.New(fmt.Sprintf("Unknown filter : %v", t.Operator))
	}
}

type Filters []Filter

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

func (t *Filters) Apply(tx *gorm.DB, fields AttrMap) (*gorm.DB, error) {
	for _, f := range *t {
		query, args, err := f.ToWhere(fields)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("Invalid filter: %v", f.FieldName))
		}
		tx = tx.Where(query, args...)
	}
	return tx, nil
}
