package controllers

import (
	"app/base/database"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type FilterData struct {
	Operator string   `json:"op"`
	Values   []string `json:"values"`
}

type Filters map[string]FilterData

// Parse a filter from field name and field value specification
func ParseFilterValue(val string) FilterData {
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
	}
}

func checkValueCount(operator string, nValues int) bool {
	switch operator {
	case "between":
		return nValues == 2
	case "in":
		fallthrough
	case "notin": // nolint: goconst
		return nValues > 0
	default:
		return nValues == 1
	}
}

// Convert a single filter to where clauses
func (t *FilterData) ToWhere(fieldName string, attributes database.AttrMap) (string, []interface{}, error) {
	var err error
	transformedValues, transformedOperator := transformFilterParams(fieldName, t.Values, t.Operator)
	var values = make([]interface{}, len(transformedValues))
	for i, v := range transformedValues {
		fieldInfo, found := attributes[fieldName]
		if !found {
			return "", nil, errors.Errorf("Unknown field: %s", fieldName)
		}

		values[i], err = fieldInfo.Parser(v)
		if err != nil {
			return "", nil, errors.Wrapf(err, "Invalid filter value %s for %s", v, fieldName)
		}
	}

	if !checkValueCount(transformedOperator, len(transformedValues)) {
		return "", nil,
			errors.Errorf("Invalid number of values: %v for operator '%s'", len(t.Values), t.Operator)
	}
	// We need to look up expression used to create the attribute, because FROM clause can't contain
	// column aliases
	switch transformedOperator {
	case "eq":
		return fmt.Sprintf("%s = ? ", attributes[fieldName].DataQuery), values, nil
	case "neq":
		return fmt.Sprintf("%s <> ? ", attributes[fieldName].DataQuery), values, nil
	case "gt":
		return fmt.Sprintf("%s > ? ", attributes[fieldName].DataQuery), values, nil
	case "lt":
		return fmt.Sprintf("%s < ? ", attributes[fieldName].DataQuery), values, nil
	case "geq":
		return fmt.Sprintf("%s >= ? ", attributes[fieldName].DataQuery), values, nil
	case "leq":
		return fmt.Sprintf("%s <= ? ", attributes[fieldName].DataQuery), values, nil
	case "between":
		return fmt.Sprintf("%s BETWEEN ? AND ? ", attributes[fieldName].DataQuery), values, nil
	case "in":
		return fmt.Sprintf("%s IN (?) ", attributes[fieldName].DataQuery), []interface{}{values}, nil
	case "notin":
		return fmt.Sprintf("%s NOT IN (?) ", attributes[fieldName].DataQuery), []interface{}{values}, nil
	default:
		return "", []interface{}{}, errors.Errorf("Unknown filter : %s", t.Operator)
	}
}

// transformFilterParams Allow exceptions in ToWhere values usage (e.g. "other").
func transformFilterParams(fieldName string, originalValues []string, originalOperator string) (
	transformedValues []string, transformedOperator string) {
	if fieldName != "advisory_type_name" {
		return originalValues, originalOperator
	}

	transformedValues = make([]string, 0, len(originalValues))
	for _, originalValue := range originalValues {
		if originalValue == "other" {
			transformedValues = append(transformedValues, database.OtherAdvisoryTypes...)
		} else {
			transformedValues = append(transformedValues, originalValue)
		}
	}

	if len(transformedValues) == len(originalValues) {
		return originalValues, originalOperator
	}

	switch originalOperator {
	case "eq":
		transformedOperator = "in"
	case "neq":
		transformedOperator = "notin"
	default:
		transformedOperator = originalOperator
	}
	return transformedValues, transformedOperator
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

func (t Filters) Update(key string, value string) {
	data := ParseFilterValue(value)
	if fdata, ok := t[key]; ok {
		data.Values = append(data.Values, fdata.Values...)
	}
	t[key] = data
}
