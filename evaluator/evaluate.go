package evaluator

import (
	"app/base"
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/mqueue"
	"app/base/utils"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/RedHatInsights/patchman-clients/vmaas"
	"github.com/antihax/optional"
	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/dialects/postgres"
	"github.com/lestrrat-go/backoff"
	"github.com/pkg/errors"
	"net/http"
	"sync"
	"time"
)

type SystemAdvisoryMap map[string]models.SystemAdvisories

var (
	consumerCount          int
	vmaasClient            *vmaas.APIClient
	evalTopic              string
	evalLabel              string
	port                   string
	enableAdvisoryAnalysis bool
	enablePackageAnalysis  bool
	enableBypass           bool
	enableStaleSysEval     bool
)

func configure() {
	core.ConfigureApp()
	port = utils.GetenvOrFail("PORT")
	traceAPI := utils.GetenvOrFail("LOG_LEVEL") == "trace"

	evalTopic = utils.GetenvOrFail("EVAL_TOPIC")
	evalLabel = utils.GetenvOrFail("EVAL_LABEL")
	consumerCount = utils.GetIntEnvOrFail("CONSUMER_COUNT")

	vmaasConfig := vmaas.NewConfiguration()
	vmaasConfig.BasePath = utils.GetenvOrFail("VMAAS_ADDRESS") + base.VMaaSAPIPrefix
	vmaasConfig.Debug = traceAPI
	disableCompression := !utils.GetBoolEnvOrDefault("ENABLE_VMAAS_CALL_COMPRESSION", true)
	enableAdvisoryAnalysis = utils.GetBoolEnvOrDefault("ENABLE_ADVISORY_ANALYSIS", true)
	enablePackageAnalysis = utils.GetBoolEnvOrDefault("ENABLE_PACKAGE_ANALYSIS", true)
	enableStaleSysEval = utils.GetBoolEnvOrDefault("ENABLE_STALE_SYSTEM_EVALUATION", true)
	enableBypass = utils.GetBoolEnvOrDefault("ENABLE_BYPASS", false)
	vmaasConfig.HTTPClient = &http.Client{Transport: &http.Transport{
		DisableCompression: disableCompression,
	}}
	vmaasClient = vmaas.NewAPIClient(vmaasConfig)
}

// nolint: funlen
func Evaluate(ctx context.Context, accountID int, inventoryID string, requested *base.Rfc3339Timestamp,
	evaluationType string) error {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, evaluationDuration.WithLabelValues(evaluationType))
	if enableBypass {
		evaluationCnt.WithLabelValues("bypassed").Inc()
		utils.Log("inventoryID", inventoryID).Info("Evaluation bypassed")
		return nil
	}

	tx := database.Db.BeginTx(base.Context, nil)
	// Don'requested allow TX to hang around locking the rows
	defer tx.RollbackUnlessCommitted()

	system, err := loadSystemData(tx, accountID, inventoryID)
	if err != nil {
		evaluationCnt.WithLabelValues("error-db-read-inventory-data").Inc()
		return nil
	}

	if system.Stale && !enableStaleSysEval {
		evaluationCnt.WithLabelValues("skipping-stale").Inc()
		return nil
	}

	if requested != nil && system.LastEvaluation != nil && requested.Time().Before(*system.LastEvaluation) {
		evaluationCnt.WithLabelValues("error-old-msg").Inc()
		return nil
	}

	updatesReq, err := parseVmaasJSON(system)
	if err != nil {
		evaluationCnt.WithLabelValues("error-parse-vmaas-json").Inc()
		return errors.Wrap(err, "Unable to parse system vmaas json")
	}

	if len(updatesReq.PackageList) == 0 {
		evaluationCnt.WithLabelValues("error-no-packages").Inc()
		return nil
	}

	vmaasData, err := callVMaas(ctx, updatesReq)
	if err != nil {
		evaluationCnt.WithLabelValues("error-call-vmaas-updates").Inc()
		return errors.Wrap(err, "vmaas API call failed")
	}

	err = evaluateAndStore(tx, system, *vmaasData)
	if err != nil {
		return errors.Wrap(err, "Unable to evaluate and store results")
	}

	err = commitWithObserve(tx)
	if err != nil {
		evaluationCnt.WithLabelValues("error-database-commit").Inc()
		return errors.New("database commit failed")
	}

	err = publishRemediationsState(system.InventoryID, *vmaasData)
	if err != nil {
		evaluationCnt.WithLabelValues("error-remediations-publish").Inc()
		return errors.Wrap(err, "remediations publish failed")
	}

	evaluationCnt.WithLabelValues("success").Inc()
	return nil
}

