package controllers

import (
	"app/base/utils"
	"app/manager/middlewares"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SystemPackageInline struct {
	SystemPackagesAttrs
	LatestInstallable string `json:"latest_installable" csv:"latest_installable"`
	LatestApplicable  string `json:"latest_applicable" csv:"latest_applicable"`
}

// @Summary Show me details about a system packages by given inventory id
// @Description Show me details about a system packages by given inventory id. Export endpoints are not paginated.
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
// @Router /export/systems/{inventory_id}/packages [get]
func SystemPackagesExportHandler(c *gin.Context) {
	account := c.GetInt(utils.KeyAccount)
	groups := c.GetStringMapString(utils.KeyInventoryGroups)

	inventoryID := c.Param("inventory_id")
	if inventoryID == "" {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{Error: "inventory_id param not found"})
		return
	}

	if !utils.IsValidUUID(inventoryID) {
		utils.LogAndRespBadRequest(c, errors.New("bad request"), "incorrect inventory_id format")
		return
	}

	var loaded []SystemPackageDBLoad
	db := middlewares.DBFromContext(c)
	q := systemPackageQuery(db, account, groups, inventoryID)
	q, err := ExportListCommon(q, c, SystemPackagesOpts)
	if err != nil {
		// Error handling and setting of result code & content is done in ListCommon
		return
	}

	err = q.Find(&loaded).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		utils.LogAndRespNotFound(c, err, "inventory_id not found")
		return
	}

	if err != nil {
		utils.LogAndRespError(c, err, "database error")
		return
	}

	data := buildSystemPackageInline(loaded)
	OutputExportData(c, data)
}

func buildSystemPackageInline(pkgs []SystemPackageDBLoad) []SystemPackageInline {
	data := make([]SystemPackageInline, len(pkgs))
	for i, v := range pkgs {
		data[i].SystemPackagesAttrs = v.SystemPackagesAttrs
		data[i].LatestInstallable = v.EVRA
		if len(v.InstallableEVRA) > 0 {
			data[i].LatestInstallable = v.InstallableEVRA
		}
		data[i].LatestApplicable = data[i].LatestInstallable
		if len(v.ApplicableEVRA) > 0 {
			data[i].LatestApplicable = v.ApplicableEVRA
		}
	}
	return data
}
