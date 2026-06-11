package database

import (
	"app/base"
	"app/base/inventory"
	"app/base/models"
	"app/base/utils"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

const (
	TestWorkspace1ID     = "aaaaaaaa-0000-0000-0000-000000000001"
	TestWorkspace2ID     = "aaaaaaaa-0000-0000-0000-000000000002"
	TestWorkspaceOtherID = "bbbbbbbb-0000-0000-0000-000000000003"
)

func TestWorkspacesGroup1() *inventory.Groups {
	g := inventory.Groups{{ID: TestWorkspace1ID, Name: "group1"}}
	return &g
}

func TestWorkspace1IDPtr() *uuid.UUID {
	id := uuid.MustParse(TestWorkspace1ID)
	return &id
}

func TestWorkspace1NamePtr() *string {
	name := "group1"
	return &name
}

func DebugWithCachesCheck(part string, fun func()) {
	fun()
	validAfter, err := CheckCachesValidRet()
	if err != nil {
		utils.LogPanic("error", err, "Could not check validity of caches")
	}

	if !validAfter {
		utils.LogPanic("part", part, "Cache mismatch created")
	}
}

type key struct {
	AccountID  int
	AdvisoryID int64
}

type advisoryCount struct {
	RhAccountID        int
	AdvisoryID         int64
	SystemsInstallable int
	SystemsApplicable  int
}

func CheckCachesValidRet() (bool, error) {
	valid := true
	var aad []models.AdvisoryAccountData

	tx := DB.WithContext(base.Context).Begin()
	defer tx.Rollback()
	err := tx.Set("gorm:query_option", "FOR SHARE OF advisory_account_data").
		Order("rh_account_id, advisory_id").Find(&aad).Error
	if err != nil {
		return false, err
	}
	var counts []advisoryCount

	err = tx.Select("si.rh_account_id, sa.advisory_id," +
		"count(*) filter (where sa.status_id = 0) as systems_installable," +
		"count(*) as systems_applicable").
		Table("system_advisories sa").
		Joins("JOIN system_inventory si ON sa.rh_account_id = si.rh_account_id AND sa.system_id = si.id").
		Joins("JOIN system_patch spatch ON si.id = spatch.system_id AND si.rh_account_id = spatch.rh_account_id").
		Where("si.stale = false AND spatch.last_evaluation IS NOT NULL").
		Order("si.rh_account_id, sa.advisory_id").
		Group("si.rh_account_id, sa.advisory_id").
		Find(&counts).Error
	if err != nil {
		return false, err
	}

	cached := make(map[key][]int, len(aad))
	calculated := make(map[key][]int, len(counts))

	for _, val := range aad {
		cached[key{val.RhAccountID, val.AdvisoryID}] = []int{val.SystemsInstallable, val.SystemsApplicable}
	}
	for _, val := range counts {
		calculated[key{val.RhAccountID, val.AdvisoryID}] = []int{val.SystemsInstallable, val.SystemsApplicable}
	}

	crossCheckCache := func(a, b map[key][]int) {
		for key, aCounts := range a {
			bCounts := b[key]
			if len(bCounts) == 0 {
				bCounts = []int{0, 0}
			}
			for i, msg := range []string{"installable", "applicable"} {
				if aCounts[i] != bCounts[i] {
					utils.LogError("advisory_id", key.AdvisoryID, "account_id", key.AccountID,
						"cached", aCounts[i], "calculated", bCounts[i], fmt.Sprintf("Cached %s counts mismatch", msg))
					valid = false
				}
			}
		}
	}
	crossCheckCache(cached, calculated)
	crossCheckCache(calculated, cached)

	tx.Commit()
	return valid, nil
}

func CheckCachesValid(t *testing.T) {
	valid, err := CheckCachesValidRet()
	assert.Nil(t, err)
	assert.True(t, valid)
}

func CheckAdvisoriesInDB(t *testing.T, advisories []string) []int64 {
	var advisoryIDs []int64
	err := DB.Model(models.AdvisoryMetadata{}).Where("name IN (?)", advisories).
		Pluck("id", &advisoryIDs).Error
	assert.Nil(t, err)
	assert.Equal(t, len(advisories), len(advisoryIDs))
	return advisoryIDs
}

func GetAdvisoriesByName(t *testing.T, advisories []string) []models.AdvisoryMetadata {
	var advisoryMetadata []models.AdvisoryMetadata
	err := DB.Model(models.AdvisoryMetadata{}).Where("name IN (?)", advisories).
		Find(&advisoryMetadata).Error
	assert.Nil(t, err)
	assert.Equal(t, len(advisories), len(advisoryMetadata))
	return advisoryMetadata
}

func DeleteAdvisoriesByName(t *testing.T, advisories []string) {
	err := DB.Model(models.AdvisoryMetadata{}).Where("name IN ?", advisories).
		Delete(&models.AdvisoryMetadata{}).Error
	assert.Nil(t, err)
}

func CheckThirdPartyRepos(t *testing.T, repoNames []string, thirdParty bool) {
	var reposObjs []models.Repo
	assert.Nil(t, DB.Where("name IN (?)", repoNames).Find(&reposObjs).Error)
	assert.Equal(t, len(reposObjs), len(repoNames), "loaded repos count match")
	for _, reposObj := range reposObjs {
		assert.Equal(t, thirdParty, reposObj.ThirdParty,
			fmt.Sprintf("thirdParty flag for '%s'", reposObj.Name))
	}
}

func CheckPackagesNamesInDB(t *testing.T, filter string, packageNames ...string) {
	var count int64
	query := DB.Model(&models.PackageName{}).Where("name in (?)", packageNames)
	if filter != "" {
		query = query.Where(filter)
	}
	assert.Nil(t, query.Count(&count).Error)
	assert.Equal(t, int64(len(packageNames)), count)
}

func CheckSystemJustEvaluated(t *testing.T, inventoryID uuid.UUID, nIAll, nIEnh, nIBug, nISec,
	nAAll, nAEnh, nABug, nASec, nInstall, nInstallable, nApplicable int,
	thirdParty bool) {
	var inv models.SystemInventory
	assert.Nil(t, DB.Where("inventory_id = ?", inventoryID).First(&inv).Error)
	var patch models.SystemPatch
	assert.Nil(t, DB.Where("rh_account_id = ? AND system_id = ?", inv.RhAccountID, inv.ID).First(&patch).Error)
	assert.NotNil(t, patch.LastEvaluation)
	assert.True(t, patch.LastEvaluation.After(time.Now().Add(-time.Second)))
	assert.Equal(t, nIAll, patch.InstallableAdvisoryCountCache)
	assert.Equal(t, nIEnh, patch.InstallableAdvisoryEnhCountCache)
	assert.Equal(t, nIBug, patch.InstallableAdvisoryBugCountCache)
	assert.Equal(t, nISec, patch.InstallableAdvisorySecCountCache)
	assert.Equal(t, nAAll, patch.ApplicableAdvisoryCountCache)
	assert.Equal(t, nAEnh, patch.ApplicableAdvisoryEnhCountCache)
	assert.Equal(t, nABug, patch.ApplicableAdvisoryBugCountCache)
	assert.Equal(t, nASec, patch.ApplicableAdvisorySecCountCache)
	assert.Equal(t, nInstall, patch.PackagesInstalled)
	assert.Equal(t, nInstallable, patch.PackagesInstallable)
	assert.Equal(t, nApplicable, patch.PackagesApplicable)
	assert.Equal(t, thirdParty, patch.ThirdParty)
}

func CheckAdvisoriesAccountData(t *testing.T, rhAccountID int, advisoryIDs []int64, systemsInstallable int) {
	var advisoryAccountData []models.AdvisoryAccountData
	err := DB.Where("rh_account_id = ? AND advisory_id IN (?)", rhAccountID, advisoryIDs).
		Find(&advisoryAccountData).Error
	assert.Nil(t, err)

	sum := 0
	for _, item := range advisoryAccountData {
		sum += item.SystemsInstallable
	}
	// covers both cases, when we have advisory_account_data stored with 0 systems_installable, and when we delete it
	assert.Equal(t, systemsInstallable*len(advisoryIDs), sum, "sum of systems_installable does not match")
}

func CheckAdvisoriesAccountDataNotified(t *testing.T, rhAccountID int, advisoryIDs []int64, notified bool) {
	var advisoryAccountData []models.AdvisoryAccountData
	err := DB.Where("rh_account_id = ? AND advisory_id IN (?)", rhAccountID, advisoryIDs).
		Find(&advisoryAccountData).Error
	assert.Nil(t, err)

	for _, item := range advisoryAccountData {
		if notified {
			assert.NotNil(t, item.Notified)
		} else {
			assert.Nil(t, item.Notified)
		}
	}
}

func CreateReportedAdvisories(reportedAdvisories []string, status []int) map[string]int {
	reportedAdvisoriesMap := make(map[string]int, len(reportedAdvisories))
	for i, adv := range reportedAdvisories {
		reportedAdvisoriesMap[adv] = status[i]
	}
	return reportedAdvisoriesMap
}

func CreateStoredAdvisories(advisoryPatched []int64) map[string]models.SystemAdvisories {
	systemAdvisoriesMap := make(map[string]models.SystemAdvisories, len(advisoryPatched))
	for _, advisoryID := range advisoryPatched {
		systemAdvisoriesMap["ER-"+strconv.FormatInt(advisoryID, 10)] = models.SystemAdvisories{
			AdvisoryID: advisoryID}
	}
	return systemAdvisoriesMap
}

func CreateSystemAdvisories(t *testing.T, rhAccountID int, systemID int64, advisoryIDs []int64) {
	for _, advisoryID := range advisoryIDs {
		err := DB.Create(&models.SystemAdvisories{
			RhAccountID: rhAccountID, SystemID: systemID, AdvisoryID: advisoryID, StatusID: 0}).Error
		assert.Nil(t, err)
	}
	CheckSystemAdvisories(t, systemID, advisoryIDs)
}

func CreateAdvisoryAccountData(t *testing.T, rhAccountID int, advisoryIDs []int64,
	systemsInstallable int) {
	for _, advisoryID := range advisoryIDs {
		err := DB.Create(&models.AdvisoryAccountData{
			AdvisoryID: advisoryID, RhAccountID: rhAccountID, SystemsInstallable: systemsInstallable,
			// create same number of applicable and installable systems because installable is subset of applicable
			SystemsApplicable: systemsInstallable}).Error
		assert.Nil(t, err)
	}
	CheckAdvisoriesAccountData(t, rhAccountID, advisoryIDs, systemsInstallable)
}

func CreateSystemRepos(t *testing.T, rhAccountID int, systemID int64, repoIDs []int64) {
	for _, repoID := range repoIDs {
		assert.Nil(t, DB.Create(&models.SystemRepo{RhAccountID: int64(rhAccountID),
			SystemID: systemID, RepoID: repoID}).Error)
	}
	CheckSystemRepos(t, rhAccountID, systemID, repoIDs)
}

func CheckSystemAdvisories(t *testing.T, systemID int64, advisoryIDs []int64) {
	var systemAdvisories []models.SystemAdvisories
	err := DB.Where("system_id = ? AND advisory_id IN (?)", systemID, advisoryIDs).
		Find(&systemAdvisories).Error
	assert.Nil(t, err)
	assert.Equal(t, len(advisoryIDs), len(systemAdvisories))
}

func CheckEVRAsInDB(t *testing.T, nExpected int, evras ...string) {
	var packages []models.Package
	assert.Nil(t, DB.Where("evra IN (?)", evras).Find(&packages).Error)
	assert.Equal(t, nExpected, len(packages))
}

func CheckEVRAsInDBSynced(t *testing.T, nExpected int, synced bool, evras ...string) {
	var packages []models.Package
	assert.Nil(t, DB.Where("evra IN (?)", evras).Find(&packages).Error)
	assert.Equal(t, nExpected, len(packages))
	for _, pkg := range packages {
		assert.Equal(t, synced, pkg.Synced)
	}
}

func CheckSystemPackages(t *testing.T, accountID int, systemID int64, nExpected int, packageIDs ...int64) {
	var systemPackages []models.SystemPackage
	query := DB.Where("rh_account_id = ? AND system_id = ?", accountID, systemID)
	if len(packageIDs) > 0 {
		query = query.Where("package_id IN (?)", packageIDs)
	}
	assert.Nil(t, query.Find(&systemPackages).Error)
	assert.Equal(t, nExpected, len(systemPackages))
}

func CheckSystemRepos(t *testing.T, rhAccountID int, systemID int64, repoIDs []int64) {
	var systemRepos []models.SystemRepo
	err := DB.Where("rh_account_id = ? AND system_id = ? AND repo_id IN (?)",
		rhAccountID, systemID, repoIDs).
		Find(&systemRepos).Error
	assert.Nil(t, err)
	assert.Equal(t, len(repoIDs), len(systemRepos))
}

func DeleteSystemAdvisories(t *testing.T, systemID int64, advisoryIDs []int64) {
	query := DB.Model(&models.SystemAdvisories{}).Where("system_id = ? AND advisory_id IN (?)",
		systemID, advisoryIDs)
	assert.Nil(t, query.Delete(&models.SystemAdvisories{}).Error)

	var cnt int64
	assert.Nil(t, query.Count(&cnt).Error)
	assert.Equal(t, int64(0), cnt)
	assert.Nil(t, DB.Exec("SELECT * FROM update_system_caches(?)", systemID).Error)
}

func CreateAccountAdvisory(t *testing.T, rhAccountID int, workspaceID string, advisoryIDs []int64,
	systemsInstallable int) {
	wsID := uuid.MustParse(workspaceID)
	for _, advisoryID := range advisoryIDs {
		err := DB.Create(&models.AccountAdvisory{
			AdvisoryID: advisoryID, RhAccountID: rhAccountID, WorkspaceID: wsID,
			SystemsInstallable: systemsInstallable, SystemsApplicable: systemsInstallable}).Error
		assert.Nil(t, err)
	}
	CheckAccountAdvisory(t, rhAccountID, workspaceID, advisoryIDs, systemsInstallable)
}

func CheckAccountAdvisory(t *testing.T, rhAccountID int, workspaceID string, advisoryIDs []int64,
	systemsInstallable int) {
	var accountAdvisory []models.AccountAdvisory
	err := DB.Where("rh_account_id = ? AND workspace_id = ? AND advisory_id IN (?)",
		rhAccountID, workspaceID, advisoryIDs).
		Find(&accountAdvisory).Error
	assert.Nil(t, err)

	sum := 0
	for _, item := range accountAdvisory {
		sum += item.SystemsInstallable
	}
	assert.Equal(t, systemsInstallable*len(advisoryIDs), sum, "sum of systems_installable does not match")
}

func DeleteAccountAdvisory(t *testing.T, rhAccountID int, workspaceID string, advisoryIDs []int64) {
	query := DB.Model(&models.AccountAdvisory{}).Where("rh_account_id = ? AND workspace_id = ? AND advisory_id IN (?)",
		rhAccountID, workspaceID, advisoryIDs)
	assert.Nil(t, query.Delete(&models.AccountAdvisory{}).Error)

	var cnt int64
	assert.Nil(t, query.Count(&cnt).Error)
	assert.Equal(t, int64(0), cnt)
}

func DeleteAccountAdvisoryByAccount(t *testing.T, rhAccountID int) {
	assert.Nil(t, DB.Where("rh_account_id = ?", rhAccountID).
		Delete(&models.AccountAdvisory{}).Error)
}

func DeleteAdvisoryAccountData(t *testing.T, rhAccountID int, advisoryIDs []int64) {
	query := DB.Model(&models.AdvisoryAccountData{}).Where("rh_account_id = ? AND advisory_id IN (?)",
		rhAccountID, advisoryIDs)
	assert.Nil(t, query.Delete(&models.AdvisoryAccountData{}).Error)

	var cnt int64
	assert.Nil(t, query.Count(&cnt).Error)
	assert.Equal(t, int64(0), cnt)
}

func DeleteSystemPackages(t *testing.T, accountID int, systemID int64, pkgIDs ...int64) {
	query := DB.Model(&models.SystemPackage{}).Where("rh_account_id = ? AND system_id = ?", accountID, systemID)
	if len(pkgIDs) > 0 {
		query = query.Where("package_id in (?)", pkgIDs)
	}
	assert.Nil(t, query.Delete(&models.SystemPackage{}).Error)

	var count int64 // ensure deleted
	assert.Nil(t, query.Count(&count).Error)
	assert.Equal(t, int64(0), count)
}

func DeleteSystemRepos(t *testing.T, rhAccountID int, systemID int64, repoIDs []int64) {
	err := DB.Where("rh_account_id = ? AND system_id = ? AND repo_id IN (?)", rhAccountID, systemID, repoIDs).
		Delete(&models.SystemRepo{}).Error
	assert.Nil(t, err)
}

func DeleteNewlyAddedPackages(t *testing.T) {
	query := DB.Table("package p").
		Where("id >= 100").
		Where("NOT EXISTS (SELECT 1 FROM system_package2 sp WHERE" +
			" p.id = sp.package_id OR p.id = sp.installable_id OR p.id = sp.applicable_id)")
	assert.Nil(t, query.Delete(models.Package{}).Error)
	var cnt int64
	assert.Nil(t, query.Count(&cnt).Error)
	assert.Equal(t, int64(0), cnt)
}

func DeleteNewlyAddedAdvisories(t *testing.T) {
	query := DB.Model(models.AdvisoryMetadata{}).Where("id >= 100")
	querySa := DB.Model(models.SystemAdvisories{}).Where("advisory_id >= 100")
	queryAad := DB.Model(models.AdvisoryAccountData{}).Where("advisory_id >= 100")
	assert.Nil(t, querySa.Delete(models.SystemAdvisories{}).Error)
	assert.Nil(t, queryAad.Delete(models.AdvisoryAccountData{}).Error)
	assert.Nil(t, query.Delete(models.AdvisoryMetadata{}).Error)
	var cnt int64
	assert.Nil(t, query.Count(&cnt).Error)
	assert.Equal(t, int64(0), cnt)
}

func GetAllSystems(t *testing.T) (systems []*models.SystemInventory) {
	assert.Nil(t, DB.Model(&models.SystemInventory{}).Order("rh_account_id").Scan(&systems).Error)
	return systems
}

func GetPackageIDs(nevras ...string) []int64 {
	ids := make([]int64, 0, len(nevras))
	for _, nevra := range nevras {
		var packageID int64
		nevra, err := utils.ParseNevra(nevra)
		if err != nil {
			continue
		}
		DB.Model(models.Package{}).
			Joins("JOIN package_name pn ON package.name_id = pn.id").
			Where("name = ? and evra = ?", nevra.Name, nevra.EVRAString()).
			Pluck("package.id", &packageID)
		if packageID != 0 {
			ids = append(ids, packageID)
		}
	}
	return ids
}

func CreateTemplate(t *testing.T, account int, uuid string, inventoryIDs []uuid.UUID) {
	template := &models.Template{
		TemplateBase: models.TemplateBase{
			RhAccountID: account, UUID: uuid, Name: uuid,
		},
		EnvironmentID: strings.ReplaceAll(uuid, "-", ""),
		Arch:          "x86_64",
		Version:       "8",
	}

	tx := DB.Begin()
	defer tx.Rollback()

	err := tx.Create(template).Error
	assert.Nil(t, err)

	for _, invID := range inventoryIDs {
		err = tx.Exec(`
		UPDATE system_patch SET template_id = ? 
		WHERE rh_account_id = ? 
		AND system_id = (
			SELECT id 
			FROM system_inventory 
			WHERE rh_account_id = ? 
			AND inventory_id = ?
		)`,
			template.ID, account, account, invID).Error
		assert.Nil(t, err)
	}
	err = tx.Commit().Error
	assert.Nil(t, err)
}

func DeleteTemplate(t *testing.T, account int, templateUUID string) {
	tx := DB.Begin()
	defer tx.Rollback()

	err := tx.Model(&models.SystemPatch{}).
		Where("rh_account_id = ? AND template_id = (SELECT id FROM template WHERE uuid = ?::uuid)", account, templateUUID).
		Updates(map[string]interface{}{"template_id": nil}).Error

	assert.Nil(t, err)

	err = tx.Delete(models.Template{}, "rh_account_id = ? AND uuid = ?::uuid", account, templateUUID).Error
	assert.Nil(t, err)

	err = tx.Commit().Error
	assert.Nil(t, err)
}

func CheckTemplateSystems(t *testing.T, account int, templateUUID string, inventoryIDs []uuid.UUID) {
	var foundInventoryIDs []uuid.UUID
	err := DB.Table("system_inventory si").Select("si.inventory_id as id").
		Joins("JOIN system_patch spatch ON si.id = spatch.system_id AND si.rh_account_id = spatch.rh_account_id").
		Joins("JOIN template tp ON tp.id = spatch.template_id AND tp.rh_account_id = spatch.rh_account_id").
		Where("si.rh_account_id = ? AND tp.uuid = ?::uuid", account, templateUUID).
		Order("id").
		Find(&foundInventoryIDs).Error

	assert.Nil(t, err)

	assert.Equal(t, len(inventoryIDs), len(foundInventoryIDs))
	if len(inventoryIDs) == len(foundInventoryIDs) {
		for index, inventoryID := range inventoryIDs {
			assert.Equal(t, inventoryID, foundInventoryIDs[index])
		}
	}
}