func commitWithObserve(tx *gorm.DB) error {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, evaluationPartDuration.WithLabelValues("commit-to-db"))

	err := tx.Commit().Error
	if err != nil {
		return err
	}
	return nil
}

func evaluateAndStore(tx *gorm.DB, system *models.SystemPlatform, vmaasData vmaas.UpdatesV2Response) error {
	oldSystemAdvisories, newSystemAdvisories, err := analyzeAdvisories(tx, system, vmaasData)
	if err != nil {
		return errors.Wrap(err, "Advisory analysis failed")
	}

	installed, updatable, err := analyzePackages(tx, system, vmaasData)
	if err != nil {
		return errors.Wrap(err, "Package analysis failed")
	}

	err = updateSystemPlatform(tx, system, oldSystemAdvisories, newSystemAdvisories, installed, updatable)
	if err != nil {
		evaluationCnt.WithLabelValues("error-update-system").Inc()
		return errors.Wrap(err, "Unable to update system")
	}
	return nil
}

func analyzeAdvisories(tx *gorm.DB, system *models.SystemPlatform, vmaasData vmaas.UpdatesV2Response) (
	SystemAdvisoryMap, SystemAdvisoryMap, error) {
	if !enableAdvisoryAnalysis {
		utils.Log().Debug("advisory analysis disabled, skipping")
		return nil, nil, nil
	}

	oldSystemAdvisories, patched, unpatched, err := processSystemAdvisories(tx, system, vmaasData, system.InventoryID)
	if err != nil {
		evaluationCnt.WithLabelValues("error-process-advisories").Inc()
		return nil, nil, errors.Wrap(err, "Unable to process system advisories")
	}

	newSystemAdvisories, err := storeAdvisoryData(tx, system, patched, unpatched)
	if err != nil {
		evaluationCnt.WithLabelValues("error-store-advisories").Inc()
		return nil, nil, errors.Wrap(err, "Unable to store advisory data")
	}
	return oldSystemAdvisories, newSystemAdvisories, nil
}

func analyzePackages(tx *gorm.DB, system *models.SystemPlatform, vmaasData vmaas.UpdatesV2Response) (
	installed int, updatable int, err error) {
	if !enablePackageAnalysis {
		utils.Log().Debug("pkg analysis disabled, skipping")
		return 0, 0, nil
	}

	pkgByName, err := loadPackages(tx, system.RhAccountID, system.ID, vmaasData)
	if err != nil {
		evaluationCnt.WithLabelValues("error-pkg-data").Inc()
		return 0, 0, errors.Wrap(err, "Unable to load package data")
	}
	installed, updatable, err = updateSystemPackages(tx, system, pkgByName, vmaasData.UpdateList)
	if err != nil {
		evaluationCnt.WithLabelValues("error-system-pkgs").Inc()
		return 0, 0, errors.Wrap(err, "Unable to update system packages")
	}
	return installed, updatable, nil
}

func deleteOldSystemPackages(tx *gorm.DB, accountID, systemID int, packagesByNEVRA map[utils.Nevra]namedPackage) error {
	pkgIds := make([]int, 0, len(packagesByNEVRA))
	for _, pkg := range packagesByNEVRA {
		pkgIds = append(pkgIds, pkg.PackageID)
	}
	return errors.Wrap(tx.
		Where("rh_account_id = ? ", accountID).
		Where("system_id = ?", systemID).
		Where("package_id not in (?)", pkgIds).
		Delete(&models.SystemPackage{}).Error, "Deleting outdated system packages")
}

