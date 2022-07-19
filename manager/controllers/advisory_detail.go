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
var advisoryDetailCacheV1 = initAdvisoryDetailCache()
var advisoryDetailCacheV2 = initAdvisoryDetailCache()

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

// AdvisoryDetail handler for v1 API
// Don't annotate it with swaggo/swag annotations
// because we want to generate openapi.json only for v2 API
func AdvisoryDetailHandlerV1(c *gin.Context) {
	advisoryDetailHandler(c, "v1")
}

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
func AdvisoryDetailHandlerV2(c *gin.Context) {
	advisoryDetailHandler(c, "v2")
}

func advisoryDetailHandler(c *gin.Context, apiver string) {
	advisoryName := c.Param("advisory_id")
	if advisoryName == "" {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{Error: "advisory_id param not found"})
		return
	}

	if !isFilterInURLValid(c) {
		return
	}

	var err error
	var respV1 *AdvisoryDetailResponseV1
	var respV2 *AdvisoryDetailResponseV2
	switch apiver {
	case "v1":
		respV1, err = getAdvisoryV1(advisoryName)
	case "v2":
		respV2, err = getAdvisoryV2(advisoryName)
	}
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			LogAndRespNotFound(c, err, "advisory not found")
		} else {
			LogAndRespError(c, err, "advisory detail error")
		}
		return
	}

	switch apiver {
	case "v1":
		c.JSON(http.StatusOK, respV1)
	case "v2":
		c.JSON(http.StatusOK, respV2)
	}
}

func getAdvisoryFromDB(advisoryName string) (*models.AdvisoryMetadata, *AdvisoryDetailAttributes, error) {
	var advisory models.AdvisoryMetadata
	err := database.Db.Table(advisory.TableName()).
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

func getAdvisoryFromDBV1(advisoryName string) (*AdvisoryDetailResponseV1, error) {
	advisory, ada, err := getAdvisoryFromDB(advisoryName)
	if err != nil {
		return nil, err
	}

	pkgs, _, err := parsePackages(advisory.PackageData)
	if err != nil {
		return nil, errors.Wrap(err, "packages parsing error")
	}

	var resp = AdvisoryDetailResponseV1{Data: AdvisoryDetailItemV1{
		AdvisoryDetailItem: AdvisoryDetailItem{ID: advisory.Name, Type: "advisory"},
		Attributes: AdvisoryDetailAttributesV1{
			AdvisoryDetailAttributes: *ada,
			Packages:                 pkgs,
		},
	}}
	return &resp, nil
}

func getAdvisoryFromDBV2(advisoryName string) (*AdvisoryDetailResponseV2, error) {
	advisory, ada, err := getAdvisoryFromDB(advisoryName)
	if err != nil {
		return nil, err
	}

	_, pkgs, err := parsePackages(advisory.PackageData)
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

func parsePackages(jsonb []byte) (packagesV1, packagesV2, error) {
	if jsonb == nil {
		return packagesV1{}, packagesV2{}, nil
	}

	js := json.RawMessage(string(jsonb))
	b, err := json.Marshal(js)
	if err != nil {
		return nil, nil, err
	}

	pkgsV1 := make(packagesV1)
	var pkgsV2 packagesV2
	err = json.Unmarshal(b, &pkgsV2)
	if err != nil {
		return nil, nil, err
	}
	// assigning first pkg to packages map in api/v1
	// it shows incorrect packages info
	// but we need to maintain backward compatibility
	if len(pkgsV2) > 0 {
		nevra, err := utils.ParseNevra(pkgsV2[0])
		if err != nil {
			return nil, pkgsV2, errors.Wrapf(err, "Could not parse nevra %s", pkgsV2[0])
		}
		pkgsV1[nevra.Name] = nevra.EVRAString()
	}
	return pkgsV1, pkgsV2, nil
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
		_, err = getAdvisoryV1(advisoryName)
		if err != nil {
			utils.Log("advisoryName", advisoryName, "err", err.Error()).Error("can not re-load item to cache - V1")
		}
		_, err = getAdvisoryV2(advisoryName)
		if err != nil {
			utils.Log("advisoryName", advisoryName, "err", err.Error()).Error("can not re-load item to cache - V2")
		}
		perc := 1000 * (i + 1) / len(advisoryNames)
		if perc%10 == 0 { // log each 1% increment
			utils.Log("percent", perc/10).Info("advisory detail cache loading")
		}
	}
}

func tryGetAdvisoryFromCacheV1(advisoryName string) *AdvisoryDetailResponseV1 {
	if advisoryDetailCacheV1 == nil {
		return nil
	}

	val, ok := advisoryDetailCacheV1.Get(advisoryName)
	if !ok {
		return nil
	}
	resp := val.(AdvisoryDetailResponseV1)
	return &resp
}

func tryGetAdvisoryFromCacheV2(advisoryName string) *AdvisoryDetailResponseV2 {
	if advisoryDetailCacheV2 == nil {
		return nil
	}

	val, ok := advisoryDetailCacheV2.Get(advisoryName)
	if !ok {
		return nil
	}
	resp := val.(AdvisoryDetailResponseV2)
	return &resp
}

func tryAddAdvisoryToCacheV1(advisoryName string, resp *AdvisoryDetailResponseV1) {
	if advisoryDetailCacheV1 == nil {
		return
	}
	evicted := advisoryDetailCacheV1.Add(advisoryName, *resp)
	utils.Log("evictedV1", evicted, "advisoryName", advisoryName).Debug("saved to cache")
}

func tryAddAdvisoryToCacheV2(advisoryName string, resp *AdvisoryDetailResponseV2) {
	if advisoryDetailCacheV2 == nil {
		return
	}
	evicted := advisoryDetailCacheV2.Add(advisoryName, *resp)
	utils.Log("evictedV2", evicted, "advisoryName", advisoryName).Debug("saved to cache")
}

func getAdvisoryV1(advisoryName string) (*AdvisoryDetailResponseV1, error) {
	resp := tryGetAdvisoryFromCacheV1(advisoryName)
	if resp != nil {
		utils.Log("advisoryName", advisoryName).Debug("found in cache")
		return resp, nil // return data found in cache
	}

	resp, err := getAdvisoryFromDBV1(advisoryName) // search for data in database
	if err != nil {
		return nil, err
	}

	tryAddAdvisoryToCacheV1(advisoryName, resp) // save data to cache if initialized
	return resp, nil
}

func getAdvisoryV2(advisoryName string) (*AdvisoryDetailResponseV2, error) {
	resp := tryGetAdvisoryFromCacheV2(advisoryName)
	if resp != nil {
		utils.Log("advisoryName", advisoryName).Debug("found in cache")
		return resp, nil // return data found in cache
	}

	resp, err := getAdvisoryFromDBV2(advisoryName) // search for data in database
	if err != nil {
		return nil, err
	}

	tryAddAdvisoryToCacheV2(advisoryName, resp) // save data to cache if initialized
	return resp, nil
}
