package controllers

import (
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"app/manager/middlewares"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	lru "github.com/hashicorp/golang-lru"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

var enableAdvisoryDetailCache = utils.GetBoolEnvOrDefault("ENABLE_ADVISORY_DETAIL_CACHE", true)
var advisoryDetailCacheSize = utils.GetIntEnvOrDefault("ADVISORY_DETAIL_CACHE_SIZE", 100)
var advisoryDetailCacheV2 = initAdvisoryDetailCache()

const logProgressDuration = 2 * time.Second

type AdvisoryDetailResponseV1 struct {
	Data AdvisoryDetailItemV1 `json:"data"`
}

type AdvisoryDetailResponseV2 struct {
	Data AdvisoryDetailItemV2 `json:"data"`
}

type AdvisoryDetailItem struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type AdvisoryDetailItemV1 struct {
	AdvisoryDetailItem
	Attributes AdvisoryDetailAttributesV1 `json:"attributes"`
}

type AdvisoryDetailItemV2 struct {
	AdvisoryDetailItem
	Attributes AdvisoryDetailAttributesV2 `json:"attributes"`
}

type AdvisoryDetailAttributes struct {
	Description      string    `json:"description"`
	ModifiedDate     time.Time `json:"modified_date"`
	PublicDate       time.Time `json:"public_date"`
	Topic            string    `json:"topic"`
	Synopsis         string    `json:"synopsis"`
	Solution         *string   `json:"solution"`
	AdvisoryTypeName string    `json:"advisory_type_name"`
	Severity         *int      `json:"severity"`
	Fixes            *string   `json:"fixes"`
	Cves             []string  `json:"cves"`
	References       []string  `json:"references"`
	RebootRequired   bool      `json:"reboot_required"`
	ReleaseVersions  []string  `json:"release_versions"`
}

type AdvisoryDetailAttributesV1 struct {
	AdvisoryDetailAttributes
	Packages packagesV1 `json:"packages"`
}

type AdvisoryDetailAttributesV2 struct {
	AdvisoryDetailAttributes
	Packages packagesV2 `json:"packages"`
}

type packagesV1 map[string]string
type packagesV2 []string

// @Summary Show me details an advisory by given advisory name
// @Description Show me details an advisory by given advisory name
// @ID detailAdvisory
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    advisory_id    path    string   true "Advisory ID"
// @Success 200 {object} AdvisoryDetailResponseV2
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /advisories/{advisory_id} [get]
func AdvisoryDetailHandler(c *gin.Context) {
	advisoryName := c.Param("advisory_id")
	if advisoryName == "" {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{Error: "advisory_id param not found"})
		return
	}

	if !isFilterInURLValid(c) {
		return
	}

	var err error
	var respV2 *AdvisoryDetailResponseV2
	db := middlewares.DBFromContext(c)
	respV2, err = getAdvisoryV2(db, advisoryName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			LogAndRespNotFound(c, err, "advisory not found")
		} else {
			LogAndRespError(c, err, "advisory detail error")
		}
		return
	}

	if c.GetInt(middlewares.KeyApiver) < 2 {
		respV1 := advisoryRespV2toV1(respV2)
		c.JSON(http.StatusOK, respV1)
	} else {
		c.JSON(http.StatusOK, respV2)
	}
}

func getAdvisoryFromDB(db *gorm.DB, advisoryName string) (*models.AdvisoryMetadata, *AdvisoryDetailAttributes, error) {
	var advisory models.AdvisoryMetadata
	err := db.Table(advisory.TableName()).
		Take(&advisory, "name = ?", advisoryName).Error
	if err != nil {
		return nil, nil, err
	}

	cves, err := parseJSONList(advisory.CveList)
	if err != nil {
		return nil, nil, errors.Wrap(err, "CVEs parsing error")
	}

	releaseVersions, err := parseJSONList(advisory.ReleaseVersions)
	if err != nil {
		return nil, nil, errors.Wrap(err, "release_versions parsing error")
	}

	ada := AdvisoryDetailAttributes{
		Description:      advisory.Description,
		ModifiedDate:     advisory.ModifiedDate,
		PublicDate:       advisory.PublicDate,
		Topic:            advisory.Summary,
		Synopsis:         advisory.Synopsis,
		Solution:         advisory.Solution,
		Severity:         advisory.SeverityID,
		AdvisoryTypeName: database.AdvisoryTypes[advisory.AdvisoryTypeID],
		Fixes:            nil,
		Cves:             cves,
		References:       []string{},
		RebootRequired:   advisory.RebootRequired,
		ReleaseVersions:  releaseVersions,
	}
	return &advisory, &ada, err
}

func advisoryRespV2toV1(respV2 *AdvisoryDetailResponseV2) *AdvisoryDetailResponseV1 {
	pkgsV1 := pkgsV2topkgsV1(respV2.Data.Attributes.Packages)
	respV1 := AdvisoryDetailResponseV1{
		Data: AdvisoryDetailItemV1{
			AdvisoryDetailItem: AdvisoryDetailItem{
				ID:   respV2.Data.ID,
				Type: "advisory",
			},
			Attributes: AdvisoryDetailAttributesV1{
				AdvisoryDetailAttributes: respV2.Data.Attributes.AdvisoryDetailAttributes,
				Packages:                 pkgsV1,
			},
		}}
	return &respV1
}