// nolint: funlen
func updateSystemPackages(tx *gorm.DB, system *models.SystemPlatform,
	packagesByNEVRA map[utils.Nevra]namedPackage,
	updates map[string]vmaas.UpdatesV2ResponseUpdateList) (int, int, error) {
	defer utils.ObserveSecondsSince(time.Now(), evaluationPartDuration.WithLabelValues("packages-store"))
	var installed, updatable int

	if err := deleteOldSystemPackages(tx, system.RhAccountID, system.ID, packagesByNEVRA); err != nil {
		return 0, 0, err
	}

	toStore := make([]models.SystemPackage, 0, len(updates))
	for nevraStr, updateData := range updates {
		var pkgUpdates []models.PackageUpdate
		// Parse each NEVRA in the input
		nevra, err := utils.ParseNevra(nevraStr)
		if err != nil {
			utils.Log("nevra", nevraStr).Warn("Invalid nevra")
			continue
		}
		currentNamedPackage := packagesByNEVRA[*nevra]
		// Check whether we have that NEVRA in DB
		if currentNamedPackage.PackageID == 0 {
			utils.Log("nevra", nevraStr).Trace("Unknown package")
			continue
		}
		installed++
		if len(updateData.AvailableUpdates) > 0 {
			updatable++
		}

		for _, upData := range updateData.AvailableUpdates {
			upNevra, err := utils.ParseNevra(upData.Package)
			// Skip invalid nevras in updates list
			if err != nil {
				utils.Log("nevra", upData.Package).Warn("Invalid nevra")
				continue
			}
			// Create correct entry for each update in the list
			pkgUpdates = append(pkgUpdates, models.PackageUpdate{
				EVRA:     upNevra.EVRAString(),
				Advisory: upData.Erratum,
			})
		}
		var updateDataJSON []byte
		if len(pkgUpdates) > 0 {
			updateDataJSON, err = json.Marshal(pkgUpdates)
			if err != nil {
				return 0, 0, errors.Wrap(err, "Serializing pkg json")
			}
		}

		// Skip overwriting entries which have the same data as before
		if bytes.Equal(updateDataJSON, currentNamedPackage.UpdateData.RawMessage) {
			// If the update_data we want to store is null, we skip only only if there was a row for this specific
			// system_package already stored.
			// If its not null, then the previous check ensured that the old update data matches new one
			if (updateDataJSON == nil && currentNamedPackage.WasStored) || updateDataJSON != nil {
				continue
			}
		}

		// Create row to update
		toStore = append(toStore, models.SystemPackage{
			RhAccountID: system.RhAccountID,
			SystemID:    system.ID,
			PackageID:   packagesByNEVRA[*nevra].PackageID,
			UpdateData:  postgres.Jsonb{RawMessage: updateDataJSON},
		})
	}
	tx = database.OnConflictUpdateMulti(tx, []string{"rh_account_id", "system_id", "package_id"}, "update_data")
	return installed, updatable, errors.Wrap(database.BulkInsert(tx, toStore), "Storing system packages")
}

type namedPackage struct {
	Name       string
	PackageID  int
	EVRA       string
	WasStored  bool
	UpdateData postgres.Jsonb
}

