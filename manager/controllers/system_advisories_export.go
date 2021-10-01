package controllers

import (
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"app/manager/middlewares"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"net/http"
	"strings"
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
// @Router /api/patch/v1/export/systems/{inventory_id}/advisories [get]
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

	var exists int64
	err := database.Db.Model(&models.SystemPlatform{}).Where("inventory_id = ?::uuid ", inventoryID).
		Count(&exists).Error

	if err != nil {
		LogAndRespError(c, err, "database error")
	}
	if exists == 0 {
		LogAndRespNotFound(c, errors.New("System not found"), "System not found")
		return
	}

	query := buildSystemAdvisoriesQuery(account, inventoryID)
	query = query.Order("id")
	query, err = ExportListCommon(query, c, AdvisoriesOpts)
	if err != nil {
		// Error handling and setting of result code & content is done in ListCommon
		return
	}

	var advisories []SystemAdvisoriesDBLookup
	err = query.Find(&advisories).Error
	if err != nil {
		LogAndRespError(c, err, "db error")
		return
	}

	accept := c.GetHeader("Accept")
	if strings.Contains(accept, "application/json") { // nolint: gocritic
		c.JSON(http.StatusOK, advisories)
	} else if strings.Contains(accept, "text/csv") {
		Csv(c, http.StatusOK, advisories)
	} else {
		LogWarnAndResp(c, http.StatusUnsupportedMediaType,
			fmt.Sprintf("Invalid content type '%s', use 'application/json' or 'text/csv'", accept))
	}
}
