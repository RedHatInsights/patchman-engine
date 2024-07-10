package controllers

import (
	"app/base/utils"
	"app/manager/middlewares"

	"github.com/gin-gonic/gin"
)

// @Summary Export systems for my account
// @Description  Export systems for my account. Export endpoints are not paginated.
// @ID exportSystems
// @Security RhIdentity
// @Accept   json
// @Produce  json,text/csv
// @Param    search         query   string  false   "Find matching text"
// @Param    filter[id]              query   string  false "Filter"
// @Param    filter[display_name]    query   string  false "Filter"
// @Param    filter[last_evaluation] query   string  false   "Filter"
// @Param    filter[last_upload]     query   string  false "Filter"
// @Param    filter[rhsa_count]      query   string  false "Filter"
// @Param    filter[rhba_count]      query   string  false "Filter"
// @Param    filter[rhea_count]      query   string  false "Filter"
// @Param    filter[other_count]     query   string  false "Filter"
// @Param    filter[installable_rhsa_count]  query   string  false   "Filter"
// @Param    filter[installable_rhba_count]  query   string  false   "Filter"
// @Param    filter[installable_rhea_count]  query   string  false   "Filter"
// @Param    filter[installable_other_count] query   string  false   "Filter"
// @Param    filter[applicable_rhsa_count]   query   string  false   "Filter"
// @Param    filter[applicable_rhba_count]   query   string  false   "Filter"
// @Param    filter[applicable_rhea_count]   query   string  false   "Filter"
// @Param    filter[applicable_other_count]  query   string  false   "Filter"
// @Param    filter[stale]           query   string  false "Filter"
// @Param    filter[packages_installed]   query string false "Filter"
// @Param    filter[packages_installable] query   string  false   "Filter"
// @Param    filter[packages_applicable]  query   string  false   "Filter"
// @Param    filter[group_name] 									query []string 	false "Filter systems by inventory groups"
// @Param    filter[system_profile][sap_system]						query string  	false "Filter only SAP systems"
// @Param    filter[system_profile][sap_sids]						query []string  false "Filter systems by their SAP SIDs"
// @Param    filter[system_profile][ansible]						query string 	false "Filter systems by ansible"
// @Param    filter[system_profile][ansible][controller_version]	query string 	false "Filter systems by ansible version"
// @Param    filter[system_profile][mssql]							query string 	false "Filter systems by mssql version"
// @Param    filter[system_profile][mssql][version]					query string 	false "Filter systems by mssql version"
// @Param    filter[baseline_name]   query   string false "Filter"
// @Param    filter[template_name]   query   string false "Filter"
// @Param    filter[template_uuid]   query   string false "Filter"
// @Param    filter[arch]            query   string false "Filter"
// @Param    filter[os]              query   string    false "Filter OS version"
// @Param    filter[osname]          query   string  false   "Filter OS name"
// @Param    filter[osmajor]         query   string  false   "Filter OS major version"
// @Param    filter[osminor]         query   string  false   "Filter OS minor version"
// @Param    tags                    query   []string  false "Tag filter"
// @Success 200 {array} SystemDBLookup
// @Failure 415 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /export/systems [get]
func SystemsExportHandler(c *gin.Context) {
	account := c.GetInt(utils.KeyAccount)
	groups := c.GetStringMapString(utils.KeyInventoryGroups)
	db := middlewares.DBFromContext(c)
	query := querySystems(db, account, groups)
	filters, err := ParseAllFilters(c, SystemOpts)
	if err != nil {
		return
	} // Error handled in method itself
	query, _ = ApplyInventoryFilter(filters, query, "sp.inventory_id")

	var systems []SystemDBLookupExtended

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

	data := systemDBLookupsExtended2SystemDBLookups(systems)
	OutputExportData(c, data)
}

func systemDBLookupsExtended2SystemDBLookups(data []SystemDBLookupExtended) []SystemDBLookup {
	res := make([]SystemDBLookup, 0, len(data))
	for _, x := range data {
		res = append(res, SystemDBLookup{
			SystemDBLookupCommon: x.SystemDBLookupCommon,
			SystemItemAttributes: x.SystemItemAttributesExtended.SystemItemAttributes,
		})
	}
	return res
}