func getAdvisoryFromDBV2(db *gorm.DB, advisoryName string) (*AdvisoryDetailResponseV2, error) {
	advisory, ada, err := getAdvisoryFromDB(db, advisoryName)
	if err != nil {
		return nil, err
	}

	pkgs, err := parsePackages(advisory.PackageData)
	if err != nil {
		return nil, errors.Wrap(err, "packages parsing error")
	}

	var resp = AdvisoryDetailResponseV2{Data: AdvisoryDetailItemV2{
		AdvisoryDetailItem: AdvisoryDetailItem{ID: advisory.Name, Type: "advisory"},
		Attributes: AdvisoryDetailAttributesV2{
			AdvisoryDetailAttributes: *ada,
			Packages:                 pkgs,
		},
	}}
	return &resp, nil
}

func parsePackages(jsonb []byte) (packagesV2, error) {
	if jsonb == nil {
		return packagesV2{}, nil
	}

	var err error
	pkgsV2, err := parseJSONList(jsonb)
	if err != nil {
		// HACK!
		// Until vmaas-sync syncs new data, `jsonb` has '{"<name>": "<evra>"}' format
		// what we need for V2 api is ["<name>-<evra>", ...]
		// 1. try to unmarshal to packagesV1 struct
		var tmpPkgV1 packagesV1
		if v1err := json.Unmarshal(jsonb, &tmpPkgV1); v1err != nil {
			// cannot unmarshal to neither V1 nor V2
			return nil, err
		}
		// 2. create `packagesV2` from `packagesV1` data
		for k, v := range tmpPkgV1 {
			// NOTE: V2 now shows the same data as V1 api until vmaas is synced
			pkgsV2 = append(pkgsV2, fmt.Sprintf("%s-%s", k, v))
		}
	}
	return pkgsV2, nil
}

func pkgsV2topkgsV1(pkgsV2 packagesV2) packagesV1 {
	// assigning first pkg to packages map in api/v1
	// it shows incorrect packages info
	// but we need to maintain backward compatibility
	pkgsV1 := make(packagesV1)
	for _, pkg := range pkgsV2 {
		nevra, err := utils.ParseNevra(pkg)
		if err != nil {
			continue
		}
		if _, has := pkgsV1[nevra.Name]; !has {
			pkgsV1[nevra.Name] = nevra.EVRAString()
		}
	}
	return pkgsV1
}

func initAdvisoryDetailCache() *lru.Cache {
	middlewares.AdvisoryDetailGauge.Set(0)
	if !enableAdvisoryDetailCache {
		return nil
	}

	cache, err := lru.New(advisoryDetailCacheSize)
	if err != nil {
		panic(err)
	}

	return cache
}

func PreloadAdvisoryCacheItems() {
	preLoadCache := utils.GetBoolEnvOrDefault("PRELOAD_ADVISORY_DETAIL_CACHE", true)
	if !preLoadCache {
		return
	}

	utils.LogInfo("cacheSize", advisoryDetailCacheSize, "loading items to advisory detail cache...")
	var advisoryNames []string
	err := database.Db.Table("advisory_metadata").Limit(advisoryDetailCacheSize).Order("public_date DESC").
		Pluck("name", &advisoryNames).Error // preload first N most recent advisories to cache
	if err != nil {
		panic(err)
	}

	progress, count := utils.LogProgress("Advisory detail cache preload", logProgressDuration, int64(len(advisoryNames)))

	for _, advisoryName := range advisoryNames {
		_, err = getAdvisoryV2(database.Db, advisoryName)
		if err != nil {
			utils.LogError("advisoryName", advisoryName, "err", err.Error(), "can not re-load item to cache - V2")
		}
		*count++
	}
	progress.Stop()
}

func tryGetAdvisoryFromCacheV2(advisoryName string) *AdvisoryDetailResponseV2 {
	if advisoryDetailCacheV2 == nil {
		return nil
	}

	val, ok := advisoryDetailCacheV2.Get(advisoryName)
	if !ok {
		middlewares.AdvisoryDetailCnt.WithLabelValues("miss").Inc()
		return nil
	}
	resp := val.(AdvisoryDetailResponseV2)
	middlewares.AdvisoryDetailCnt.WithLabelValues("hit").Inc()
	return &resp
}

func tryAddAdvisoryToCacheV2(advisoryName string, resp *AdvisoryDetailResponseV2) {
	if advisoryDetailCacheV2 == nil {
		return
	}
	evicted := advisoryDetailCacheV2.Add(advisoryName, *resp)
	middlewares.AdvisoryDetailGauge.Inc()
	utils.LogDebug("evictedV2", evicted, "advisoryName", advisoryName, "saved to cache")
}

func getAdvisoryV2(db *gorm.DB, advisoryName string) (*AdvisoryDetailResponseV2, error) {
	resp := tryGetAdvisoryFromCacheV2(advisoryName)
	if resp != nil {
		utils.LogDebug("advisoryName", advisoryName, "found in cache")
		return resp, nil // return data found in cache
	}

	resp, err := getAdvisoryFromDBV2(db, advisoryName) // search for data in database
	if err != nil {
		return nil, err
	}

	tryAddAdvisoryToCacheV2(advisoryName, resp) // save data to cache if initialized
	return resp, nil
}
