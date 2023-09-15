package controllers

import (
	"app/base/database"
	"app/base/utils"
	"app/manager/middlewares"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

var PackageSystemFields = database.MustGetQueryAttrs(&PackageSystemDBLookup{})
var PackageSystemsSelect = database.MustGetSelect(&PackageSystemDBLookup{})
var PackageSystemsOpts = ListOpts{
	Fields: PackageSystemFields,
	// By default, we show only fresh systems. If all systems are required, you must pass in:true,false filter into the api
	DefaultFilters: map[string]FilterData{},
	DefaultSort:    "id",
	StableSort:     "id",
	SearchFields:   []string{"sp.display_name"},
}

//nolint:lll
type PackageSystemItemCommon struct {
	SystemIDAttribute
	SystemDisplayName
	InstalledEVRA string         `json:"installed_evra" csv:"installed_evra" query:"p.evra" gorm:"column:installed_evra"`
	AvailableEVRA string         `json:"available_evra" csv:"available_evra" query:"spkg.latest_evra" gorm:"column:available_evra"`
	Updatable     bool           `json:"updatable" csv:"updatable" query:"(update_status(spkg.update_data) = 'Installable')" gorm:"column:updatable"`
	Tags          SystemTagsList `json:"tags" csv:"tags" query:"null" gorm:"-"`
	BaselineAttributes
}

type PackageSystemItemV2 struct {
	PackageSystemItemCommon
}

//nolint:lll
type PackageSystemItemV3 struct {
	PackageSystemItemCommon
	BaselineIDAttr
	OSAttributes
	UpdateStatus string `json:"update_status" csv:"update_status" query:"update_status(spkg.update_data)" gorm:"column:update_status"`
	SystemGroups
}

type PackageSystemDBLookup struct {
	SystemsMetaTagTotal

	PackageSystemItemV3
}

type PackageSystemsResponseV2 struct {
	Data  []PackageSystemItemV2 `json:"data"`
	Links Links                 `json:"links"`
	Meta  ListMeta              `json:"meta"`
}

type PackageSystemsResponseV3 struct {
	Data  []PackageSystemItemV3 `json:"data"`
	Links Links                 `json:"links"`
	Meta  ListMeta              `json:"meta"`
}

func packagesByNameQuery(db *gorm.DB, pkgName string) *gorm.DB {
	return db.Table("package p").
		Joins("INNER JOIN package_name pn ON p.name_id = pn.id").
		Where("pn.name = ?", pkgName)
}

func packageSystemsQuery(db *gorm.DB, acc int, groups map[string]string, packageName string, packageIDs []int,
) *gorm.DB {
	query := database.SystemPackages(db, acc, groups).
		Select(PackageSystemsSelect).
		Joins("LEFT JOIN baseline bl ON sp.baseline_id = bl.id AND sp.rh_account_id = bl.rh_account_id").
		Where("sp.stale = false").
		Where("pn.name = ?", packageName).
		Where("spkg.package_id in (?)", packageIDs)

	return query
}

func packageSystemsCommon(db *gorm.DB, c *gin.Context) (*gorm.DB, *ListMeta, []string, error) {
	account := c.GetInt(middlewares.KeyAccount)
	groups := c.GetStringMapString(middlewares.KeyInventoryGroups)
	var filters map[string]FilterData

	packageName := c.Param("package_name")
	if packageName == "" {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{Error: "package_name param not found"})
		return nil, nil, nil, errors.New("package_name param not found")
	}

	var packageIDs []int
	if err := packagesByNameQuery(db, packageName).Pluck("p.id", &packageIDs).Error; err != nil {
		LogAndRespError(c, err, "database error")
		return nil, nil, nil, err
	}

	if len(packageIDs) == 0 {
		LogAndRespNotFound(c, errors.New("not found"), "package not found")
		return nil, nil, nil, errors.New("package not found")
	}

	query := packageSystemsQuery(db, account, groups, packageName, packageIDs)
	filters, err := ParseAllFilters(c, PackageSystemsOpts)
	if err != nil {
		return nil, nil, nil, err
	} // Error handled in method itself
	query, _ = ApplyInventoryFilter(filters, query, "sp.inventory_id")
	query, meta, params, err := ListCommon(query, c, filters, PackageSystemsOpts)
	// Error handled in method itself
	return query, meta, params, err
}

