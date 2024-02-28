package controllers

import (
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"app/manager/middlewares"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

var AdvisorySystemsFields = database.MustGetQueryAttrs(&AdvisorySystemDBLookup{})
var AdvisorySystemsSelect = database.MustGetSelect(&AdvisorySystemDBLookup{})

type AdvisorySystemsResponse struct {
	Data  []AdvisorySystemItem `json:"data"`
	Links Links                `json:"links"`
	Meta  ListMeta             `json:"meta"`
}

type AdvisorySystemItem struct {
	Attributes AdvisorySystemItemAttributes `json:"attributes"`
	ID         string                       `json:"id"`
	Type       string                       `json:"type"`
}

// nolint: lll
type AdvisorySystemDBLookup struct {
	SystemsMetaTagTotal
	AdvisorySystemItemAttributes
	SystemIDAttribute
}

// nolint: lll
type AdvisorySystemItemAttributes struct {
	SystemDisplayName
	SystemLastUpload
	SystemStale
	OSAttributes
	SystemTimestamps
	SystemTags
	SystemGroups
	BaselineIDAttr
	BaselineNameAttr
	SystemAdvisoryStatus
	SystemSatelliteManaged
	SystemBuiltPkgcache
}

type SystemsStatusID struct {
	SystemsID
	SystemAdvisoryStatus
}

var AdvisorySystemOpts = ListOpts{
	Fields: AdvisorySystemsFields,
	// By default, we show only fresh systems. If all systems are required, you must pass in:true,false filter into the api
	DefaultFilters: map[string]FilterData{
		"stale": {
			Operator: "eq",
			Values:   []string{"false"},
		},
	},
	DefaultSort:  "-last_upload",
	StableSort:   "sp.id",
	SearchFields: []string{"sp.display_name"},
}

func advisorySystemsCommon(c *gin.Context) (*gorm.DB, *ListMeta, []string, error) {
	account := c.GetInt(utils.KeyAccount)
	groups := c.GetStringMapString(utils.KeyInventoryGroups)

	advisoryName := c.Param("advisory_id")
	if advisoryName == "" {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{Error: "advisory_id param not found"})
		return nil, nil, nil, errors.New("advisory_id param not found")
	}

	db := middlewares.DBFromContext(c)
	var exists int64
	err := db.Model(&models.AdvisoryMetadata{}).
		Where("name = ? ", advisoryName).Count(&exists).Error
	if err != nil {
		LogAndRespError(c, err, "database error")
		return nil, nil, nil, err
	}
	if exists == 0 {
		err = errors.New("advisory not found")
		LogAndRespNotFound(c, err, "Advisory not found")
		return nil, nil, nil, err
	}

	query := buildAdvisorySystemsQuery(db, account, groups, advisoryName)
	opts := AdvisorySystemOpts
	filters, err := ParseAllFilters(c, opts)
	if err != nil {
		return nil, nil, nil, err
	} // Error handled in method itself
	query, _ = ApplyInventoryFilter(filters, query, "sp.inventory_id")
	query, meta, params, err := ListCommon(query, c, filters, opts)
	// Error handled in method itself
	return query, meta, params, err
}

// nolint: lll
// @Summary Show me systems on which the given advisory is applicable
// @Description Show me systems on which the given advisory is applicable
// @ID listAdvisorySystems
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    advisory_id    path    string  true    "Advisory ID"
// @Param    limit          query   int     false   "Limit for paging, set -1 to return all"
// @Param    offset         query   int     false   "Offset for paging"
// @Param    sort           query   string  false   "Sort field" Enums(id,display_name,last_evaluation,last_upload,stale,status,template,groups,satellite_managed,built_pkgcache)
// @Param    search         query   string  false   "Find matching text"
// @Param    filter[id]             query   string    false "Filter"
// @Param    filter[display_name]   query   string    false "Filter"
// @Param    filter[stale]          query   string    false "Filter"
// @Param    filter[status]         query   string    false "Filter"
// @Param    filter[template]       query   string    false "Filter"
// @Param    filter[os]             query   string    false "Filter OS version"
// @Param    filter[satellite_managed] query string  false "Filter"
// @Param    filter[built_pkgcache]    query string  false "Filter"
// @Param    tags                   query   []string  false "Tag filter"
// @Param    filter[group_name] 									query []string 	false "Filter systems by inventory groups"
// @Param    filter[system_profile][sap_system]						query string  	false "Filter only SAP systems"
// @Param    filter[system_profile][sap_sids]						query []string  false "Filter systems by their SAP SIDs"
// @Param    filter[system_profile][ansible]						query string 	false "Filter systems by ansible"
// @Param    filter[system_profile][ansible][controller_version]	query string 	false "Filter systems by ansible version"
// @Param    filter[system_profile][mssql]							query string 	false "Filter systems by mssql version"
// @Param    filter[system_profile][mssql][version]					query string 	false "Filter systems by mssql version"
// @Success 200 {object} AdvisorySystemsResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /advisories/{advisory_id}/systems [get]
func AdvisorySystemsListHandler(c *gin.Context) {
	query, meta, params, err := advisorySystemsCommon(c)
	if err != nil {
		return
	} // Error handled in method itself

	var dbItems []AdvisorySystemDBLookup

	if err = query.Scan(&dbItems).Error; err != nil {
		LogAndRespError(c, err, "database error")
		return
	}

	data, total := buildAdvisorySystemsData(dbItems)

	meta, links, err := UpdateMetaLinks(c, meta, total, nil, params...)
	if err != nil {
		return // Error handled in method itself
	}
	var resp = AdvisorySystemsResponse{
		Data:  data,
		Links: *links,
		Meta:  *meta,
	}
	c.JSON(http.StatusOK, &resp)
}

