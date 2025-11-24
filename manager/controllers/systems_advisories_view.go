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
	Data  map[SystemID][]AdvisoryName `json:"data"`
	Links Links                       `json:"links"`
	Meta  ListMeta                    `json:"meta"`
}
type AdvisoriesSystemsResponse struct {
	Data  map[AdvisoryName][]SystemID `json:"data"`
	Links Links                       `json:"links"`
	Meta  ListMeta                    `json:"meta"`
}

type systemsAdvisoriesDBLoad struct {
	SystemID   SystemID     `query:"sp.inventory_id" gorm:"column:system_id"`
	AdvisoryID AdvisoryName `query:"am.name" gorm:"column:advisory_id"`
}

type systemsAdvisoriesViewSubDBLookup struct {
	RhAccountID int      `query:"sp.rh_account_id" gorm:"column:rh_account_id"`
	ID          int64    `query:"sp.id" gorm:"column:id"`
	SystemID    SystemID `query:"sp.inventory_id" gorm:"column:inventory_id"`
}

type advisoriesSystemsViewSubDBLookup struct {
	ID         int64        `query:"am.id" gorm:"column:advisory_id"`
	AdvisoryID AdvisoryName `query:"am.name" gorm:"column:advisory_name"`
}

var systemsAdvisoriesSelect = database.MustGetSelect(&systemsAdvisoriesDBLoad{})
var systemsAdvisoriesViewFields = database.MustGetQueryAttrs(&systemsAdvisoriesViewSubDBLookup{})
var systemsAdvisoriesViewOpts = ListOpts{
	Fields:         systemsAdvisoriesViewFields,
	DefaultFilters: nil,
	DefaultSort:    "inventory_id",
	StableSort:     "inventory_id",
	SearchFields:   nil,
}
var advisoriesSystemsViewFields = database.MustGetQueryAttrs(&advisoriesSystemsViewSubDBLookup{})
var advisoriesSystemsViewOpts = ListOpts{
	Fields:         advisoriesSystemsViewFields,
	DefaultFilters: nil,
	DefaultSort:    "advisory_id",
	StableSort:     "am.id",
	SearchFields:   nil,
}

func totalItems(tx *gorm.DB, cols string) (int, error) {
	var count int64
	err := database.DB.Table("(?) AS cq", tx.Select(cols)).Count(&count).Error
	return int(count), err
}

func systemsAdvisoriesQuery(c *gin.Context, db *gorm.DB, acc int, groups map[string]string,
	req SystemsAdvisoriesRequest) (*gorm.DB, *ListMeta, *Links, error) {
	systems := req.Systems
	advisories := req.Advisories
	sysq := database.Systems(db, acc, groups).
		Distinct("sp.rh_account_id, sp.id, sp.inventory_id").
		// we need to join system_advisories to make `limit` work properly
		// without this join it can happen that we display less items on some pages
		Joins(`LEFT JOIN system_advisories sa ON sa.system_id = sp.id
			AND sa.rh_account_id = sp.rh_account_id AND sa.rh_account_id = ?`, acc)
	if len(systems) > 0 {
		sysq = sysq.Where("sp.inventory_id in (?)", systems)
	}

	filters, err := ParseAllFilters(c, systemsAdvisoriesViewOpts)
	if err != nil {
		return nil, nil, nil, err
	} // Error handled by method itself

	sysq, _ = ApplyInventoryFilter(filters, sysq, "sp.inventory_id")
	sysq, meta, params, err := ListCommonNoLimitOffset(sysq, c, filters, systemsAdvisoriesViewOpts)
	if err != nil {
		return nil, nil, nil, err
	} // Error handled by method itself

	total, err := totalItems(sysq, "sp.rh_account_id, sp.id, sp.inventory_id")
	if err != nil {
		return nil, nil, nil, err
	}

	sysq = ApplyLimitOffset(sysq, meta)

	query := db.Table("(?) as sp", sysq).
		Select(systemsAdvisoriesSelect).
		Joins(`LEFT JOIN system_advisories sa ON sa.system_id = sp.id
			AND sa.rh_account_id = sp.rh_account_id AND sa.rh_account_id = ? AND sa.status_id = 0`, acc)
	if len(advisories) > 0 {
		query = query.Joins("LEFT JOIN advisory_metadata am ON am.id = sa.advisory_id AND am.name in (?)", advisories)
	} else {
		query = query.Joins("LEFT JOIN advisory_metadata am ON am.id = sa.advisory_id")
	}
	query = query.Order("sp.inventory_id, am.id")

	meta, links, err := UpdateMetaLinks(c, meta, total, nil, params...)
	return query, meta, links, err
}