// nolint: lll
// Find relevant package data based on vmaas results
func loadPackages(tx *gorm.DB, accountID, systemID int,
	data vmaas.UpdatesV2Response) (map[utils.Nevra]namedPackage, error) {
	names := make([]string, 0, len(data.UpdateList))
	evras := make([]string, 0, len(data.UpdateList))

	defer utils.ObserveSecondsSince(time.Now(), evaluationPartDuration.WithLabelValues("packages-load"))

	for nevra := range data.UpdateList {
		// Parse and reformat nevras to avoid issues with 0 epoch
		parsed, err := utils.ParseNevra(nevra)
		if err != nil {
			utils.Log("err", err.Error(), "nevra", nevra).Warn("Unable to parse nevra")
			continue
		}
		names = append(names, parsed.Name)
		evras = append(evras, parsed.EVRAString())
	}

	// Might return more data than we need (one EVRA being applicable to more packages)
	// But it was only way to get somewhat fast query plan which only uses index scans
	var packages []namedPackage
	err := tx.Table("package").
		// We need to have data about the package, and what data we had stored in relation to this system.
		Select("pn.name, package.id as package_id, package.evra, (sp.system_id IS NOT NULL) as was_stored, sp.update_data").
		Joins("join package_name pn on package.name_id = pn.id").
		// We need to perform left join, so thats why the parameters are here
		Joins(`left join system_package sp on sp.package_id = package.id AND sp.rh_account_id = ? AND sp.system_id = ?`, accountID, systemID).
		Where("pn.name in (?)", names).
		Where("package.evra in (?)", evras).Find(&packages).Error

	if err != nil {
		return nil, errors.Wrap(err, "loading packages")
	}

	pkgByNevra := map[utils.Nevra]namedPackage{}
	for _, p := range packages {
		nevra, err := utils.ParseNameEVRA(p.Name, p.EVRA)
		if err != nil {
			utils.Log("err", err.Error(), "name", p.Name, "evra", p.EVRA).Warn("Unable to parse nevra")
			continue
		}
		pkgByNevra[*nevra] = p
	}

	return pkgByNevra, nil
}

func updateSystemPlatform(tx *gorm.DB, system *models.SystemPlatform,
	old, new SystemAdvisoryMap, installed, updatable int) error {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, evaluationPartDuration.WithLabelValues("system-update"))
	defer utils.ObserveSecondsSince(*system.LastUpload, uploadEvaluationDelay)
	if system.LastEvaluation != nil {
		defer utils.ObserveHoursSince(*system.LastEvaluation, twoEvaluationsInterval)
	}

	data := map[string]interface{}{}
	data["last_evaluation"] = time.Now()

	if enableAdvisoryAnalysis {
		if old == nil || new == nil {
			return errors.New("Invalid args")
		}
		for name, newSysAdvisory := range new {
			old[name] = newSysAdvisory
		}
		counts := make([]int, 4)
		for _, sa := range old {
			if sa.WhenPatched == nil && sa.Advisory.AdvisoryTypeID > 0 {
				counts[sa.Advisory.AdvisoryTypeID]++
			}
			counts[0]++
		}
		data["advisory_count_cache"] = counts[0]
		data["advisory_enh_count_cache"] = counts[1]
		data["advisory_bug_count_cache"] = counts[2]
		data["advisory_sec_count_cache"] = counts[3]
	}

	if enablePackageAnalysis {
		data["packages_installed"] = installed
		data["packages_updatable"] = updatable
	}
	return tx.Model(system).Update(data).Error
}

// nolint: bodyclose
func callVMaas(ctx context.Context, request vmaas.UpdatesV3Request) (*vmaas.UpdatesV2Response, error) {
	var policy = backoff.NewExponential(
		backoff.WithInterval(time.Second),
		backoff.WithMaxRetries(8),
	)
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, evaluationPartDuration.WithLabelValues("vmaas-updates-call"))

	vmaasCallArgs := vmaas.AppUpdatesHandlerV3PostPostOpts{
		UpdatesV3Request: optional.NewInterface(request),
	}
	backoffState, cancel := policy.Start(base.Context)
	defer cancel()
	for backoff.Continue(backoffState) {
		vmaasData, resp, err := vmaasClient.DefaultApi.AppUpdatesHandlerV3PostPost(ctx, &vmaasCallArgs)

		// VMaaS is probably refreshing caches, continue waiting
		if resp != nil && resp.StatusCode == http.StatusServiceUnavailable {
			continue
		}

		if err != nil {
			responseDetails := utils.TryGetResponseDetails(resp)
			return nil, errors.Wrap(err, "vmaas API call failed"+responseDetails+fmt.Sprintf(
				", (packages: %d, basearch: %s, modules: %d, releasever: %s, repolist: %d)",
				len(request.PackageList), request.Basearch, len(request.ModulesList), request.Releasever,
				len(request.RepositoryList)))
		}
		return &vmaasData, nil
	}
	return nil, errors.New("VMaaS is unavailable")
}

