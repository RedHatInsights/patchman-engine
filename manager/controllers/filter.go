package controllers

import (
	"app/base/database"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type FilterType int8

const (
	ColumnFilter FilterType = iota
	InventoryFilter
	TagFilter
)

const (
	OpEq      = "eq"
	OpNeq     = "neq"
	OpGt      = "gt"
	OpLt      = "lt"
	OpGte     = "gte"
	OpLte     = "lte"
	OpBetween = "between"
	OpIn      = "in"
	OpNotIn   = "notin"
	OpNull    = "null"
	OpNotNull = "notnull"
)

type FilterData struct {
	Type     FilterType `json:"-"`
	Operator string     `json:"op"`
	Values   []string   `json:"values"`
}

type Filters map[string]FilterData

// Parse a filter from field name and field value specification
func ParseFilterValue(ftype FilterType, val string) FilterData {
	idx := strings.Index(val, ":")

	var operator string
	var value string

	if idx < 0 {
		operator = OpEq
		value = val
	} else {
		operator = val[:idx]
		value = val[idx+1:]
	}

	values := strings.Split(value, ",")

	return FilterData{
		Type:     ftype,
		Operator: operator,
		Values:   values,
	}
}

func checkValueCount(operator string, nValues int) bool {
	switch operator {
	case OpBetween:
		return nValues == 2
	case OpIn:
		fallthrough
	case OpNotIn:
		return nValues > 0
	default:
		return nValues == 1
	}
}

// Convert a single filter to where clauses
func (t *FilterData) ToWhere(fieldName string, attributes database.AttrMap) (string, []any, error) {
	var err error
	transformedOperator, transformedValues := t.transformParams(fieldName)
	var values = make([]any, len(transformedValues))
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
	case OpEq:
		return fmt.Sprintf("%s = ? ", attributes[fieldName].DataQuery), values, nil
	case OpNeq:
		return fmt.Sprintf("%s <> ? ", attributes[fieldName].DataQuery), values, nil
	case OpGt:
		return fmt.Sprintf("%s > ? ", attributes[fieldName].DataQuery), values, nil
	case OpLt:
		return fmt.Sprintf("%s < ? ", attributes[fieldName].DataQuery), values, nil
	case OpGte:
		return fmt.Sprintf("%s >= ? ", attributes[fieldName].DataQuery), values, nil
	case OpLte:
		return fmt.Sprintf("%s <= ? ", attributes[fieldName].DataQuery), values, nil
	case OpBetween:
		return fmt.Sprintf("%s BETWEEN ? AND ? ", attributes[fieldName].DataQuery), values, nil
	case OpIn:
		return fmt.Sprintf("%s IN (?) ", attributes[fieldName].DataQuery), []any{values}, nil
	case OpNotIn:
		return fmt.Sprintf("%s NOT IN (?) ", attributes[fieldName].DataQuery), []any{values}, nil
	case OpNull:
		return fmt.Sprintf("%s IS NULL ", attributes[fieldName].DataQuery), []any{}, nil
	case OpNotNull:
		return fmt.Sprintf("%s IS NOT NULL ", attributes[fieldName].DataQuery), []any{}, nil
	default:
		return "", []any{}, errors.Errorf("Unknown filter : %s", t.Operator)
	}
}

func (t *FilterData) transformParams(fieldName string) (transformedOperator string, transformedValues []string) {
	if len(t.Values) == 1 && (t.Values[0] == "null" || t.Values[0] == "notnull") {
		// handle special cases filter=null and filter=notnull
		return t.Values[0], []string{"0"}
	}
	if fieldName != "advisory_type_name" {
		// no change
		return t.Operator, t.Values
	}

	// special case, in advisory_type_name filter expand "other" into list of what we mean by other
	transformedValues = make([]string, 0, len(t.Values))
	for _, originalValue := range t.Values {
		if originalValue == "other" {
			transformedValues = append(transformedValues, database.OtherAdvisoryTypes...)
		} else {
			transformedValues = append(transformedValues, originalValue)
		}
	}

	if len(transformedValues) == len(t.Values) {
		return t.Operator, t.Values
	}

	switch t.Operator {
	case OpEq:
		transformedOperator = OpIn
	case OpNeq:
		transformedOperator = OpNotIn
	default:
		transformedOperator = t.Operator
	}
	return transformedOperator, transformedValues
}

func (t Filters) ToQueryParams() string {
	parts := make([]string, 0, len(t))
	for name, v := range t {
		values := strings.Join(v.Values, ",")
		if v.Type == TagFilter {
			parts = append(parts, fmt.Sprintf("tags=%s=%s", name, values))
		} else {
			parts = append(parts, fmt.Sprintf("filter[%s]=%s:%s", name, v.Operator, values))
		}
	}
	return strings.Join(parts, "&")
}

func (t Filters) Apply(tx *gorm.DB, fields database.AttrMap) (*gorm.DB, error) {
	for name, f := range t {
		if f.Type != ColumnFilter {
			continue
		}
		query, args, err := f.ToWhere(name, fields)
		if err != nil {
			return nil, err
		}
		tx = tx.Where(query, args...)
	}
	return tx, nil
}

func (t Filters) Update(ftype FilterType, key string, value string) {
	data := ParseFilterValue(ftype, value)
	if fdata, ok := t[key]; ok {
		data.Values = append(data.Values, fdata.Values...)
	}
	t[key] = data
}