// nolint: dupl
// @Summary Show me all my systems which have a package installed
// @Description  Show me all my systems which have a package installed
// @ID packageSystems
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    limit          query   int     false   "Limit for paging, set -1 to return all"
// @Param    offset         query   int     false   "Offset for paging"
// @Param    package_name    path    string    true  "Package name"
// @Param    tags            query   []string  false "Tag filter"
// @Param    filter[group_name] 									query []string 	false "Filter systems by inventory groups"
// @Param    filter[system_profile][sap_system]						query string  	false "Filter only SAP systems"
// @Param    filter[system_profile][sap_sids]						query []string  false "Filter systems by their SAP SIDs"
// @Param    filter[system_profile][ansible]						query string 	false "Filter systems by ansible"
// @Param    filter[system_profile][ansible][controller_version]	query string 	false "Filter systems by ansible version"
// @Param    filter[system_profile][mssql]							query string 	false "Filter systems by mssql version"
// @Param    filter[system_profile][mssql][version]					query string 	false "Filter systems by mssql version"
// @Param    filter[updatable]       								query   bool    false "Filter"
// @Success 200 {object} PackageSystemsResponseV3
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /packages/{package_name}/systems [get]
func PackageSystemsListHandler(c *gin.Context) {
	db := middlewares.DBFromContext(c)
	apiver := c.GetInt(middlewares.KeyApiver)

	query, meta, params, err := packageSystemsCommon(db, c)
	if err != nil {
		return
	} // Error handled in method itself

	var systems []PackageSystemDBLookup
	err = query.Find(&systems).Error
	if err != nil {
		LogAndRespError(c, err, "database error")
		return
	}

	outputItems, total := packageSystemDBLookups2PackageSystemItemsV3(systems)

	meta, links, err := UpdateMetaLinks(c, meta, total, nil, params...)
	if err != nil {
		return // Error handled in method itself
	}
	if apiver < 3 {
		dataV2 := packageSystemItemV3toV2(outputItems)
		var resp = PackageSystemsResponseV2{
			Data:  dataV2,
			Links: *links,
			Meta:  *meta,
		}
		c.JSON(http.StatusOK, &resp)
		return
	}

	response := PackageSystemsResponseV3{
		Data:  outputItems,
		Links: *links,
		Meta:  *meta,
	}
	c.JSON(http.StatusOK, response)
}

// nolint: dupl
// @Summary Show me all my systems which have a package installed
// @Description  Show me all my systems which have a package installed
// @ID packageSystemsIds
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    limit          query   int     false   "Limit for paging, set -1 to return all"
// @Param    offset         query   int     false   "Offset for paging"
// @Param    package_name    path    string    true  "Package name"
// @Param    tags            query   []string  false "Tag filter"
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
// @Router /ids/packages/{package_name}/systems [get]
func PackageSystemsListIDsHandler(c *gin.Context) {
	db := middlewares.DBFromContext(c)
	apiver := c.GetInt(middlewares.KeyApiver)
	query, meta, _, err := packageSystemsCommon(db, c)
	if err != nil {
		return
	} // Error handled in method itself

	var sids []SystemsStatusID
	err = query.Find(&sids).Error
	if err != nil {
		LogAndRespError(c, err, "database error")
		return
	}

	resp, err := systemsIDsStatus(c, sids, meta)
	if err != nil {
		return // Error handled in method itself
	}
	if apiver < 3 {
		c.JSON(http.StatusOK, &resp.IDsResponse)
		return
	}
	c.JSON(http.StatusOK, &resp)
}

func packageSystemDBLookups2PackageSystemItemsV3(systems []PackageSystemDBLookup) ([]PackageSystemItemV3, int) {
	var total int
	if len(systems) > 0 {
		total = systems[0].Total
	}
	data := make([]PackageSystemItemV3, len(systems))
	for i, system := range systems {
		if err := parseSystemItems(system.TagsStr, &system.PackageSystemItemV3.Tags); err != nil {
			utils.LogDebug("err", err.Error(), "inventory_id", system.ID, "system tags parsing failed")
		}
		if err := parseSystemItems(system.GroupsStr, &system.PackageSystemItemV3.Groups); err != nil {
			utils.LogDebug("err", err.Error(), "inventory_id", system.ID, "system groups parsing failed")
		}
		data[i] = system.PackageSystemItemV3
	}
	return data, total
}

func packageSystemItemV3toV2(systems []PackageSystemItemV3) []PackageSystemItemV2 {
	nSystems := len(systems)
	systemsV2 := make([]PackageSystemItemV2, nSystems)
	for i := 0; i < nSystems; i++ {
		systemsV2[i] = PackageSystemItemV2{
			PackageSystemItemCommon: systems[i].PackageSystemItemCommon,
		}
	}
	return systemsV2
}
