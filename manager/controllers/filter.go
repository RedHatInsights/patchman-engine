package controllers

import (
	"app/base/database"
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"strings"
)

type FilterData struct {
	Operator string   `json:"op"`
	Values   []string `json:"values"`
}

type Filters map[string]FilterData

// Parse a filter from field name and field value specification
func ParseFilterValue(val string) (FilterData, error) {
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

	return FilterData{
		Operator: operator,
		Values:   values,
	}, nil
}

// Convert a single filter to where clauses
func (t *FilterData) ToWhere(fieldName string, attributes database.AttrMap) (string, []interface{}, error) {
	// Gorm deals with interface{} but for ease of use we only use strings
	var values = make([]interface{}, len(t.Values))
	for i, v := range t.Values {
		values[i] = v
	}
	// We need to look up expression used to create the attribute, because FROM clause can't contain
	// column aliases
	switch t.Operator {
	case "eq":
		return fmt.Sprintf("%v = ? ", attributes[fieldName]), values, nil
	case "neq":
		return fmt.Sprintf("%v <> ? ", attributes[fieldName]), values, nil
	case "gt":
		return fmt.Sprintf("%v > ? ", attributes[fieldName]), values, nil
	case "lt":
		return fmt.Sprintf("%v < ? ", attributes[fieldName]), values, nil
	case "geq":
		return fmt.Sprintf("%v >= ? ", attributes[fieldName]), values, nil
	case "leq":
		return fmt.Sprintf("%v <= ? ", attributes[fieldName]), values, nil
	case "between":
		if len(t.Values) != 2 {
			return "", []interface{}{}, errors.New("the `between` filter needs 2 values")
		}
		return fmt.Sprintf("%v BETWEEN ? AND ? ", attributes[fieldName]), values, nil
	case "in":
		return fmt.Sprintf("%v IN (?) ", attributes[fieldName]), []interface{}{values}, nil
	case "notin":
		return fmt.Sprintf("%v NOT IN (?) ", attributes[fieldName]), []interface{}{values}, nil
	default:
		return "", []interface{}{}, errors.New(fmt.Sprintf("Unknown filter : %v", t.Operator))
	}
}

func (t *Filters) ToQueryParams() string {
	parts := make([]string, 0, len(*t))
	for name, v := range *t {
		values := strings.Join(v.Values, ",")
		parts = append(parts, fmt.Sprintf("filter[%v]=%v:%v", name, v.Operator, values))
	}
	return strings.Join(parts, "&")
}

func (t *Filters) Apply(tx *gorm.DB, fields database.AttrMap) (*gorm.DB, error) {
	for name, f := range *t {
		query, args, err := f.ToWhere(name, fields)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("Invalid filter: %v", name))
		}
		tx = tx.Where(query, args...)
	}
	return tx, nil
}
