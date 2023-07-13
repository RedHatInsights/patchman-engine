package controllers

import (
	"app/manager/middlewares"
	"fmt"

	"github.com/gin-gonic/gin"
)

// nolint: lll
// @Summary Export systems belonging to a baseline
// @Description  Export systems applicable to a baseline
// @ID exportBaselineSystems
// @Security RhIdentity
// @Accept   json
// @Produce  json,text/csv
// @Param    baseline_id    path    int     true    "Baseline ID"
// @Param    search         query   string  false   "Find matching text"
// @Param    filter[display_name]           query   string  false "Filter"
// @Param    filter[os]           			query   string  false "Filter"
// @Param    tags           query   []string  false "Tag filter"
// @Param    filter[system_profile][sap_system]						query string  	false "Filter only SAP systems"
// @Param    filter[system_profile][sap_sids][in]					query []string  false "Filter systems by their SAP SIDs"
// @Param    filter[system_profile][ansible]						query string 	false "Filter systems by ansible"
// @Param    filter[system_profile][ansible][controller_version]	query string 	false "Filter systems by ansible version"
// @Param    filter[system_profile][mssql]							query string 	false "Filter systems by mssql version"
// @Param    filter[system_profile][mssql][version]					query string 	false "Filter systems by mssql version"
// @Success 200 {array} BaselineSystemsDBLookup
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 415 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /export/baselines/{baseline_id}/systems [get]
func BaselineSystemsExportHandler(c *gin.Context) {
	account := c.GetInt(middlewares.KeyAccount)
	apiver := c.GetInt(middlewares.KeyApiver)
	groups := c.GetStringMapString(middlewares.KeyInventoryGroups)
	if apiver < 3 {
		err := fmt.Errorf("endpoint does not exist in v%d API, use API >= v3", apiver)
		LogAndRespNotFound(c, err, err.Error())
		return
	}

	query, err := queryBaselineSystems(c, account, apiver, groups)
	if err != nil {
		return
	} // Error handled in method itself

	query, err = ExportListCommon(query, c, BaselineSystemOpts)
	if err != nil {
		return
	} // Error handled in method itself

	var baselineSystems BaselineSystemsDBLookupSlice
	err = query.Find(&baselineSystems).Error
	if err != nil {
		LogAndRespError(c, err, err.Error())
		return
	}

	baselineSystems.ParseAndFillTags()
	OutputExportData(c, baselineSystems)
}
