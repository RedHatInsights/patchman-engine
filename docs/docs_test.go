package docs

import (
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

func TestValidateOpenAPI3DocStr(t *testing.T) {
	doc, err := ioutil.ReadFile("openapi.json")
	assert.Nil(t, err)
	_, err = openapi3.NewSwaggerLoader().LoadSwaggerFromData(doc)
	assert.Nil(t, err)
}
