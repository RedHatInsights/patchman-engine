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

func TestNevraCmp(t *testing.T) {
	ff0, err := ParseNevra("firefox-76.0.1-1.fc31.x86_64")
	assert.NoError(t, err)
	ff1, err := ParseNevra("firefox-0:76.0.1-1.fc31.x86_64")
	assert.NoError(t, err)
	ff2, err := ParseNevra("firefox-1:76.0.1-1.fc31.x86_64")
	assert.NoError(t, err)
	ff3, err := ParseNevra("firefox-1:77.0.1-1.fc31.x86_64")
	assert.NoError(t, err)
	ff4, err := ParseNevra("firefox-1:77.0.1-1.fc33.x86_64")
	assert.NoError(t, err)
	fb4, err := ParseNevra("firebird-1:77.0.1-1.fc33.x86_64")
	assert.NoError(t, err)

	assert.Equal(t, 0, ff0.Cmp(ff1))
	// epoch
	assert.Equal(t, -1, ff1.Cmp(ff2))
	// version
	assert.Equal(t, 1, ff3.Cmp(ff2))
	// release
	assert.Equal(t, 1, ff4.Cmp(ff3))
	// name
	assert.Equal(t, 1, ff4.Cmp(fb4))
}
