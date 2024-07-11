package rbac

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParsing(t *testing.T) {
	data := []byte(`
		{
			"resourceDefinitions": [
				{"attributeFilter": {
					"key": "single_string",
					"value": "string"
				}},
				{"attributeFilter": {
					"key": "comma_separated",
					"value": "comma,separated"
				}},
				{"attributeFilter": {
					"key": "null",
					"value": null
				}},
				{"attributeFilter": {
					"key": "string_array",
					"value": ["string", "array"]
				}},
				{"attributeFilter": {
					"key": "string_array_with_null",
					"value": ["string", "array", null]
				}},
				{"attributeFilter": {
					"key": "null_array",
					"value": [null]
				}},
				{"attributeFilter": {
					"key": "empty_array",
					"value": []
				}}
			]
		}
	`)
	stringS := "string"
	commaS := "comma,separated"
	arrayS := "array"

	expected := []ResourceDefinition{
		{AttributeFilter: AttributeFilter{Key: "single_string", Value: []*string{&stringS}}},
		{AttributeFilter: AttributeFilter{Key: "comma_separated", Value: []*string{&commaS}}},
		{AttributeFilter: AttributeFilter{Key: "null", Value: []*string{nil}}},
		{AttributeFilter: AttributeFilter{Key: "string_array", Value: []*string{&stringS, &arrayS}}},
		{AttributeFilter: AttributeFilter{Key: "string_array_with_null", Value: []*string{&stringS, &arrayS, nil}}},
		{AttributeFilter: AttributeFilter{Key: "null_array", Value: []*string{nil}}},
		{AttributeFilter: AttributeFilter{Key: "empty_array", Value: []*string{}}},
	}

	var v Access
	err := json.Unmarshal(data, &v)
	if assert.NoError(t, err) {
		assert.Equal(t, expected, v.ResourceDefinitions)
	}
}
