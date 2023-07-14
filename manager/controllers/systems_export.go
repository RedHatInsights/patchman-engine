package controllers

import (
	"app/manager/middlewares"

	"github.com/gin-gonic/gin"
)

// @Summary Export systems for my account
// @Description  Export systems for my account
// @ID exportSystems
// @Security RhIdentity
// @Accept   json
// @Produce  json,text/csv
// @Param    search         query   string  false   "Find matching text"
// @Param    filter[id]              query   string  false "Filter"
// @Param    filter[display_name]    query   string  false "Filter"
// @Param    filter[last_upload]     query   string  false "Filter"
// @Param    filter[rhsa_count]      query   string  false "Filter"
// @Param    filter[rhba_count]      query   string  false "Filter"
// @Param    filter[rhea_count]      query   string  false "Filter"
// @Param    filter[other_count]     query   string  false "Filter"
// @Param    filter[stale]           query   string  false "Filter"
// @Param    filter[packages_installed] query string false "Filter"
// @Param    filter[system_profile][sap_system]						query string  	false "Filter only SAP systems"
// @Param    filter[system_profile][sap_sids][in]					query []string  false "Filter systems by their SAP SIDs"
// @Param    filter[system_profile][ansible]						query string 	false "Filter systems by ansible"
// @Param    filter[system_profile][ansible][controller_version]	query string 	false "Filter systems by ansible version"
// @Param    filter[system_profile][mssql]							query string 	false "Filter systems by mssql version"
// @Param    filter[system_profile][mssql][version]					query string 	false "Filter systems by mssql version"
// @Param    filter[baseline_name]   query   string false "Filter"
// @Param    filter[os]              query   string    false "Filter OS version"
// @Param    tags                    query   []string  false "Tag filter"
// @Success 200 {array} SystemDBLookupV3
// @Failure 415 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /export/systems [get]
func SystemsExportHandler(c *gin.Context) {
	account := c.GetInt(middlewares.KeyAccount)
	apiver := c.GetInt(middlewares.KeyApiver)
	groups := c.GetStringMapString(middlewares.KeyInventoryGroups)
	db := middlewares.DBFromContext(c)
	query := querySystems(db, account, apiver, groups)
	filters, err := ParseInventoryFilters(c)
	if err != nil {
		return
	} // Error handled in method itself
	query, _ = ApplyInventoryFilter(filters, query, "sp.inventory_id")

	var systems SystemDBLookupSlice

	query = query.Order("sp.id")
	query, err = ExportListCommon(query, c, SystemOpts)
	if err != nil {
		return
	} // Error handled in method itself

	err = query.Find(&systems).Error
	if err != nil {
		LogAndRespError(c, err, "db error")
		return
	}

	systems.ParseAndFillTags()
	if apiver < 3 {
		dataV2 := systemDBLookups2SystemDBLookupsV2(systems)
		OutputExportData(c, dataV2)
		return
	}
	dataV3 := systemDBLookups2SystemDBLookupsV3(systems)
	OutputExportData(c, dataV3)
}

func systemDBLookups2SystemDBLookupsV2(data []SystemDBLookup) []SystemDBLookupV2 {
	res := make([]SystemDBLookupV2, 0, len(data))
	for _, x := range data {
		res = append(res, SystemDBLookupV2{
			SystemDBLookupCommon: x.SystemDBLookupCommon,
			SystemItemAttributesV2: SystemItemAttributesV2{
				x.SystemItemAttributesCommon, x.SystemItemAttributesV2Only,
			},
		})
	}
	return res
}

func systemDBLookups2SystemDBLookupsV3(data []SystemDBLookup) []SystemDBLookupV3 {
	res := make([]SystemDBLookupV3, 0, len(data))
	for _, x := range data {
		res = append(res, SystemDBLookupV3{
			SystemDBLookupCommon: x.SystemDBLookupCommon,
			SystemItemAttributesV3: SystemItemAttributesV3{
				x.SystemItemAttributesCommon, x.SystemItemAttributesV3Only,
			},
		})
	}
	return res
}
