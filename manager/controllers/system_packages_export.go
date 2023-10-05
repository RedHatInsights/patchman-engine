package controllers

import (
	"app/base/models"
	"app/base/utils"
	"app/manager/middlewares"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SystemPackageInlineV2 struct {
	SystemPackagesAttrsV2
	LatestEVRA string `json:"latest_evra" csv:"latest_evra"`
}

type SystemPackageInlineV3 struct {
	SystemPackagesAttrsV3
	LatestInstallable string `json:"latest_installable" csv:"latest_installable"`
	LatestApplicable  string `json:"latest_applicable" csv:"latest_applicable"`
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
// @Success 200 {array} SystemPackageInlineV3
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 415 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /export/systems/{inventory_id}/packages [get]
func SystemPackagesExportHandler(c *gin.Context) {
	account := c.GetInt(utils.KeyAccount)
	apiver := c.GetInt(utils.KeyApiver)
	groups := c.GetStringMapString(utils.KeyInventoryGroups)

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
	db := middlewares.DBFromContext(c)
	q := systemPackageQuery(db, account, groups, inventoryID)
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

	if apiver < 3 {
		data := buildSystemPackageInlineV2(loaded)
		OutputExportData(c, data)
		return
	}
	data := buildSystemPackageInlineV3(loaded)
	OutputExportData(c, data)
}

func findLatestEVRA(pkg SystemPackageDBLoad) (installable models.PackageUpdate, applicable models.PackageUpdate) {
	installable = models.PackageUpdate{
		EVRA:   pkg.EVRA,
		Status: "Installable",
	}
	applicable = models.PackageUpdate{
		EVRA:   pkg.EVRA,
		Status: "Applicable",
	}
	if pkg.Updates == nil {
		return
	}
	var updates models.PackageUpdateData
	if err := json.Unmarshal(pkg.Updates, &updates); err != nil {
		panic(err)
	}
	if updates.Installable != "" {
		installable.EVRA = updates.Installable
		applicable.EVRA = updates.Installable
	}
	if updates.Applicable != "" {
		applicable.EVRA = updates.Applicable
	}
	return
}

func buildSystemPackageInlineV2(pkgs []SystemPackageDBLoad) []SystemPackageInlineV2 {
	data := make([]SystemPackageInlineV2, len(pkgs))
	for i, v := range pkgs {
		data[i].SystemPackagesAttrsCommon = v.SystemPackagesAttrsCommon
		installable, _ := findLatestEVRA(v)
		data[i].LatestEVRA = installable.EVRA
	}
	return data
}

func buildSystemPackageInlineV3(pkgs []SystemPackageDBLoad) []SystemPackageInlineV3 {
	data := make([]SystemPackageInlineV3, len(pkgs))
	for i, v := range pkgs {
		data[i].SystemPackagesAttrsV3 = v.SystemPackagesAttrsV3
		latestInstallable, latestApplicable := findLatestEVRA(v)
		data[i].LatestInstallable = latestInstallable.EVRA
		data[i].LatestApplicable = latestApplicable.EVRA
	}
	return data
}
