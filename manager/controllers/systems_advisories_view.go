package controllers

import (
	"app/base/database"
	"app/manager/middlewares"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"net/http"
)

type AdvisoryName string
type SystemID string

type SystemsAdvisoriesRequest struct {
	Systems    []SystemID     `json:"systems"`
	Advisories []AdvisoryName `json:"advisories"`
}

type SystemsAdvisoriesResponse struct {
	Data map[SystemID][]AdvisoryName `json:"data"`
}
type AdvisoriesSystemsResponse struct {
	Data map[AdvisoryName][]SystemID `json:"data"`
}

type systemsAdvisoriesDBLoad struct {
	SystemID   SystemID     `query:"sp.inventory_id"`
	AdvisoryID AdvisoryName `query:"am.name"`
}

var systemsAdvisoriesSelect = database.MustGetSelect(&systemsAdvisoriesDBLoad{})

func systemsAdvisoriesQuery(acc int, systems []SystemID, advisories []AdvisoryName) *gorm.DB {
	query := database.Db.
		Table("system_advisories sa").
		Select(systemsAdvisoriesSelect).
		Joins("join system_platform sp on sp.rh_account_id = ? and sp.id = sa.system_id", acc).
		Joins("join advisory_metadata am on am.id = sa.advisory_id").
		Where("sp.rh_account_id = ?", acc).
		Where("sp.inventory_id in (?)", systems).
		Where("am.name in (?)", advisories).
		Order("sp.inventory_id, am.id")

	if applyInventoryHosts {
		query = query.Joins("JOIN inventory.hosts ih ON ih.id::text = sp.inventory_id")
	}
	return query
}

// @Summary View system-advisory pairs for selected systems and advisories
// @Description View system-advisory pairs for selected systems and advisories
// @ID viewSystemsAdvisories
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    body    body    SystemsAdvisoriesRequest true "Request body"
// @Success 200 {object} SystemsAdvisoriesResponse
// @Router /api/patch/v1/packages/ [post]
func PostSystemsAdvisories(c *gin.Context) {
	var req SystemsAdvisoriesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		LogAndRespBadRequest(c, err, "Invalid request body")
	}
	acc := c.GetInt(middlewares.KeyAccount)
	q := systemsAdvisoriesQuery(acc, req.Systems, req.Advisories)
	var data []systemsAdvisoriesDBLoad
	if err := q.Find(&data).Error; err != nil {
		LogAndRespError(c, err, "Database error")
	}

	response := SystemsAdvisoriesResponse{
		Data: map[SystemID][]AdvisoryName{},
	}

	for _, i := range data {
		response.Data[i.SystemID] = append(response.Data[i.SystemID], i.AdvisoryID)
	}
	c.JSON(http.StatusOK, response)
}

// @Summary View advisory-system pairs for selected systems and advisories
// @Description View advisory-system pairs for selected systems and advisories
// @ID viewSystemsAdvisories
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    body    body    SystemsAdvisoriesRequest true "Request body"
// @Success 200 {object} AdvisoriesSystemsResponse
// @Router /api/patch/v1/packages/ [post]
func PostAdvisoriesSystems(c *gin.Context) {
	var req SystemsAdvisoriesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		LogAndRespBadRequest(c, err, "Invalid request body")
	}
	acc := c.GetInt(middlewares.KeyAccount)
	q := systemsAdvisoriesQuery(acc, req.Systems, req.Advisories)
	var data []systemsAdvisoriesDBLoad
	if err := q.Find(&data).Error; err != nil {
		LogAndRespError(c, err, "Database error")
	}

	response := AdvisoriesSystemsResponse{
		Data: map[AdvisoryName][]SystemID{},
	}

	for _, i := range data {
		response.Data[i.AdvisoryID] = append(response.Data[i.AdvisoryID], i.SystemID)
	}
	c.JSON(http.StatusOK, response)
}
