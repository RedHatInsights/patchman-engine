package listener

import (
	"app/base/core"
	"app/base/database"
	"app/base/utils"
	"testing"

	"github.com/stretchr/testify/assert"
)

func int64ptr(x int64) *int64 {
	return &x
}

func TestAssignTemplates(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()

	type result struct {
		templateID *int64
		isErr      bool
	}
	tests := []struct {
		name         string
		environments []string
		res          result
	}{
		{"NoEnvironments", []string{}, result{nil, false}},
		{"NoTemplatesFound", []string{"env-with-no-template"}, result{nil, false}},
		{"SingleTemplate", []string{"99900000000000000000000000000001", "env-with-no-template"}, result{int64ptr(1), false}},
		{
			"MultipleTemplatesFound", []string{"99900000000000000000000000000001", "99900000000000000000000000000002"},
			result{int64ptr(1), false},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			templateID, err := getTemplate(database.DB, 1, test.environments)
			isErrReturned := err != nil
			assert.Equal(t, test.res.isErr, isErrReturned)
			if test.res.templateID == nil {
				assert.Equal(t, test.res.templateID, templateID)
			} else {
				assert.Equal(t, *test.res.templateID, *templateID)
			}
		})
	}
}
