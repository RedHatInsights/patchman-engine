package controllers

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFirst(t *testing.T) {
	pager := pager{"/", 0, 10, 1000, ""}
	assert.Equal(t, "/?offset=10&limit=10", *pager.createNextLink())
	assert.Equal(t, "/?offset=990&limit=10", pager.createLastLink())
	assert.Nil(t, pager.createPreviousLink())
}

func TestMiddle(t *testing.T) {
	pager := pager{"/", 20, 10, 1000, ""}
	assert.Equal(t, "/?offset=30&limit=10", *pager.createNextLink())
	assert.Equal(t, "/?offset=990&limit=10", pager.createLastLink())
	assert.Equal(t, "/?offset=10&limit=10", *pager.createPreviousLink())
}

func TestLast(t *testing.T) {
	pager := pager{"/", 990, 10, 1000, ""}
	assert.Nil(t, pager.createNextLink())
	assert.Equal(t, "/?offset=990&limit=10", pager.createLastLink())
	assert.Equal(t, "/?offset=980&limit=10", *pager.createPreviousLink())
}

func TestFewItems(t *testing.T) {
	pager := pager{"/", 0, 10, 8, ""}
	assert.Nil(t, pager.createNextLink())
	assert.Equal(t, "/?offset=0&limit=10", pager.createLastLink())
	assert.Nil(t, pager.createPreviousLink())
}
