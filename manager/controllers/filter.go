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

func (t *FilterData) CheckValueCount(attrMap database.AttrMap) bool {
	switch t.Operator {
	case "between":
		return len(t.Values) == 2
	case "in":
		fallthrough
	case "notin":
		return len(t.Values) > 0
	default:
		return len(t.Values) == 1
	}
}

// Convert a single filter to where clauses
func (t *FilterData) ToWhere(fieldName string, attributes database.AttrMap) (string, []interface{}, error) {
	var err error
	var values = make([]interface{}, len(t.Values))
	for i, v := range t.Values {
		fieldInfo, found := attributes[fieldName]
		if !found {
			return "", nil, errors.Errorf("Unknown field: %s", fieldName)
		}

		values[i], err = fieldInfo.Parser(v)
		if err != nil {
			return "", nil, errors.Wrapf(err, "Invalid filter value %s for %s", v, fieldName)
		}
	}

	if !t.CheckValueCount(attributes) {
		return "", nil,
			errors.Errorf("Invalid number of values: %v for operator '%s'", len(t.Values), t.Operator)
	}
	// We need to look up expression used to create the attribute, because FROM clause can't contain
	// column aliases
	switch t.Operator {
	case "eq":
		return fmt.Sprintf("%s = ? ", attributes[fieldName].Query), values, nil
	case "neq":
		return fmt.Sprintf("%s <> ? ", attributes[fieldName].Query), values, nil
	case "gt":
		return fmt.Sprintf("%s > ? ", attributes[fieldName].Query), values, nil
	case "lt":
		return fmt.Sprintf("%s < ? ", attributes[fieldName].Query), values, nil
	case "geq":
		return fmt.Sprintf("%s >= ? ", attributes[fieldName].Query), values, nil
	case "leq":
		return fmt.Sprintf("%s <= ? ", attributes[fieldName].Query), values, nil
	case "between":
		if len(t.Values) != 2 {
			return "", []interface{}{}, errors.New("the `between` filter needs 2 values")
		}
		return fmt.Sprintf("%s BETWEEN ? AND ? ", attributes[fieldName].Query), values, nil
	case "in":
		return fmt.Sprintf("%s IN (?) ", attributes[fieldName].Query), []interface{}{values}, nil
	case "notin":
		return fmt.Sprintf("%s NOT IN (?) ", attributes[fieldName].Query), []interface{}{values}, nil
	default:
		return "", []interface{}{}, errors.New(fmt.Sprintf("Unknown filter : %s", t.Operator))
	}
}

func (t Filters) ToQueryParams() string {
	parts := make([]string, 0, len(t))
	for name, v := range t {
		values := strings.Join(v.Values, ",")
		parts = append(parts, fmt.Sprintf("filter[%s]=%s:%s", name, v.Operator, values))
	}
	return strings.Join(parts, "&")
}

func (t Filters) Apply(tx *gorm.DB, fields database.AttrMap) (*gorm.DB, error) {
	for name, f := range t {
		query, args, err := f.ToWhere(name, fields)
		if err != nil {
			return nil, err
		}
		tx = tx.Where(query, args...)
	}
	return tx, nil
}
