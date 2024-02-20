package controllers

import (
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"app/manager/middlewares"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

var enableAdvisoryDetailCache = utils.GetBoolEnvOrDefault("ENABLE_ADVISORY_DETAIL_CACHE", true)
var advisoryDetailCacheSize = utils.GetIntEnvOrDefault("ADVISORY_DETAIL_CACHE_SIZE", 100)
var advisoryDetailCache = initAdvisoryDetailCache()

const logProgressDuration = 2 * time.Second

type AdvisoryDetailResponse struct {
	Data AdvisoryDetailItem `json:"data"`
}

type AdvisoryDetailItem struct {
	ID         string                   `json:"id"`
	Type       string                   `json:"type"`
	Attributes AdvisoryDetailAttributes `json:"attributes"`
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
	Packages         packages  `json:"packages"`
}

type packages []string

// @Summary Show me details an advisory by given advisory name
// @Description Show me details an advisory by given advisory name
// @ID detailAdvisory
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    advisory_id    path    string   true "Advisory ID"
// @Success 200 {object} AdvisoryDetailResponse
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
	var resp *AdvisoryDetailResponse
	db := middlewares.DBFromContext(c)
	resp, err = getAdvisory(db, advisoryName, false)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			LogAndRespNotFound(c, err, "advisory not found")
		} else {
			LogAndRespError(c, err, "advisory detail error")
		}
		return
	}

	c.JSON(http.StatusOK, resp)
}

func getAdvisoryFromDB(db *gorm.DB, advisoryName string) (*AdvisoryDetailResponse, error) {
	var advisory models.AdvisoryMetadata
	err := db.Table(advisory.TableName()).
		Take(&advisory, "name = ?", advisoryName).Error
	if err != nil {
		return nil, err
	}

	cves, err := parseJSONList(advisory.CveList)
	if err != nil {
		return nil, errors.Wrap(err, "CVEs parsing error")
	}

	releaseVersions, err := parseJSONList(advisory.ReleaseVersions)
	if err != nil {
		return nil, errors.Wrap(err, "release_versions parsing error")
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

	pkgs, err := parsePackages(advisory.PackageData)
	if err != nil {
		return nil, errors.Wrap(err, "packages parsing error")
	}

	var resp = AdvisoryDetailResponse{Data: AdvisoryDetailItem{
		ID:         advisory.Name,
		Type:       "advisory",
		Attributes: ada,
	}}
	resp.Data.Attributes.Packages = pkgs

	return &resp, nil
}

func parsePackages(jsonb []byte) (packages, error) {
	if jsonb == nil {
		return packages{}, nil
	}

	var err error
	pkgs, err := parseJSONList(jsonb)
	if err != nil {
		return nil, err
	}
	return pkgs, nil
}

func initAdvisoryDetailCache() *lru.Cache[string, AdvisoryDetailResponse] {
	middlewares.AdvisoryDetailGauge.Set(0)
	if !enableAdvisoryDetailCache {
		return nil
	}

	cache, err := lru.New[string, AdvisoryDetailResponse](advisoryDetailCacheSize)
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
		_, err = getAdvisory(database.Db, advisoryName, true)
		if err != nil {
			utils.LogError("advisoryName", advisoryName, "err", err.Error(), "can not re-load item to cache")
		}
		*count++
	}
	progress.Stop()
}

func tryGetAdvisoryFromCache(advisoryName string) *AdvisoryDetailResponse {
	if advisoryDetailCache == nil {
		return nil
	}

	val, ok := advisoryDetailCache.Get(advisoryName)
	if !ok {
		middlewares.AdvisoryDetailCnt.WithLabelValues("miss").Inc()
		return nil
	}
	middlewares.AdvisoryDetailCnt.WithLabelValues("hit").Inc()
	return &val
}

func tryAddAdvisoryToCache(advisoryName string, resp *AdvisoryDetailResponse) {
	if advisoryDetailCache == nil {
		return
	}
	evicted := advisoryDetailCache.Add(advisoryName, *resp)
	if !evicted {
		middlewares.AdvisoryDetailGauge.Inc()
	}
	utils.LogDebug("evictedV3", evicted, "advisoryName", advisoryName, "saved to cache")
}

func getAdvisory(db *gorm.DB, advisoryName string, isPreload bool) (*AdvisoryDetailResponse, error) {
	if !isPreload {
		resp := tryGetAdvisoryFromCache(advisoryName)
		if resp != nil {
			utils.LogDebug("advisoryName", advisoryName, "found in cache")
			return resp, nil // return data found in cache
		}
	}

	resp, err := getAdvisoryFromDB(db, advisoryName) // search for data in database
	if err != nil {
		return nil, err
	}

	tryAddAdvisoryToCache(advisoryName, resp) // save data to cache if initialized
	return resp, nil
}
