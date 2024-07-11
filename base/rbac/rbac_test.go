package rbac

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

var data = []byte(`
{
	"resourceDefinitions": [
		{"attributeFilter": {
			"key": "single_string",
			"operation": "equal",
			"value": "string"
		}},
		{"attributeFilter": {
			"key": "comma_separated",
			"operation": "equal",
			"value": "comma,separated"
		}},
		{"attributeFilter": {
			"key": "null",
			"operation": "equal",
			"value": null
		}},
		{"attributeFilter": {
			"key": "string_array",
			"operation": "in",
			"value": ["string", "array"]
		}},
		{"attributeFilter": {
			"key": "string_array_with_null",
			"operation": "in",
			"value": ["string", "array", null]
		}},
		{"attributeFilter": {
			"key": "null_array",
			"operation": "in",
			"value": [null]
		}},
		{"attributeFilter": {
			"key": "empty_array",
			"operation": "in",
			"value": []
		}}
	]
}
`)

func TestParsing(t *testing.T) {
	stringS := "string"
	commaS := "comma,separated"
	arrayS := "array"

	expected := []ResourceDefinition{
		{AttributeFilter: AttributeFilter{Operation: "equal", Key: "single_string", Value: []*string{&stringS}}},
		{AttributeFilter: AttributeFilter{Operation: "equal", Key: "comma_separated", Value: []*string{&commaS}}},
		{AttributeFilter: AttributeFilter{Operation: "equal", Key: "null", Value: []*string{nil}}},
		{AttributeFilter: AttributeFilter{Operation: "in", Key: "string_array", Value: []*string{&stringS, &arrayS}}},
		{AttributeFilter: AttributeFilter{Operation: "in", Key: "string_array_with_null",
			Value: []*string{&stringS, &arrayS, nil}}},
		{AttributeFilter: AttributeFilter{Operation: "in", Key: "null_array", Value: []*string{nil}}},
		{AttributeFilter: AttributeFilter{Operation: "in", Key: "empty_array", Value: []*string{}}},
	}

	var v Access
	err := json.Unmarshal(data, &v)
	if assert.NoError(t, err) {
		assert.Equal(t, expected, v.ResourceDefinitions)
	}
}
