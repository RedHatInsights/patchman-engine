package controllers

import (
	"app/base/database"
	"app/manager/middlewares"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"net/http"
)

type SystemsAdvisoriesRequest struct {
	Systems    []string `json:"systems"`
	Advisories []string `json:"advisories"`
}

type SystemsAdvisoriesResponse struct {
	Data map[string][]string `json:"data"`
}

type systemsAdvisoriesDBLoad struct {
	SystemID   string `query:"sp.inventory_id"`
	AdvisoryID string `query:"am.name"`
}

var systemsAdvisoriesSelect = database.MustGetSelect(&systemsAdvisoriesDBLoad{})

func systemsAdvisoriesQuery(acc int, systems, advisories []string) *gorm.DB {
	return database.Db.
		Table("system_advisories sa").
		Select(systemsAdvisoriesSelect).
		Joins("join system_platform sp on sp.rh_account_id = ? and sp.id = sa.system_id", acc).
		Joins("join advisory_metadata am on am.id = sa.advisory_id").
		Where("sa.rh_account_id = ?", acc).
		Where("sp.inventory_id in (?)", systems).
		Where("am.name in (?)", advisories).
		Order("sp.inventory_id, am.id")
}

// @Summary View system-advisory pairs for selected systems and advisories
// @Description View system-advisory pairs for selected systems and advisories
// @ID viewSystemsAdvisories
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Success 200 {object} PackagesResponse
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
		Data: map[string][]string{},
	}

	for _, i := range data {
		response.Data[i.SystemID] = append(response.Data[i.SystemID], i.AdvisoryID)
	}
	c.JSON(http.StatusOK, response)
}