func loadSystemData(tx *gorm.DB, accountID int, inventoryID string) (*models.SystemPlatform, error) {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, evaluationPartDuration.WithLabelValues("data-loading"))

	var system models.SystemPlatform
	err := tx.Set("gorm:query_option", "FOR UPDATE OF system_platform").
		Where("rh_account_id = ?", accountID).
		Where("inventory_id = ?::uuid", inventoryID).Find(&system).Error
	return &system, err
}

func parseVmaasJSON(system *models.SystemPlatform) (vmaas.UpdatesV3Request, error) {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, evaluationPartDuration.WithLabelValues("parse-vmaas-json"))

	var updatesReq vmaas.UpdatesV3Request
	err := json.Unmarshal([]byte(system.VmaasJSON), &updatesReq)
	return updatesReq, err
}

// Changes data stored in system_advisories, in order to match newest evaluation
// Before this methods stores the entries into the system_advisories table, it locks
// advisory_account_data table, so other evaluations don't interfere with this one
func processSystemAdvisories(tx *gorm.DB, system *models.SystemPlatform, vmaasData vmaas.UpdatesV2Response,
	inventoryID string) (oldSystemAdvisories SystemAdvisoryMap, patched []int, unpatched []int, err error) {
	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, evaluationPartDuration.WithLabelValues("advisories-processing"))

	reported := getReportedAdvisories(vmaasData)
	oldSystemAdvisories, err = getStoredAdvisoriesMap(tx, system.RhAccountID, system.ID)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "Unable to get system stored advisories")
	}

	patched = getPatchedAdvisories(reported, oldSystemAdvisories)
	updatesCnt.WithLabelValues("patched").Add(float64(len(patched)))
	utils.Log("inventoryID", inventoryID, "patched", len(patched)).Debug("patched advisories")

	newsAdvisoriesNames, unpatched := getNewAndUnpatchedAdvisories(reported, oldSystemAdvisories)
	utils.Log("inventoryID", inventoryID, "newAdvisories", len(newsAdvisoriesNames)).Debug("new advisories")

	newIDs, err := getAdvisoriesFromDB(tx, newsAdvisoriesNames)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "Unable to ensure new system advisories in db")
	}
	nUnknown := len(newsAdvisoriesNames) - len(newIDs)
	if nUnknown > 0 {
		utils.Log("inventoryID", inventoryID, "unknown", nUnknown).Debug("unknown advisories - ignored")
		updatesCnt.WithLabelValues("unknown").Add(float64(nUnknown))
	}

	unpatched = append(unpatched, newIDs...)
	updatesCnt.WithLabelValues("unpatched").Add(float64(len(unpatched)))
	utils.Log("inventoryID", inventoryID, "unpatched", len(unpatched)).Debug("patched advisories")
	return oldSystemAdvisories, patched, unpatched, nil
}

func storeAdvisoryData(tx *gorm.DB, system *models.SystemPlatform,
	patched, unpatched []int) (SystemAdvisoryMap, error) {
	defer utils.ObserveSecondsSince(time.Now(), evaluationPartDuration.WithLabelValues("advisories-store"))
	newSystemAdvisories, err := updateSystemAdvisories(tx, system, patched, unpatched)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to update system advisories")
	}

	err = updateAdvisoryAccountDatas(tx, system, patched, unpatched)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to update advisory_account_data caches")
	}
	return newSystemAdvisories, nil
}

func getReportedAdvisories(vmaasData vmaas.UpdatesV2Response) map[string]bool {
	advisories := map[string]bool{}
	for _, updates := range vmaasData.UpdateList {
		for _, u := range updates.AvailableUpdates {
			advisories[u.Erratum] = true
		}
	}
	return advisories
}

