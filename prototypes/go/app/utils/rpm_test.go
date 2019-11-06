package utils

import (
	"github.com/bmizerany/assert"
	"testing"
)


func TestNevraParse(t *testing.T) {
	nevra, err := ParseNevra("389-ds-base-1.3.7.8-1.fc27.src")
	assert.Equal(t, nil, err)
	assert.Equal(t, "389-ds-base", nevra.Name)
	assert.Equal(t, "1.3.7.8", nevra.Version)
	assert.Equal(t, "1.fc27", nevra.Release)
	assert.Equal(t, "src", nevra.Arch)
}
