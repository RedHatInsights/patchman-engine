package evaluator

import (
	"app/base"
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"app/base/vmaas"
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

	errata, err := loadTemplateAdvisoryErrata(database.TestDB(), 1, templateID)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(errata))
	_, ok := errata["RH-1"]
	assert.True(t, ok)
	_, ok = errata["RH-2"]
	assert.True(t, ok)
}

func TestGetUpdatesDataTemplateAdvisoryEval(t *testing.T) {
	// yum update must not be merged; template installability comes from VMaaS errata only
	yumUpdatesRaw := []byte(`{
		"update_list": {
			"firefox-0:76.0.1-1.fc31.x86_64": {
				"available_updates": [
					{
						"erratum": "RH-YUM-ONLY",
						"package": "firefox-9:9.9.9-1.fc31.x86_64"
					}
				]
			}
		}
	}`)

	vmaasJSON := platformPackagesVmaasJSON
	templateID := int64(1)
	system := &models.SystemPlatformV2{
		Inventory: models.SystemInventory{
			RhAccountID: 1,
			VmaasJSON:   &vmaasJSON,
			YumUpdates:  yumUpdatesRaw,
		},
		Patch: models.SystemPatch{
			TemplateID: &templateID,
		},
	}

	utils.SkipWithoutDB(t)
	utils.SkipWithoutPlatform(t)
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

	database.CreateTemplateAdvisories(t, 1, templateID, []int64{1})
	defer database.DeleteTemplateAdvisories(t, templateID, []int64{1})

	result, err := getUpdatesData(base.Context, system)
	assert.Nil(t, err)
	assert.NotNil(t, result)

	// platform /updates returns RH-1, RH-2, and RH-100. Template advisory id 1 is RH-1.
	total, statuses := updateStatuses(result)
	assert.Equal(t, 3, total)
	assert.Equal(t, INSTALLABLE, statuses["RH-1"])
	assert.Equal(t, APPLICABLE, statuses["RH-2"])
	assert.Equal(t, APPLICABLE, statuses["RH-100"])
	_, yumMerged := statuses["RH-YUM-ONLY"]
	assert.False(t, yumMerged)
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
