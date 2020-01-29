package controllers

import (
	"app/base/core"
	"app/base/database"
	"app/base/utils"
	"app/manager/middlewares"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"net/http"
	"time"
)

var AdvisoriesSortFields = []string{"type", "synopsis", "public_date"}

type AdvisoriesResponse struct {
	Data  []AdvisoryItem `json:"data"`
	Links Links          `json:"links"`
	Meta  AdvisoryMeta   `json:"meta"`
}

type AdvisoryWithApplicableSystems struct {
	Name              string
	Description       string
	Synopsis          string
	PublicDate        time.Time
	AdvisoryTypeID    int
	ApplicableSystems int
}

// nolint:lll
// @Summary Show me all applicable advisories for all my systems
// @Description Show me all applicable advisories for all my systems
// @ID listAdvisories
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    limit          query   int     false   "Limit for paging"
// @Param    offset         query   int     false   "Offset for paging"
// @Param    sort           query   string  false   "Sort field"    Enums(id,type,synopsis,public_date)
// @Success 200 {object} AdvisoriesResponse
// @Router /api/patch/v1/advisories [get]
func AdvisoriesListHandler(c *gin.Context) {
	account := c.GetString(middlewares.KeyAccount)

	limit, offset, err := utils.LoadLimitOffset(c, core.DefaultLimit)
	if err != nil {
		LogAndRespBadRequest(c, err, err.Error())
		return
	}

	query := buildQueryAdvisories(account)
	query, err = ApplySort(c, query, AdvisoriesSortFields...)
	if err != nil {
		LogAndRespBadRequest(c, err, "sort application failed")
		return
	}

	var total int
	err = query.Count(&total).Error
	if err != nil {
		LogAndRespError(c, err, "db error")
		return
	}

	if offset > total {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{Error: "too big offset"})
		return
	}

	var advisories []AdvisoryWithApplicableSystems
	err = query.Limit(limit).Offset(offset).Find(&advisories).Error
	if err != nil {
		LogAndRespError(c, err, "db error")
		return
	}

	data := buildAdvisoriesData(&advisories)
	links := CreateLinks("/api/patch/v1/advisories", offset, limit, total,
		"&data_format=json")
	meta := buildAdvisoriesMeta(limit, offset, total)
	var resp = AdvisoriesResponse{
		Data:  *data,
		Links: links,
		Meta:  *meta,
	}
	c.JSON(http.StatusOK, &resp)
}

func buildQueryAdvisories(account string) *gorm.DB {
	query := database.Db.Table("advisory_metadata am").
		Select("am.id AS id, am.name AS name, COALESCE(systems_affected, 0) AS applicable_systems,"+
			"synopsis,description, public_date, advisory_type_id, advisory_type_id as type").
		Joins("LEFT JOIN advisory_account_data aad ON am.id = aad.advisory_id").
		Joins("LEFT JOIN rh_account ra ON aad.rh_account_id = ra.id").
		Where("ra.name = ? OR ra.name IS NULL", account)
	return query
}

func buildAdvisoriesData(advisories *[]AdvisoryWithApplicableSystems) *[]AdvisoryItem {
	data := make([]AdvisoryItem, len(*advisories))
	for i := 0; i < len(*advisories); i++ {
		advisory := (*advisories)[i]
		data[i] = AdvisoryItem{
			Attributes: AdvisoryItemAttributes{
				Description:       advisory.Description,
				Severity:          "",
				PublicDate:        advisory.PublicDate,
				Synopsis:          advisory.Synopsis,
				AdvisoryType:      advisory.AdvisoryTypeID,
				ApplicableSystems: advisory.ApplicableSystems},
			ID:   advisory.Name,
			Type: "advisory",
		}
	}
	return &data
}

func buildAdvisoriesMeta(limit, offset, total int) *AdvisoryMeta {
	meta := AdvisoryMeta{
		DataFormat: "json",
		Filter:     nil,
		Limit:      limit,
		Offset:     offset,
		Page:       offset / limit,
		PageSize:   limit,
		Pages:      total / limit,
		TotalItems: total,
	}
	return &meta
}
