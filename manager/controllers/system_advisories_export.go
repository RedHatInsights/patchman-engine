package controllers

import (
	"app/base/models"
	"app/base/utils"
	"app/manager/middlewares"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

// @Summary Export applicable advisories for all my systems
// @Description  Export applicable advisories for all my systems
// @ID exportSystemAdvisories
// @Security RhIdentity
// @Accept   json
// @Produce  json,text/csv
// @Param    inventory_id   path    string  true    "Inventory ID"
// @Param    search         query   string  false   "Find matching text"
// @Param    filter[id]                  query   string  false "Filter"
// @Param    filter[description]         query   string  false "Filter"
// @Param    filter[public_date]         query   string  false "Filter"
// @Param    filter[synopsis]            query   string  false "Filter"
// @Param    filter[advisory_type]       query   string  false "Filter"
// @Param    filter[advisory_type_name]  query   string  false "Filter"
// @Param    filter[severity]            query   string  false "Filter"
// @Success 200 {array} SystemAdvisoriesDBLookup
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 415 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /export/systems/{inventory_id}/advisories [get]
func SystemAdvisoriesExportHandler(c *gin.Context) {
	account := c.GetInt(middlewares.KeyAccount)

	inventoryID := c.Param("inventory_id")
	if inventoryID == "" {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{Error: "inventory_id param not found"})
		return
	}

	if !utils.IsValidUUID(inventoryID) {
		LogAndRespBadRequest(c, errors.New("bad request"), "incorrect inventory_id format")
		return
	}

	db := middlewares.DBFromContext(c)
	var exists int64
	err := db.Model(&models.SystemPlatform{}).Where("inventory_id = ?::uuid ", inventoryID).
		Count(&exists).Error

	if err != nil {
		LogAndRespError(c, err, "database error")
	}
	if exists == 0 {
		LogAndRespNotFound(c, errors.New("System not found"), "System not found")
		return
	}

	query := buildSystemAdvisoriesQuery(db, account, inventoryID)
	query = query.Order("id")
	query, err = ExportListCommon(query, c, SystemAdvisoriesOpts)
	if err != nil {
		// Error handling and setting of result code & content is done in ListCommon
		return
	}

	var advisories []SystemAdvisoriesDBLookup
	err = query.Find(&advisories).Error
	for i := 0; i < len(advisories); i++ {
		advisories[i].AdvisoryItemAttributesCommon =
			fillAdvisoryItemAttributeReleaseVersion(advisories[i].AdvisoryItemAttributesCommon)
	}
	if err != nil {
		LogAndRespError(c, err, "db error")
		return
	}

	OutputExportData(c, advisories)
}
