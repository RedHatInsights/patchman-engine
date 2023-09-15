package controllers

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

const InvalidContentTypeErr = `{"error":"Invalid content type 'test-format', use 'application/json' or 'text/csv'"}`

func ParseResponseBody(t *testing.T, bytes []byte, out interface{}) {
	err := json.Unmarshal(bytes, out)
	assert.Nil(t, err, string(bytes))
}
