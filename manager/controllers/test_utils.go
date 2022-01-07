package controllers

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

func ParseResponseBody(t *testing.T, bytes []byte, out interface{}) {
	err := json.Unmarshal(bytes, out)
	assert.Nil(t, err, string(bytes))
}