func getReportedPackageUpdates(vmaasData vmaas.UpdatesV2Response) map[string]bool {
	packages := map[string]bool{}
	for _, updates := range vmaasData.UpdateList {
		for _, u := range updates.AvailableUpdates {
			packages[u.Package] = true
		}
	}
	return packages
}

func getStoredAdvisoriesMap(tx *gorm.DB, accountID, systemID int) (map[string]models.SystemAdvisories, error) {
	var advisories []models.SystemAdvisories
	err := database.SystemAdvisoriesQueryByID(tx, accountID, systemID).Preload("Advisory").Find(&advisories).Error
	if err != nil {
		return nil, err
	}

	advisoriesMap := map[string]models.SystemAdvisories{}
	for _, advisory := range advisories {
		advisoriesMap[advisory.Advisory.Name] = advisory
	}
	return advisoriesMap, nil
}

func getNewAndUnpatchedAdvisories(reported map[string]bool, stored map[string]models.SystemAdvisories) (
	[]string, []int) {
	newAdvisories := []string{}
	unpatchedAdvisories := []int{}
	for reportedAdvisory := range reported {
		if storedAdvisory, found := stored[reportedAdvisory]; found {
			if storedAdvisory.WhenPatched != nil { // this advisory was already patched and now is un-patched again
				unpatchedAdvisories = append(unpatchedAdvisories, storedAdvisory.AdvisoryID)
			}
		} else {
			newAdvisories = append(newAdvisories, reportedAdvisory)
		}
	}
	return newAdvisories, unpatchedAdvisories
}

func getPatchedAdvisories(reported map[string]bool, stored map[string]models.SystemAdvisories) []int {
	patchedAdvisories := make([]int, 0, len(stored))
	for storedAdvisory, storedAdvisoryObj := range stored {
		if _, found := reported[storedAdvisory]; found {
			continue
		}

		// advisory contained in reported - it's patched
		if storedAdvisoryObj.WhenPatched != nil {
			// it's already marked as patched
			continue
		}

		// advisory was patched from last evaluation, let's mark it as patched
		patchedAdvisories = append(patchedAdvisories, storedAdvisoryObj.AdvisoryID)
	}
	return patchedAdvisories
}

func updateSystemAdvisoriesWhenPatched(tx *gorm.DB, system *models.SystemPlatform, advisoryIDs []int,
	whenPatched *time.Time) error {
	err := tx.Model(models.SystemAdvisories{}).
		Where("system_id = ? AND rh_account_id = ? AND advisory_id IN (?)",
			system.ID, system.RhAccountID, advisoryIDs).
		Update("when_patched", whenPatched).Error
	if err != nil {
		return err
	}
	return nil
}

// Return advisory IDs, created advisories count, error
func getAdvisoriesFromDB(tx *gorm.DB, advisories []string) ([]int, error) {
	var advisoryIDs []int
	err := tx.Model(&models.AdvisoryMetadata{}).Where("name IN (?)", advisories).
		Pluck("id", &advisoryIDs).Error
	if err != nil {
		return nil, err
	}
	return advisoryIDs, nil
}

func ensureSystemAdvisories(tx *gorm.DB, rhAccountID int, systemID int, advisoryIDs []int) error {
	advisoriesObjs := models.SystemAdvisoriesSlice{}
	for _, advisoryID := range advisoryIDs {
		advisoriesObjs = append(advisoriesObjs,
			models.SystemAdvisories{RhAccountID: rhAccountID, SystemID: systemID, AdvisoryID: advisoryID})
	}

	txOnConflict := tx.Set("gorm:insert_option", "ON CONFLICT DO NOTHING")
	err := database.BulkInsert(txOnConflict, advisoriesObjs)
	return err
}

func lockAdvisoryAccountData(tx *gorm.DB, system *models.SystemPlatform, patched, unpatched []int) error {
	// Lock advisory-account data, so it's not changed by other concurrent queries
	var aads []models.AdvisoryAccountData
	err := tx.Set("gorm:query_option", "FOR UPDATE OF advisory_account_data").
		Order("advisory_id").
		Find(&aads, "rh_account_id = ? AND (advisory_id in (?) OR advisory_id in (?))",
			system.RhAccountID, patched, unpatched).Error

	return err
}

