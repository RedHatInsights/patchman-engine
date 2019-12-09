package docs

import (
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestValidateOpenAPI3DocStr(t *testing.T) {
	_, err := openapi3.NewSwaggerLoader().LoadSwaggerFromData([]byte(doc))
	assert.Nil(t, err)
}
