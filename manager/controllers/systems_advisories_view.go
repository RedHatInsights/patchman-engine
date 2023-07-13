package controllers

import (
	"app/base/database"
	"app/base/utils"
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
	Meta ListMeta                    `json:"meta"`
}
type AdvisoriesSystemsResponse struct {
	Data map[AdvisoryName][]SystemID `json:"data"`
	Meta ListMeta                    `json:"meta"`
}

type systemsAdvisoriesDBLoad struct {
	SystemID   SystemID     `query:"sp.inventory_id" gorm:"column:system_id"`
	AdvisoryID AdvisoryName `query:"am.name" gorm:"column:advisory_id"`
}

var systemsAdvisoriesSelect = database.MustGetSelect(&systemsAdvisoriesDBLoad{})

func totalItems(tx *gorm.DB, cols string) (int, error) {
	var count int64
	err := database.Db.Table("(?) AS cq", tx.Select(cols)).Count(&count).Error
	return int(count), err
}

func systemsAdvisoriesQuery(db *gorm.DB, acc int, groups map[string]string, systems []SystemID,
	advisories []AdvisoryName, limit, offset *int, apiver int) (*gorm.DB, int, int, int, error) {
	sysq := database.Systems(db, acc, groups).
		Distinct("sp.rh_account_id, sp.id, sp.inventory_id").
		// we need to join system_advisories to make `limit` work properly
		// without this join it can happen that we display less items on some pages
		Joins(`LEFT JOIN system_advisories sa ON sa.system_id = sp.id
			AND sa.rh_account_id = sp.rh_account_id AND sa.rh_account_id = ?`, acc).
		Order("sp.inventory_id")
	if len(systems) > 0 {
		sysq = sysq.Where("sp.inventory_id::text in (?)", systems)
	}

	total, err := totalItems(sysq, "sp.rh_account_id, sp.id, sp.inventory_id")
	if err != nil {
		return nil, 0, 0, 0, err
	}

	lim, off, err := Paginate(sysq, limit, offset)
	if err != nil {
		return nil, total, lim, off, err
	}

	installableOnly := ""
	if apiver > 2 {
		// display only installable advisories in v3 api
		installableOnly = "AND sa.status_id = 0"
	}

	query := db.Table("(?) as sp", sysq).
		Select(systemsAdvisoriesSelect).
		Joins(fmt.Sprintf(`LEFT JOIN system_advisories sa ON sa.system_id = sp.id
			AND sa.rh_account_id = sp.rh_account_id AND sa.rh_account_id = ? %s`, installableOnly), acc)
	if len(advisories) > 0 {
		query = query.Joins("LEFT JOIN advisory_metadata am ON am.id = sa.advisory_id AND am.name in (?)", advisories)
	} else {
		query = query.Joins("LEFT JOIN advisory_metadata am ON am.id = sa.advisory_id")
	}
	query = query.Order("sp.inventory_id, am.id")

	return query, total, lim, off, nil
}

func advisoriesSystemsQuery(db *gorm.DB, acc int, groups map[string]string, systems []SystemID,
	advisories []AdvisoryName, limit, offset *int, apiver int) (*gorm.DB, int, int, int, error) {
	// TODO: inventory groups check
	utils.LogWarn("groups", groups, "TODO: USE INVENTORY GROUPS!")
	advq := db.Table("advisory_metadata am").
		Distinct("am.id, am.name").
		// we need to join system_advisories to make `limit` work properly
		// without this join it can happen that we display less items on some pages
		Joins("JOIN system_advisories sa ON am.id = sa.advisory_id AND sa.rh_account_id = ?", acc).
		Order("am.name, am.id")
	if len(advisories) > 0 {
		advq = advq.Where("am.name in (?)", advisories)
	}

	total, err := totalItems(advq, "am.id, am.name")
	if err != nil {
		return nil, 0, 0, 0, err
	}

	lim, off, err := Paginate(advq, limit, offset)
	if err != nil {
		return nil, total, lim, off, err
	}

	installableOnly := ""
	if apiver > 2 {
		// display only systems with installable advisories in v3 api
		installableOnly = "AND sa.status_id = 0"
	}

	spJoin := "LEFT JOIN system_platform sp ON sp.id = sa.system_id AND sa.rh_account_id = sp.rh_account_id"
	query := db.Table("(?) as am", advq).
		Distinct(systemsAdvisoriesSelect).
		Joins(
			fmt.Sprintf(
				"LEFT JOIN system_advisories sa ON am.id = sa.advisory_id AND sa.rh_account_id = ? %s",
				installableOnly,
			),
			acc,
		)
	if len(systems) > 0 {
		query = query.Joins(fmt.Sprintf("%s AND sp.inventory_id::text in (?)", spJoin), systems)
	} else {
		query = query.Joins(spJoin)
	}
	query = query.Order("am.name, sp.inventory_id")
	return query, total, lim, off, nil
}