func buildAdvisorySystemsData(fields []AdvisorySystemDBLookup) ([]AdvisorySystemItem, int) {
	var total int
	var err error
	if len(fields) > 0 {
		total = fields[0].Total
	}
	data := make([]AdvisorySystemItem, len(fields))
	for i := range fields {
		if err = parseSystemItems(fields[i].TagsStr, &fields[i].AdvisorySystemItemAttributes.Tags); err != nil {
			utils.LogDebug("err", err.Error(), "inventory_id", fields[i].ID, "system tags parsing failed")
		}
		if err = parseSystemItems(fields[i].GroupsStr, &fields[i].AdvisorySystemItemAttributes.Groups); err != nil {
			utils.LogDebug("err", err.Error(), "inventory_id", fields[i].ID, "system groups parsing failed")
		}
		data[i] = AdvisorySystemItem{
			Attributes: fields[i].AdvisorySystemItemAttributes,
			ID:         fields[i].ID,
			Type:       "system",
		}
	}
	return data, total
}

func systemsIDsStatus(c *gin.Context, systems []SystemsStatusID, meta *ListMeta) (IDsStatusResponse, error) {
	var total int
	resp := IDsStatusResponse{}
	if len(systems) > 0 {
		total = systems[0].Total
	}
	if meta.Offset > total {
		err := errors.New("Offset")
		LogAndRespBadRequest(c, err, InvalidOffsetMsg)
		return resp, err
	}
	if systems == nil {
		return resp, nil
	}
	ids := make([]string, len(systems))
	data := make([]IDStatus, len(systems))
	for i, x := range systems {
		ids[i] = x.ID
		data[i] = IDStatus{x.ID, x.Status}
	}
	resp.IDs = ids
	resp.Data = data
	return resp, nil
}

// nolint: lll
// @Summary Show me systems on which the given advisory is applicable
// @Description Show me systems on which the given advisory is applicable
// @ID listAdvisorySystemsIds
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    advisory_id    path    string  true    "Advisory ID"
// @Param    limit          query   int     false   "Limit for paging, set -1 to return all"
// @Param    offset         query   int     false   "Offset for paging"
// @Param    sort    query   string  false   "Sort field" Enums(id,display_name,last_evaluation,last_upload,rhsa_count,rhba_count,rhea_count,other_count,satellite_managed,stale,built_pkgcache)
// @Param    search         query   string  false   "Find matching text"
// @Param    filter[id]              query   string  false "Filter"
// @Param    filter[insights_id]     query   string  false "Filter"
// @Param    filter[display_name]    query   string  false "Filter"
// @Param    filter[last_evaluation] query   string  false "Filter"
// @Param    filter[last_upload]     query   string  false "Filter"
// @Param    filter[rhsa_count]      query   string  false "Filter"
// @Param    filter[rhba_count]      query   string  false "Filter"
// @Param    filter[rhea_count]      query   string  false "Filter"
// @Param    filter[other_count]     query   string  false "Filter"
// @Param    filter[satellite_managed] query string  false "Filter"
// @Param    filter[stale]           query   string  false "Filter"
// @Param    filter[stale_timestamp] query   string false "Filter"
// @Param    filter[stale_warning_timestamp] query string false "Filter"
// @Param    filter[culled_timestamp] query string false "Filter"
// @Param    filter[created] query string false "Filter"
// @Param    filter[osname] query string false "Filter"
// @Param    filter[osminor] query string false "Filter"
// @Param    filter[osmajor] query string false "Filter"
// @Param    filter[os]              query   string    false "Filter OS version"
// @Param    filter[built_pkgcache]  query   string    false "Filter"
// @Param    tags                    query   []string  false "Tag filter"
// @Param    filter[group_name] 									query []string 	false "Filter systems by inventory groups"
// @Param    filter[system_profile][sap_system]						query string  	false "Filter only SAP systems"
// @Param    filter[system_profile][sap_sids]						query []string  false "Filter systems by their SAP SIDs"
// @Param    filter[system_profile][ansible]						query string 	false "Filter systems by ansible"
// @Param    filter[system_profile][ansible][controller_version]	query string 	false "Filter systems by ansible version"
// @Param    filter[system_profile][mssql]							query string 	false "Filter systems by mssql version"
// @Param    filter[system_profile][mssql][version]					query string 	false "Filter systems by mssql version"
// @Success 200 {object} IDsStatusResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /ids/advisories/{advisory_id}/systems [get]
func AdvisorySystemsListIDsHandler(c *gin.Context) {
	query, meta, _, err := advisorySystemsCommon(c)
	if err != nil {
		return
	} // Error handled in method itself

	var sids []SystemsStatusID

	if err = query.Scan(&sids).Error; err != nil {
		LogAndRespError(c, err, "database error")
		return
	}

	resp, err := systemsIDsStatus(c, sids, meta)
	if err != nil {
		return // Error handled in method itself
	}
	c.JSON(http.StatusOK, &resp)
}

func buildAdvisorySystemsQuery(db *gorm.DB, account int, groups map[string]string, advisoryName string) *gorm.DB {
	selectQuery := AdvisorySystemsSelect
	query := database.SystemAdvisories(db, account, groups).
		Select(selectQuery).
		Joins("JOIN advisory_metadata am ON am.id = sa.advisory_id").
		Joins("LEFT JOIN status st ON sa.status_id = st.id").
		Joins("LEFT JOIN baseline bl ON sp.baseline_id = bl.id AND sp.rh_account_id = bl.rh_account_id").
		Where("am.name = ?", advisoryName).
		Where("sp.stale = false")

	return query
}
