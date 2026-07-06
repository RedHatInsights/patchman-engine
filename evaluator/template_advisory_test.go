package evaluator

import (
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"app/base/vmaas"
	"context"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
)

func TestApplyTemplateAdvisoryInstallability(t *testing.T) {
	erratumInstallable := "RHSA-2023:3246"
	erratumApplicable := "RHSA-2023:3240"
	vmaasDataResp := `
	{
		"update_list": {
			"git-2.30.1-1.el8_8.x86_64": {
				"available_updates": [
					{
						"erratum": "RHSA-2023:3246",
						"package": "git-2.39.3-1.el8_8.x86_64"
					},
					{
						"erratum": "RHSA-2023:3240",
						"package": "git-2.39.4-1.el8_8.x86_64"
					}
				]
			}
		}
	}
	`
	var vmaasData vmaas.UpdatesV3Response
	err := sonic.Unmarshal([]byte(vmaasDataResp), &vmaasData)
	assert.Nil(t, err)

	templateErrata := map[string]struct{}{
		erratumInstallable: {},
	}
	applyTemplateAdvisoryInstallability(&vmaasData, templateErrata)

	updateList := vmaasData.GetUpdateList()["git-2.30.1-1.el8_8.x86_64"].GetAvailableUpdates()
	assert.Equal(t, 2, len(updateList))
	assert.Equal(t, INSTALLABLE, updateList[0].StatusID)
	assert.Equal(t, erratumInstallable, updateList[0].GetErratum())
	assert.Equal(t, APPLICABLE, updateList[1].StatusID)
	assert.Equal(t, erratumApplicable, updateList[1].GetErratum())
}

func TestLoadTemplateAdvisoryErrata(t *testing.T) {
	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()

	templateID := int64(1)
	database.CreateTemplateAdvisories(t, 1, templateID, []int64{1, 2})
	defer database.DeleteTemplateAdvisories(t, templateID, []int64{1, 2})

	errata, err := loadTemplateAdvisoryErrata(database.DB, 1, templateID)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(errata))
	_, ok := errata["RH-1"]
	assert.True(t, ok)
	_, ok = errata["RH-2"]
	assert.True(t, ok)
}

//nolint:funlen
func TestGetUpdatesDataTemplateAdvisoryEval(t *testing.T) {
	vmaasJSON := `
	{
		"package_list": ["git-2.30.1-1.el8_8.x86_64"],
		"repository_list": ["rhel-8-for-x86_64-appstream-rpms"],
		"releasever": "8",
		"basearch": "x86_64",
		"latest_only": true
	}
	`
	vmaasDataResp := `
	{
		"update_list": {
			"git-2.30.1-1.el8_8.x86_64": {
				"available_updates": [
					{
						"erratum": "RH-1",
						"package": "git-2.39.3-1.el8_8.x86_64"
					},
					{
						"erratum": "RH-2",
						"package": "git-2.39.4-1.el8_8.x86_64"
					}
				]
			}
		}
	}
	`
	yumUpdatesRaw := []byte(`
	{
		"update_list": {
			"git-2.30.1-1.el8_8.x86_64": {
				"available_updates": [
					{
						"erratum": "RH-100",
						"package": "git-9.9.9-1.el8_8.x86_64"
					}
				]
			}
		}
	}
	`)

	var vmaasData vmaas.UpdatesV3Response
	assert.Nil(t, sonic.Unmarshal([]byte(vmaasDataResp), &vmaasData))
	vmaasJSONChecksum := "template-advisory-eval-test"

	templateID := int64(1)
	system := &models.SystemPlatformV2{
		Inventory: models.SystemInventory{
			RhAccountID:  1,
			JSONChecksum: &vmaasJSONChecksum,
			VmaasJSON:    &vmaasJSON,
			YumUpdates:   yumUpdatesRaw,
		},
		Patch: models.SystemPatch{
			TemplateID: &templateID,
		},
	}

	utils.SkipWithoutDB(t)
	core.SetupTestEnvironment()
	configure()
	loadCache()

	ogYum := enableYumUpdatesEval
	ogTemplateEval := enableTemplateAdvisoryEval
	enableYumUpdatesEval = true
	enableTemplateAdvisoryEval = true
	defer func() {
		enableYumUpdatesEval = ogYum
		enableTemplateAdvisoryEval = ogTemplateEval
	}()

	memoryVmaasCache.Add(&vmaasJSONChecksum, &vmaasData)
	database.CreateTemplateAdvisories(t, 1, templateID, []int64{1})
	defer database.DeleteTemplateAdvisories(t, templateID, []int64{1})

	result, err := getUpdatesData(context.Background(), system)
	assert.Nil(t, err)
	assert.NotNil(t, result)

	var installableCnt, applicableCnt int
	for _, updates := range result.GetUpdateList() {
		for _, update := range updates.GetAvailableUpdates() {
			switch update.StatusID {
			case INSTALLABLE:
				installableCnt++
			case APPLICABLE:
				applicableCnt++
			}
		}
	}
	assert.Equal(t, 1, installableCnt)
	assert.Equal(t, 1, applicableCnt)
}

func TestUseTemplateAdvisoryEval(t *testing.T) {
	templateID := int64(1)
	system := &models.SystemPlatformV2{
		Patch: models.SystemPatch{TemplateID: &templateID},
	}

	ogTemplateEval := enableTemplateAdvisoryEval
	enableTemplateAdvisoryEval = true
	defer func() { enableTemplateAdvisoryEval = ogTemplateEval }()

	assert.True(t, useTemplateAdvisoryEval(system))

	system.Inventory.SatelliteManaged = true
	assert.False(t, useTemplateAdvisoryEval(system))
}
