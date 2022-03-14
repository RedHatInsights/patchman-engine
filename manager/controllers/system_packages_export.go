package controllers

import (
	"app/base/models"
	"app/base/utils"
	"app/manager/middlewares"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
	"strings"
)

type SystemPackageInline struct {
	SystemPackagesAttrs
	LatestEVRA string `json:"latest_evra" csv:"latest_evra"`
}

// @Summary Show me details about a system packages by given inventory id
// @Description Show me details about a system packages by given inventory id
// @ID exportSystemPackages
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    inventory_id    path    string   true "Inventory ID"
// @Param    search          query   string  false   "Find matching text"
// @Param    filter[name]            query   string  false "Filter"
// @Param    filter[description]     query   string  false "Filter"
// @Param    filter[evra]            query   string  false "Filter"
// @Param    filter[summary]         query   string  false "Filter"
// @Param    filter[updatable]       query   bool    false "Filter"
// @Success 200 {array} SystemPackageInline
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 415 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/patch/v1/export/systems/{inventory_id}/packages [get]
func SystemPackagesExportHandler(c *gin.Context) {
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

	var loaded []SystemPackageDBLoad
	q := systemPackageQuery(account, inventoryID)
	q, err := ExportListCommon(q, c, SystemPackagesOpts)
	if err != nil {
		// Error handling and setting of result code & content is done in ListCommon
		return
	}

	err = q.Find(&loaded).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		LogAndRespNotFound(c, err, "inventory_id not found")
		return
	}

	if err != nil {
		LogAndRespError(c, err, "database error")
		return
	}

	data := convertToOutputArray(&loaded)
	accept := c.GetHeader("Accept")
	if strings.Contains(accept, "application/json") { // nolint: gocritic
		c.JSON(http.StatusOK, data)
	} else if strings.Contains(accept, "text/csv") {
		Csv(c, 200, data)
	} else {
		LogWarnAndResp(c, http.StatusUnsupportedMediaType,
			fmt.Sprintf("Invalid content type '%s', use 'application/json' or 'text/csv'", accept))
	}
}

func convertToOutputArray(inArr *[]SystemPackageDBLoad) *[]SystemPackageInline {
	outData := make([]SystemPackageInline, len(*inArr))
	for i, v := range *inArr {
		outData[i].SystemPackagesAttrs = v.SystemPackagesAttrs
		if v.Updates == nil {
			outData[i].LatestEVRA = v.SystemPackagesAttrs.EVRA
			continue
		}
		var updates []models.PackageUpdate
		if err := json.Unmarshal(v.Updates, &updates); err != nil {
			panic(err)
		}
		if len(updates) > 0 {
			outData[i].LatestEVRA = updates[len(updates)-1].EVRA
		}
	}
	return &outData
}
