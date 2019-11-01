package utils

import (
	"github.com/bmizerany/assert"
	"testing"
)


func TestNevraParse(t *testing.T) {
	nevra, err := ParseRpmName("389-ds-base-1.3.7.8-1.fc27.src")
	assert.Equal(t, nil, err)
	assert.Equal(t, "src", nevra.Arch)
}
