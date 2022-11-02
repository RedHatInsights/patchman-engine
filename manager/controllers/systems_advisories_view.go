package controllers

import (
	"app/base/database"
	"app/manager/middlewares"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AdvisoryName string
type SystemID string

type SystemsAdvisoriesRequest struct {
	Systems    []SystemID     `json:"systems"`
	Advisories []AdvisoryName `json:"advisories"`
	Limit      *int           `json:"limit,omitempty"`
	Offset     *int           `json:"offset,omitempty"`
}

type SystemsAdvisoriesResponse struct {
	Data map[SystemID][]AdvisoryName `json:"data"`
}
type AdvisoriesSystemsResponse struct {
	Data map[AdvisoryName][]SystemID `json:"data"`
}

type systemsAdvisoriesDBLoad struct {
	SystemID   SystemID     `query:"sp.inventory_id" gorm:"column:system_id"`
	AdvisoryID AdvisoryName `query:"am.name" gorm:"column:advisory_id"`
}

var systemsAdvisoriesSelect = database.MustGetSelect(&systemsAdvisoriesDBLoad{})

func systemsAdvisoriesQuery(acc int, systems []SystemID, advisories []AdvisoryName,
	limit, offset *int) (*gorm.DB, error) {
	sysq := database.Systems(database.Db, acc).
		Distinct("sp.rh_account_id, sp.id, sp.inventory_id").
		// we need to join system_advisories to make `limit` work properly
		// without this join it can happen that we display less items on some pages
		Joins(`JOIN system_advisories sa ON sa.system_id = sp.id
			AND sa.rh_account_id = sp.rh_account_id AND sa.rh_account_id = ?`, acc).
		Where("when_patched IS NULL").
		Order("sp.inventory_id")
	if len(systems) > 0 {
		sysq = sysq.Where("sp.inventory_id::text in (?)", systems)
	}

	err := Paginate(sysq, limit, offset)
	if err != nil {
		return nil, err
	}
	query := database.Db.Table("(?) as sp", sysq).
		Select(systemsAdvisoriesSelect).
		Joins(`JOIN system_advisories sa ON sa.system_id = sp.id
			AND sa.rh_account_id = sp.rh_account_id AND sa.rh_account_id = ?`, acc).
		Joins("JOIN advisory_metadata am ON am.id = sa.advisory_id").
		Where("when_patched IS NULL").
		Order("sp.inventory_id, am.id")
	if len(advisories) > 0 {
		query = query.Where("am.name in (?)", advisories)
	}

	return query, nil
}

func advisoriesSystemsQuery(acc int, systems []SystemID, advisories []AdvisoryName,
	limit, offset *int) (*gorm.DB, error) {
	advq := database.Db.Table("advisory_metadata am").
		Distinct("am.id, am.name").
		// we need to join system_advisories to make `limit` work properly
		// without this join it can happen that we display less items on some pages
		Joins("JOIN system_advisories sa ON am.id = sa.advisory_id AND sa.rh_account_id = ?", acc).
		Where("when_patched IS NULL").
		Order("am.id")
	if len(advisories) > 0 {
		advq = advq.Where("am.name in (?)", advisories)
	}

	err := Paginate(advq, limit, offset)
	if err != nil {
		return nil, err
	}
	query := database.Db.Table("(?) as am", advq).
		Select(systemsAdvisoriesSelect).
		Joins("JOIN system_advisories sa ON am.id = sa.advisory_id AND sa.rh_account_id = ?", acc).
		Joins("JOIN system_platform sp ON sp.id = sa.system_id AND sa.rh_account_id = sp.rh_account_id").
		Where("when_patched IS NULL").
		Order("am.id, sp.inventory_id")
	if len(systems) > 0 {
		query = query.Where("sp.inventory_id::text in (?)", systems)
	}
	return query, nil
}

func queryDB(c *gin.Context, endpoint string) ([]systemsAdvisoriesDBLoad, error) {
	var req SystemsAdvisoriesRequest
	var q *gorm.DB
	var err error
	if err := c.ShouldBindJSON(&req); err != nil {
		LogAndRespBadRequest(c, err, "Invalid request body")
		return nil, err
	}
	acc := c.GetInt(middlewares.KeyAccount)
	switch endpoint {
	case "SystemsAdvisories":
		q, err = systemsAdvisoriesQuery(acc, req.Systems, req.Advisories, req.Limit, req.Offset)
	case "AdvisoriesSystems":
		q, err = advisoriesSystemsQuery(acc, req.Systems, req.Advisories, req.Limit, req.Offset)
	default:
		return nil, fmt.Errorf("unknown endpoint '%s'", endpoint)
	}
	if err != nil {
		LogAndRespBadRequest(c, err, err.Error())
	}

	var data []systemsAdvisoriesDBLoad
	if err := q.Find(&data).Error; err != nil {
		LogAndRespError(c, err, "Database error")
		return nil, err
	}
	return data, nil
}

// @Summary View system-advisory pairs for selected systems and advisories
// @Description View system-advisory pairs for selected systems and advisories
// @ID viewSystemsAdvisories
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    body    body    SystemsAdvisoriesRequest true "Request body"
// @Success 200 {object} SystemsAdvisoriesResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /views/systems/advisories [post]
func PostSystemsAdvisories(c *gin.Context) {
	data, err := queryDB(c, "SystemsAdvisories")
	if err != nil {
		return
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
// @ID viewAdvisoriesSystems
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    body    body    SystemsAdvisoriesRequest true "Request body"
// @Success 200 {object} AdvisoriesSystemsResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /views/advisories/systems [post]
func PostAdvisoriesSystems(c *gin.Context) {
	data, err := queryDB(c, "AdvisoriesSystems")
	if err != nil {
		return
	}

	response := AdvisoriesSystemsResponse{
		Data: map[AdvisoryName][]SystemID{},
	}

	for _, i := range data {
		response.Data[i.AdvisoryID] = append(response.Data[i.AdvisoryID], i.SystemID)
	}
	c.JSON(http.StatusOK, response)
}