func advisoriesSystemsQuery(c *gin.Context, db *gorm.DB, acc int, groups map[string]string,
	req SystemsAdvisoriesRequest) (*gorm.DB, *ListMeta, *Links, error) {
	systems := req.Systems
	advisories := req.Advisories
	// get all advisories for all systems in the account (with inventory.hosts join)
	advq := database.SystemAdvisories(db, acc, groups, database.JoinAdvisoryMetadata).
		Distinct("am.id, am.name")
		// we need to join system_advisories to make `limit` work properly
		// without this join it can happen that we display less items on some pages
	if len(advisories) > 0 {
		advq = advq.Where("am.name in (?)", advisories)
	}

	filters, err := ParseAllFilters(c, advisoriesSystemsViewOpts)
	if err != nil {
		return nil, nil, nil, err
	} // Error handled by method itself

	advq, meta, params, err := ListCommonNoLimitOffset(advq, c, filters, advisoriesSystemsViewOpts)
	if err != nil {
		return nil, nil, nil, err
	} // Error handled by method itself

	total, err := totalItems(advq, "am.id, am.name")
	if err != nil {
		return nil, nil, nil, err
	}

	advq = ApplyLimitOffset(advq, meta)

	spJoin := "LEFT JOIN system_platform sp ON sp.id = sa.system_id AND sa.rh_account_id = sp.rh_account_id"
	query := db.Table("(?) as am", advq).
		Distinct(systemsAdvisoriesSelect).
		Joins("LEFT JOIN system_advisories sa ON am.id = sa.advisory_id AND sa.rh_account_id = ? AND sa.status_id = 0", acc)
	if len(systems) > 0 {
		query = query.Joins(fmt.Sprintf("%s AND sp.inventory_id in (?::uuid)", spJoin), systems)
	} else {
		query = query.Joins(spJoin)
	}
	query = query.Order("am.name, sp.inventory_id")

	meta, links, err := UpdateMetaLinks(c, meta, total, nil, params...)
	return query, meta, links, err
}

func queryDB(c *gin.Context, endpoint string) ([]systemsAdvisoriesDBLoad, *ListMeta, *Links, error) {
	var req SystemsAdvisoriesRequest
	var q *gorm.DB
	var err error
	var meta *ListMeta
	var links *Links
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.LogAndRespBadRequest(c, err, fmt.Sprintf("Invalid request body: %s", err.Error()))
		return nil, nil, nil, err
	}
	acc := c.GetInt(utils.KeyAccount)
	groups := c.GetStringMapString(utils.KeyInventoryGroups)
	db := middlewares.DBFromContext(c)
	// backward compatibility, put limit/offset from json into querystring
	if req.Limit != nil {
		c.Request.URL.RawQuery += fmt.Sprintf("&limit=%d", *req.Limit)
	}
	if req.Offset != nil {
		c.Request.URL.RawQuery += fmt.Sprintf("&offset=%d", *req.Offset)
	}
	switch endpoint {
	case "SystemsAdvisories":
		q, meta, links, err = systemsAdvisoriesQuery(c, db, acc, groups, req)
	case "AdvisoriesSystems":
		q, meta, links, err = advisoriesSystemsQuery(c, db, acc, groups, req)
	default:
		return nil, nil, nil, fmt.Errorf("unknown endpoint '%s'", endpoint)
	}
	if err != nil {
		return nil, nil, nil, err
	} // Error handled by method itself

	var data []systemsAdvisoriesDBLoad
	if err := q.Find(&data).Error; err != nil {
		utils.LogAndRespError(c, err, "Database error")
		return nil, nil, nil, err
	}
	return data, meta, links, nil
}

// @Summary View system-advisory pairs for selected systems and installable advisories
// @Description View system-advisory pairs for selected systems and installable advisories
// @ID viewSystemsAdvisories
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    body    body    SystemsAdvisoriesRequest true "Request body"
// @Param    limit          query   int     false   "Limit for paging" minimum(1) maximum(100)
// @Param    offset         query   int     false   "Offset for paging"
// @Param    tags                    query   []string  false "Tag filter"
// @Param    filter[group_name] 									query []string 	false "Filter systems by inventory groups"
// @Param    filter[system_profile][sap_system]						query bool  	false "Filter only SAP systems"
// @Param    filter[system_profile][sap_sids]						query []string  false "Filter systems by their SAP SIDs"
// @Param    filter[system_profile][ansible]						query string 	false "Filter systems by ansible"
// @Param    filter[system_profile][ansible][controller_version]	query string 	false "Filter systems by ansible version"
// @Param    filter[system_profile][mssql]							query string 	false "Filter systems by mssql version"
// @Param    filter[system_profile][mssql][version]					query string 	false "Filter systems by mssql version"
// @Success 200 {object} SystemsAdvisoriesResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /views/systems/advisories [post]
func PostSystemsAdvisories(c *gin.Context) {
	data, meta, links, err := queryDB(c, "SystemsAdvisories")
	if err != nil {
		return
	}

	response := SystemsAdvisoriesResponse{
		Data:  map[SystemID][]AdvisoryName{},
		Links: *links,
		Meta:  *meta,
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
// @Param    limit          query   int     false   "Limit for paging" minimum(1) maximum(100)
// @Param    offset         query   int     false   "Offset for paging"
// @Param    tags                    query   []string  false "Tag filter"
// @Param    filter[group_name] 									query []string 	false "Filter systems by inventory groups"
// @Param    filter[system_profile][sap_system]						query bool  	false "Filter only SAP systems"
// @Param    filter[system_profile][sap_sids]						query []string  false "Filter systems by their SAP SIDs"
// @Param    filter[system_profile][ansible]						query string 	false "Filter systems by ansible"
// @Param    filter[system_profile][ansible][controller_version]	query string 	false "Filter systems by ansible version"
// @Param    filter[system_profile][mssql]							query string 	false "Filter systems by mssql version"
// @Param    filter[system_profile][mssql][version]					query string 	false "Filter systems by mssql version"
// @Success 200 {object} AdvisoriesSystemsResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /views/advisories/systems [post]
func PostAdvisoriesSystems(c *gin.Context) {
	data, meta, links, err := queryDB(c, "AdvisoriesSystems")
	if err != nil {
		return
	}

	response := AdvisoriesSystemsResponse{
		Data:  map[AdvisoryName][]SystemID{},
		Links: *links,
		Meta:  *meta,
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