func queryDB(c *gin.Context, endpoint string) ([]systemsAdvisoriesDBLoad, *ListMeta, error) {
	var req SystemsAdvisoriesRequest
	var q *gorm.DB
	var err error
	var total int
	var limit int
	var offset int
	if err := c.ShouldBindJSON(&req); err != nil {
		LogAndRespBadRequest(c, err, "Invalid request body")
		return nil, nil, err
	}
	acc := c.GetInt(middlewares.KeyAccount)
	apiver := c.GetInt(middlewares.KeyApiver)
	groups := c.GetStringMapString(middlewares.KeyInventoryGroups)
	db := middlewares.DBFromContext(c)
	switch endpoint {
	case "SystemsAdvisories":
		q, total, limit, offset, err = systemsAdvisoriesQuery(
			db, acc, groups, req.Systems, req.Advisories, req.Limit, req.Offset, apiver)
	case "AdvisoriesSystems":
		q, total, limit, offset, err = advisoriesSystemsQuery(
			db, acc, groups, req.Systems, req.Advisories, req.Limit, req.Offset, apiver)
	default:
		return nil, nil, fmt.Errorf("unknown endpoint '%s'", endpoint)
	}
	if err != nil {
		LogAndRespBadRequest(c, err, err.Error())
	}

	var data []systemsAdvisoriesDBLoad
	if err := q.Find(&data).Error; err != nil {
		LogAndRespError(c, err, "Database error")
		return nil, nil, err
	}
	meta := ListMeta{
		Limit:      limit,
		Offset:     offset,
		TotalItems: total,
	}
	return data, &meta, nil
}

// @Summary View system-advisory pairs for selected systems and installable advisories
// @Description View system-advisory pairs for selected systems and installable advisories
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
	data, meta, err := queryDB(c, "SystemsAdvisories")
	if err != nil {
		return
	}

	response := SystemsAdvisoriesResponse{
		Data: map[SystemID][]AdvisoryName{},
		Meta: *meta,
	}

	for _, i := range data {
		if _, has := response.Data[i.SystemID]; has && i.AdvisoryID == "" {
			// don't append empty values to slices with len > 1
			continue
		}
		if _, has := response.Data[i.SystemID]; !has && i.AdvisoryID == "" {
			response.Data[i.SystemID] = []AdvisoryName{}
			continue
		}
		response.Data[i.SystemID] = append(response.Data[i.SystemID], i.AdvisoryID)
	}
	c.JSON(http.StatusOK, response)
}

// @Summary View advisory-system pairs for selected systems and installable advisories
// @Description View advisory-system pairs for selected systems and installable advisories
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
	data, meta, err := queryDB(c, "AdvisoriesSystems")
	if err != nil {
		return
	}

	response := AdvisoriesSystemsResponse{
		Data: map[AdvisoryName][]SystemID{},
		Meta: *meta,
	}

	for _, i := range data {
		if _, has := response.Data[i.AdvisoryID]; has && i.SystemID == "" {
			// don't append empty values to slices with len > 1
			continue
		}
		if _, has := response.Data[i.AdvisoryID]; !has && i.SystemID == "" {
			response.Data[i.AdvisoryID] = []SystemID{}
			continue
		}
		response.Data[i.AdvisoryID] = append(response.Data[i.AdvisoryID], i.SystemID)
	}
	c.JSON(http.StatusOK, response)
}
