package listener

import (
	"app/base/core"
	"app/base/database"
	"app/base/utils"
	"context"
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

func TestCallCandlepinEnvironment(t *testing.T) {
	ctx := context.Background()
	result, err := callCandlepinEnvironment(ctx, "00000000-0000-0000-0000-000000000001")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Environments, 1)
	assert.Equal(t, "99900000000000000000000000000001", result.Environments[0].ID)
}

func TestCallCandlepinEnvironmentError(t *testing.T) {
	ctx := context.Background()
	result, err := callCandlepinEnvironment(ctx, "return_404")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "candlepin /consumers call failed")
	assert.Nil(t, result)
}

func TestCallCandlepinEnvNetworkError(t *testing.T) {
	// Override the candlepin address to invalid URL
	originalAddress := utils.CoreCfg.CandlepinAddress
	utils.CoreCfg.CandlepinAddress = "http://invalid-host:12345"
	defer func() {
		utils.CoreCfg.CandlepinAddress = originalAddress
	}()

	ctx := context.Background()
	result, err := callCandlepinEnvironment(ctx, "test-consumer")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "candlepin error")
	assert.Nil(t, result)
}