func calcAdvisoryChanges(system *models.SystemPlatform, patched, unpatched []int) []models.AdvisoryAccountData {
	aadMap := map[int]models.AdvisoryAccountData{}
	// If system is stale, we won't change any rows  in advisory_account_data
	if system.Stale {
		return []models.AdvisoryAccountData{}
	}

	for _, id := range unpatched {
		aadMap[id] = models.AdvisoryAccountData{
			AdvisoryID:      id,
			RhAccountID:     system.RhAccountID,
			SystemsAffected: 1,
		}
	}

	for _, id := range patched {
		aadMap[id] = models.AdvisoryAccountData{
			AdvisoryID:      id,
			RhAccountID:     system.RhAccountID,
			SystemsAffected: -1,
		}
	}

	deltas := make([]models.AdvisoryAccountData, 0, len(patched)+len(unpatched))
	for _, aad := range aadMap {
		deltas = append(deltas, aad)
	}
	return deltas
}

func updateSystemAdvisories(tx *gorm.DB, system *models.SystemPlatform,
	patched, unpatched []int) (SystemAdvisoryMap, error) {
	whenPatched := time.Now()

	err := ensureSystemAdvisories(tx, system.RhAccountID, system.ID, unpatched)
	if err != nil {
		return nil, err
	}

	err = updateSystemAdvisoriesWhenPatched(tx, system, patched, &whenPatched)
	if err != nil {
		return nil, err
	}

	err = updateSystemAdvisoriesWhenPatched(tx, system, unpatched, nil)
	if err != nil {
		return nil, err
	}

	newSystemAdvisories := SystemAdvisoryMap{}
	var data []models.SystemAdvisories
	err = tx.Preload("Advisory").
		Find(&data, "system_id = ? AND rh_account_id = ? AND (advisory_id IN (?) OR advisory_id in (?))",
			system.ID, system.RhAccountID, unpatched, patched).Error

	if err != nil {
		return nil, err
	}

	for _, sa := range data {
		newSystemAdvisories[sa.Advisory.Name] = sa
	}
	return newSystemAdvisories, nil
}

func updateAdvisoryAccountDatas(tx *gorm.DB, system *models.SystemPlatform, patched, unpatched []int) error {
	err := lockAdvisoryAccountData(tx, system, patched, unpatched)
	if err != nil {
		return err
	}

	changes := calcAdvisoryChanges(system, patched, unpatched)
	txOnConflict := database.OnConflictDoUpdateExpr(tx, []string{"rh_account_id", "advisory_id"},
		database.UpExpr{Name: "systems_affected", Expr: "advisory_account_data.systems_affected + excluded.systems_affected"})

	return database.BulkInsert(txOnConflict, changes)
}

func evaluateHandler(event mqueue.PlatformEvent) error {
	err := Evaluate(base.Context, event.AccountID, event.ID, event.Timestamp, evalLabel)
	if err != nil {
		utils.Log("err", err.Error(), "inventoryID", event.ID, "evalLabel", evalLabel).
			Error("Eval message handling")
		return err
	}
	utils.Log("inventoryID", event.ID, "evalLabel", evalLabel).Debug("system evaluated successfully")
	return nil
}

func run(wg *sync.WaitGroup, readerBuilder mqueue.CreateReader) {
	utils.Log().Info("evaluator starting")
	configure()

	go RunMetrics(port)

	var handler = mqueue.MakeRetryingHandler(mqueue.MakeMessageHandler(evaluateHandler))
	// We create multiple consumers, and hope that the partition rebalancing
	// algorithm assigns each consumer a single partition
	for i := 0; i < consumerCount; i++ {
		mqueue.SpawnReader(wg, evalTopic, readerBuilder, handler)
	}
}

func RunEvaluator() {
	var wg sync.WaitGroup
	run(&wg, mqueue.ReaderFromEnv)
	wg.Wait()
	utils.Log().Info("evaluator completed")
}
