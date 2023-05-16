package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNevraParse(t *testing.T) {
	nevra, err := ParseNevra("389-ds-base-0:1.3.7-1.fc27.src")
	assert.Equal(t, nil, err)
	assert.Equal(t, "389-ds-base", nevra.Name)
	assert.Equal(t, "1.3.7", nevra.Version)
	assert.Equal(t, 0, nevra.Epoch)
	assert.Equal(t, "1.fc27", nevra.Release)
	assert.Equal(t, "src", nevra.Arch)

	nevra, err = ParseNevra("389-ds-base-1.3.7-1.fc27.src")
	assert.NoError(t, err)
	assert.Equal(t, -1, nevra.Epoch)
}

func TestNevraParse2(t *testing.T) {
	nevra, err := ParseNevra("firefox-1:76.0.1-1.fc31.x86_64")
	assert.NoError(t, err)
	assert.Equal(t, "firefox", nevra.Name)
	assert.Equal(t, 1, nevra.Epoch)
	_, err = ParseNevra("kernel-5.6.13-200.fc31.x86_64")
	assert.NoError(t, err)
}

func TestNevraParse3(t *testing.T) {
	nevra, err := ParseNevra("connectwisecontrol-1330664eb22f9e21-0:21.14.5924.8013-.noarch")
	assert.NoError(t, err)
	assert.Equal(t, "connectwisecontrol-1330664eb22f9e21", nevra.Name)
	assert.Equal(t, 0, nevra.Epoch)
	assert.Equal(t, "", nevra.Release)
}

func TestNevraParse4(t *testing.T) {
	nevra1, err := ParseNevra("rh-ruby24-rubygems-2.6.14.4-92.el7.noarch")
	assert.Equal(t, nil, err)
	assert.Equal(t, "rh-ruby24-rubygems", nevra1.Name)
	assert.Equal(t, "2.6.14.4", nevra1.Version)
	assert.Equal(t, -1, nevra1.Epoch)
	assert.Equal(t, "92.el7", nevra1.Release)

	nevra2, err := ParseNevra("rh-ruby24-rubygems-2.6.14-90.el7.noarch")
	assert.Equal(t, nil, err)
	assert.Equal(t, "rh-ruby24-rubygems", nevra2.Name)
	assert.Equal(t, "2.6.14", nevra2.Version)
	assert.Equal(t, -1, nevra2.Epoch)
	assert.Equal(t, "90.el7", nevra2.Release)

	cmp := nevra1.Cmp(nevra2)
	assert.Equal(t, 1, cmp)
}

func TestNevraCmp(t *testing.T) {
	ff0, err := ParseNevra("firefox-76.0.1-1.fc31.x86_64")
	assert.NoError(t, err)
	assert.Equal(t, -1, ff0.Epoch)
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

	// epoch
	assert.Equal(t, -1, ff1.Cmp(ff2))
	// version
	assert.Equal(t, 1, ff3.Cmp(ff2))
	// release
	assert.Equal(t, 1, ff4.Cmp(ff3))
	// name
	assert.Equal(t, 1, ff4.Cmp(fb4))
}
