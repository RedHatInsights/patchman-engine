package controllers

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func ParseResponseBody(t *testing.T, bytes []byte, out interface{}) {
	err := json.Unmarshal(bytes, out)
	assert.Nil(t, err, string(bytes))
}
