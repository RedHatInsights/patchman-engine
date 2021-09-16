package controllers

import (
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/hashicorp/golang-lru"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"net/http"
	"time"
)

var enableAdvisoryDetailCache = utils.GetBoolEnvOrDefault("ENABLE_ADVISORY_DETAIL_CACHE", true)
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
	Description    string            `json:"description"`
	ModifiedDate   time.Time         `json:"modified_date"`
	PublicDate     time.Time         `json:"public_date"`
	Topic          string            `json:"topic"`
	Synopsis       string            `json:"synopsis"`
	Solution       string            `json:"solution"`
	Severity       *int              `json:"severity"`
	Fixes          *string           `json:"fixes"`
	Cves           []string          `json:"cves"`
	Packages       map[string]string `json:"packages"`
	References     []string          `json:"references"`
	RebootRequired bool              `json:"reboot_required"`
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

func getAdvisoryFromDB(advisoryName string) (*AdvisoryDetailResponse, error) {
	var advisory models.AdvisoryMetadata
	err := database.Db.Where("name = ?", advisoryName).First(&advisory).Error
	if err != nil {
		return nil, err
	}

	cves, err := parseCVEs(advisory.CveList)
	if err != nil {
		return nil, errors.Wrap(err, "CVEs parsing error")
	}

	packages, err := parsePackages(advisory.PackageData)
	if err != nil {
		return nil, errors.Wrap(err, "packages parsing error")
	}

	var resp = AdvisoryDetailResponse{
		Data: AdvisoryDetailItem{
			Attributes: AdvisoryDetailAttributes{
				Description:    advisory.Description,
				ModifiedDate:   advisory.ModifiedDate,
				PublicDate:     advisory.PublicDate,
				Topic:          advisory.Summary,
				Synopsis:       advisory.Synopsis,
				Solution:       advisory.Solution,
				Severity:       advisory.SeverityID,
				Fixes:          nil,
				Cves:           cves,
				Packages:       packages,
				References:     []string{},
				RebootRequired: advisory.RebootRequired,
			},
			ID:   advisory.Name,
			Type: "advisory",
		}}
	return &resp, nil
}

func parseCVEs(jsonb []byte) ([]string, error) {
	if jsonb == nil {
		return []string{}, nil
	}

	js := json.RawMessage(string(jsonb))
	b, err := json.Marshal(js)
	if err != nil {
		return nil, err
	}

	var cves []string
	err = json.Unmarshal(b, &cves)
	if err != nil {
		return nil, err
	}
	return cves, nil
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

	cacheSize := utils.GetIntEnvOrDefault("ADVISORY_DETAIL_CACHE_SIZE", 1000)
	cache, err := lru.New(cacheSize)
	if err != nil {
		panic(err)
	}

	return cache
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
