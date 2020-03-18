package database

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type inherited struct {
	Bare string
}

type queryStruct struct {
	ID    int   `query:"am.id"`
	Int64 int64 `query:"am.id"`
	Int32 int64 `query:"am.id"`
	Bool  bool  `query:"am.id != 0"`
	// We have to take gorm column name into account
	Note    string     `gorm:"column:note_str" query:"COALESCE(am.text_note, '')"`
	Date    time.Time  `gorm:"column:date"`
	DatePtr *time.Time `gorm:"column:date"`
	inherited
}

type queryInvalid struct {
	// Not a nested struct, should fail
	Test *inherited
	queryStruct
}

func testQueryAttrsOk(t *testing.T, v interface{}) {
	attrs, _, err := GetQueryAttrs(v)
	assert.NoError(t, err)

	assert.NotNil(t, attrs["id"].Parser)
	assert.Equal(t, "am.id", attrs["id"].Query)

	assert.NotNil(t, attrs["note_str"].Parser)
	assert.Equal(t, "COALESCE(am.text_note, '')", attrs["note_str"].Query)

	assert.NotNil(t, attrs["bare"].Parser)
	assert.Equal(t, "bare", attrs["bare"].Query)
}

func TestGetAttrs(t *testing.T) {
	testQueryAttrsOk(t, queryStruct{})
	testQueryAttrsOk(t, &queryStruct{})
	testQueryAttrsOk(t, []queryStruct{})

	_, _, err := GetQueryAttrs([]string{})
	assert.Error(t, err)

	_, _, err = GetQueryAttrs(queryInvalid{})
	assert.Error(t, err)

	_, _, err = GetQueryAttrs(0)
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

	assert.Contains(t, sel, "am.id as id")
	assert.Contains(t, sel, "COALESCE(am.text_note, '') as note_str")
	assert.Contains(t, sel, "bare as bare")
}
