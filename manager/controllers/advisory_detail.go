package controllers

import (
	"app/base/database"
	"app/base/utils"
	"app/manager/config"
	"app/manager/middlewares"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/pkg/errors"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var advisoryDetailCache *lru.Cache[string, AdvisoryDetailResponse]

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
	Description      string                      `json:"description"`
	ModifiedDate     *time.Time                  `json:"modified_date"`
	PublicDate       *time.Time                  `json:"public_date"`
	Topic            string                      `json:"topic" gorm:"column:summary"`
	Synopsis         string                      `json:"synopsis"`
	Solution         *string                     `json:"solution"`
	AdvisoryTypeName string                      `json:"advisory_type_name"`
	Severity         *int                        `json:"severity" gorm:"column:severity_id"`
	SeverityName     *string                     `json:"severity_name,omitempty"`
	Fixes            *string                     `json:"fixes"`
	Cves             datatypes.JSONSlice[string] `json:"cves" gorm:"column:cve_list" swaggertype:"array,string"`
	References       []string                    `json:"references" query:"null" gorm:"-"`
	RebootRequired   bool                        `json:"reboot_required"`
	ReleaseVersions  datatypes.JSONSlice[string] `json:"release_versions" swaggertype:"array,string"`
	Packages         datatypes.JSONSlice[string] `json:"packages" gorm:"column:package_data" swaggertype:"array,string"`
}

func (AdvisoryDetailAttributes) TableName() string {
	return "advisory_metadata AS am"
}

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
			utils.LogAndRespNotFound(c, err, "advisory not found")
		} else {
			utils.LogAndRespError(c, err, "advisory detail error")
		}
		return
	}

	c.JSON(http.StatusOK, resp)
}

func getAdvisoryFromDB(db *gorm.DB, advisoryName string) (*AdvisoryDetailResponse, error) {
	var advisory AdvisoryDetailAttributes
	err := db.Table(advisory.TableName()).
		Select("am.*", "sev.name as severity_name", "at.name as advisory_type_name").
		Joins("LEFT JOIN advisory_severity sev ON am.severity_id = sev.id").
		Joins("JOIN advisory_type at ON am.advisory_type_id = at.id").
		Take(&advisory, "am.name = ?", advisoryName).Error
	if err != nil {
		return nil, err
	}

	var resp = AdvisoryDetailResponse{Data: AdvisoryDetailItem{
		ID:         advisoryName,
		Type:       "advisory",
		Attributes: advisory,
	}}

	return &resp, nil
}

func InitAdvisoryDetailCache() {
	middlewares.AdvisoryDetailGauge.Set(0)
	if !config.EnableAdvisoryDetailCache {
		return
	}

	var err error
	advisoryDetailCache, err = lru.New[string, AdvisoryDetailResponse](config.AdvisoryDetailCacheSize)
	if err != nil {
		panic(err)
	}
}

func PreloadAdvisoryCacheItems() {
	if !config.PreLoadCache {
		return
	}

	utils.LogInfo("cacheSize", config.AdvisoryDetailCacheSize, "loading items to advisory detail cache...")
	var advisoryNames []string
	err := database.DB.Table("advisory_metadata").Limit(config.AdvisoryDetailCacheSize).Order("public_date DESC").
		Pluck("name", &advisoryNames).Error // preload first N most recent advisories to cache
	if err != nil {
		panic(err)
	}

	progress, count := utils.LogProgress("Advisory detail cache preload", logProgressDuration, int64(len(advisoryNames)))

	for _, advisoryName := range advisoryNames {
		_, err = getAdvisory(database.DB, advisoryName, true)
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

	emptyTime := time.Time{}
	if val.Data.Attributes.PublicDate == nil || val.Data.Attributes.PublicDate.Equal(emptyTime) {
		// advisory is found in cache but was inserted from yum_updates
		// it is missing all attributes such as description, public_date, modified_date, etc.
		// these attributes are added after the advisory is synced by vmaas-sync
		// don't use the value from cache but from DB
		return nil
	}
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
