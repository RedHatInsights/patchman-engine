package controllers

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

const InvalidContentTypeErr = `{"error":"Invalid content type 'test-format', use 'application/json' or 'text/csv'"}`

func ParseResponseBody(t *testing.T, bytes []byte, out interface{}) {
	// don't use sonic.Unmarshal as some tests receive empty output
	err := json.Unmarshal(bytes, &out)
	assert.Nil(t, err, string(bytes))
}
