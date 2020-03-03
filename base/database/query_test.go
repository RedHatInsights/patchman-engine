package database

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

type Inherited struct {
	Bare string
}

type queryStruct struct {
	ID int `query:"am.id"`
	// We have to take gorm column name into account
	Note string `gorm:"column:note_str" query:"COALESCE(am.text_note, '')"`
	Inherited
}

func testQueryAttrsOk(t *testing.T, v interface{}) {
	attrs, named, err := GetQueryAttrs(v)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(named))
	assert.Equal(t, []string{"id", "note_str", "bare"}, named)
	assert.Equal(t, "am.id", attrs["id"])
	assert.Equal(t, "COALESCE(am.text_note, '')", attrs["note_str"])
	assert.Equal(t, "bare", attrs["bare"])
}

func TestGetAttrs(t *testing.T) {
	testQueryAttrsOk(t, queryStruct{})
	testQueryAttrsOk(t, &queryStruct{})
	testQueryAttrsOk(t, []queryStruct{})

	_, _, err := GetQueryAttrs([]string{})
	assert.Error(t, err)

	assert.NotPanics(t, func() {
		MustGetSelect(queryStruct{})
		MustGetQueryAttrs(queryStruct{})
	})
	assert.Panics(t, func() {
		MustGetQueryAttrs([]string{})
	})
	assert.Panics(t, func() {
		MustGetSelect([]string{})
	})
}

func TestSelect(t *testing.T) {
	sel := MustGetSelect(queryStruct{})
	assert.Equal(t, "am.id as id, COALESCE(am.text_note, '') as note_str, bare as bare", sel)
}
