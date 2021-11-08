package controllers

import (
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	lru "github.com/hashicorp/golang-lru"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

var enableAdvisoryDetailCache = utils.GetBoolEnvOrDefault("ENABLE_ADVISORY_DETAIL_CACHE", true)
var advisoryDetailCacheSize = utils.GetIntEnvOrDefault("ADVISORY_DETAIL_CACHE_SIZE", 100)
var advisoryDetailCache = initAdvisoryDetailCache()

type AdvisoryDetailResponse struct {
	Data AdvisoryDetailItem `json:"data"`
}

type AdvisoryDetailItem struct {
	Attributes AdvisoryDetailAttributes `json:"attributes"`
	ID         string                   `json:"id"`
	Type       string                   `json:"type"`
}

type AdvisoryDetailAttributes struct {
	Description      string            `json:"description"`
	ModifiedDate     time.Time         `json:"modified_date"`
	PublicDate       time.Time         `json:"public_date"`
	Topic            string            `json:"topic"`
	Synopsis         string            `json:"synopsis"`
	Solution         string            `json:"solution"`
	AdvisoryTypeName string            `json:"advisory_type_name"`
	Severity         *int              `json:"severity"`
	Fixes            *string           `json:"fixes"`
	Cves             []string          `json:"cves"`
	Packages         map[string]string `json:"packages"`
	References       []string          `json:"references"`
	RebootRequired   bool              `json:"reboot_required"`
	ReleaseVersions  []string          `json:"release_versions"`
}

// @Summary Show me details an advisory by given advisory name
// @Description Show me details an advisory by given advisory name
// @ID detailAdvisory
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    advisory_id    path    string   true "Advisory ID"
// @Success 200 {object} AdvisoryDetailResponse
// @Router /api/patch/v1/advisories/{advisory_id} [get]
func AdvisoryDetailHandler(c *gin.Context) {
	advisoryName := c.Param("advisory_id")
	if advisoryName == "" {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{Error: "advisory_id param not found"})
		return
	}

	resp, err := getAdvisory(advisoryName)
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

type advisoryDetailItem struct {
	models.AdvisoryMetadata
	AdvisoryTypeName string
}

func getAdvisoryFromDB(advisoryName string) (*AdvisoryDetailResponse, error) {
	var advisory advisoryDetailItem
	err := database.Db.Table("advisory_metadata am").
		Select("am.*, at.name as advisory_type_name").
		Joins("JOIN advisory_type at ON am.advisory_type_id = at.id").
		Where("am.name = ?", advisoryName).Take(&advisory).Error
	if err != nil {
		return nil, err
	}

	cves, err := parseJSONList(advisory.CveList)
	if err != nil {
		return nil, errors.Wrap(err, "CVEs parsing error")
	}

	packages, err := parsePackages(advisory.PackageData)
	if err != nil {
		return nil, errors.Wrap(err, "packages parsing error")
	}

	releaseVersions, err := parseJSONList(advisory.ReleaseVersions)
	if err != nil {
		return nil, errors.Wrap(err, "release_versions parsing error")
	}

	var resp = AdvisoryDetailResponse{
		Data: AdvisoryDetailItem{
			Attributes: AdvisoryDetailAttributes{
				Description:      advisory.Description,
				ModifiedDate:     advisory.ModifiedDate,
				PublicDate:       advisory.PublicDate,
				Topic:            advisory.Summary,
				Synopsis:         advisory.Synopsis,
				Solution:         advisory.Solution,
				Severity:         advisory.SeverityID,
				AdvisoryTypeName: advisory.AdvisoryTypeName,
				Fixes:            nil,
				Cves:             cves,
				Packages:         packages,
				References:       []string{},
				RebootRequired:   advisory.RebootRequired,
				ReleaseVersions:  releaseVersions,
			},
			ID:   advisory.Name,
			Type: "advisory",
		}}
	return &resp, nil
}

func parsePackages(jsonb []byte) (map[string]string, error) {
	if jsonb == nil {
		return map[string]string{}, nil
	}

	js := json.RawMessage(string(jsonb))
	b, err := json.Marshal(js)
	if err != nil {
		return nil, err
	}

	var packages map[string]string
	err = json.Unmarshal(b, &packages)
	if err != nil {
		return nil, err
	}
	return packages, nil
}

func initAdvisoryDetailCache() *lru.Cache {
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

	utils.Log("cacheSize", advisoryDetailCacheSize).Info("loading items to advisory detail cache...")
	var advisoryNames []string
	err := database.Db.Table("advisory_metadata").Limit(advisoryDetailCacheSize).Order("public_date DESC").
		Pluck("name", &advisoryNames).Error // preload first N most recent advisories to cache
	if err != nil {
		panic(err)
	}

	for i, advisoryName := range advisoryNames {
		_, err = getAdvisory(advisoryName)
		if err != nil {
			utils.Log("advisoryName", advisoryName, "err", err.Error()).Error("can not re-load item to cache")
		}
		perc := 1000 * (i + 1) / len(advisoryNames)
		if perc%10 == 0 { // log each 1% increment
			utils.Log("percent", perc/10).Info("advisory detail cache loading")
		}
	}
}

func tryGetAdvisoryFromCache(advisoryName string) *AdvisoryDetailResponse {
	if advisoryDetailCache == nil {
		return nil
	}

	val, ok := advisoryDetailCache.Get(advisoryName)
	if !ok {
		return nil
	}

	resp := val.(AdvisoryDetailResponse)
	return &resp
}

func tryAddAdvisoryToCache(advisoryName string, resp *AdvisoryDetailResponse) {
	if advisoryDetailCache == nil {
		return
	}
	evicted := advisoryDetailCache.Add(advisoryName, *resp)
	utils.Log("evicted", evicted, "advisoryName", advisoryName).Debug("saved to cache")
}

func getAdvisory(advisoryName string) (*AdvisoryDetailResponse, error) {
	resp := tryGetAdvisoryFromCache(advisoryName)
	if resp != nil {
		utils.Log("advisoryName", advisoryName).Debug("found in cache")
		return resp, nil // return data found in cache
	}

	resp, err := getAdvisoryFromDB(advisoryName) // search for data in database
	if err != nil {
		return nil, err
	}

	tryAddAdvisoryToCache(advisoryName, resp) // save data to cache if initialized
	return resp, nil
}
