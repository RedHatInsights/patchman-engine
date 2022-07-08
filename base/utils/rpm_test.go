package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNevraParse(t *testing.T) {
	nevra, err := ParseNevra("389-ds-base-1.3.7-1.fc27.src")
	assert.Equal(t, nil, err)
	assert.Equal(t, "389-ds-base", nevra.Name)
	assert.Equal(t, "1.3.7", nevra.Version)
	assert.Equal(t, 0, nevra.Epoch)
	assert.Equal(t, "1.fc27", nevra.Release)
	assert.Equal(t, "src", nevra.Arch)
}

func TestNevraParse2(t *testing.T) {
	nevra, err := ParseNevra("firefox-1:76.0.1-1.fc31.x86_64")
	assert.NoError(t, err)
	assert.Equal(t, "firefox", nevra.Name)
	assert.Equal(t, 1, nevra.Epoch)
	_, err = ParseNevra("kernel-5.6.13-200.fc31.x86_64")
	assert.NoError(t, err)
}
