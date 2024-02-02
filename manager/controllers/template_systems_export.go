package controllers

import (
	"app/base/utils"

	"github.com/gin-gonic/gin"
)

// nolint: lll
// @Summary Export systems belonging to a template
// @Description  Export systems applicable to a template
// @ID exportTemplateSystems
// @Security RhIdentity
// @Accept   json
// @Produce  json,text/csv
// @Param    template_id                                         path  string   true  "Template ID"
// @Param    search                                              query string   false "Find matching text"
// @Param    filter[display_name]                                query string   false "Filter"
// @Param    filter[os]                                          query string   false "Filter"
// @Param    tags                                                query []string false "Tag filter"
// @Param    filter[group_name]                                  query []string false "Filter systems by inventory groups"
// @Param    filter[system_profile][sap_system]                  query string   false "Filter only SAP systems"
// @Param    filter[system_profile][sap_sids]                    query []string false "Filter systems by their SAP SIDs"
// @Param    filter[system_profile][ansible]                     query string 	false "Filter systems by ansible"
// @Param    filter[system_profile][ansible][controller_version] query string 	false "Filter systems by ansible version"
// @Param    filter[system_profile][mssql]                       query string 	false "Filter systems by mssql version"
// @Param    filter[system_profile][mssql][version]              query string 	false "Filter systems by mssql version"
// @Success 200 {array} TemplateSystemsDBLookup
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 415 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /export/templates/{template_id}/systems [get]
func TemplateSystemsExportHandler(c *gin.Context) {
	account := c.GetInt(utils.KeyAccount)
	groups := c.GetStringMapString(utils.KeyInventoryGroups)

	query, err := templateSystemsQuery(c, account, groups)
	if err != nil {
		return
	} // Error handled in method itself

	query, err = ExportListCommon(query, c, TemplateSystemOpts)
	if err != nil {
		return
	} // Error handled in method itself

	var templateSystems []TemplateSystemsDBLookup
	err = query.Find(&templateSystems).Error
	if err != nil {
		LogAndRespError(c, err, err.Error())
		return
	}

	OutputExportData(c, templateSystems)
}
