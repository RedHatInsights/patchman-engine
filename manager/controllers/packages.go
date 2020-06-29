package controllers

import (
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"app/manager/middlewares"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/dialects/postgres"
	"net/http"
)

type SystemPackagesAttrs struct {
	Name             string `json:"name" query:"p.name"`
	VersionInstalled string `json:"version_installed" query:"spkg.version_installed"`
	Summary          string `json:"summary" query:"p.summary"`
	Description      string `json:"description" query:"p.description"`
}

type SystemPackageData struct {
	SystemPackagesAttrs
	Updates models.PackageUpdates `json:"updates"`
}
type SystemPackageResponse []SystemPackageData

var PackagesSelect = fmt.Sprintf("%s,spkg.update_data as updates", database.MustGetSelect(&SystemPackagesAttrs{}))

//var PackagesAttrs = database.MustGetQueryAttrs(&SystemPackageData{})

type SystemPackageDBLoad struct {
	SystemPackagesAttrs
	Updates postgres.Jsonb `json:"updates" query:"spkg.update_data"`
}

func systemPackageQuery(account string, inventoryID string) *gorm.DB {
	return database.Db.
		Table("system_package spkg").
		Joins("inner join system_platform sp on sp.id = spkg.system_id").
		Joins("inner join rh_account ra on sp.rh_account_id = ra.id").
		Joins("inner join package p on p.id = spkg.package_id").
		Where("ra.name = ? and sp.inventory_id = ?", account, inventoryID)
}

// @Summary Show me details about a system packages by given inventory id
// @Description Show me details about a system packages by given inventory id
// @ID systemPackages
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    inventory_id    path    string   true "Inventory ID"
// @Success 200 {object} SystemPackageResponse
// @Router /api/patch/v1/systems/{inventory_id}/packages [get]
func SystemPackagesHandler(c *gin.Context) {
	account := c.GetString(middlewares.KeyAccount)

	inventoryID := c.Param("inventory_id")
	if inventoryID == "" {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{Error: "inventory_id param not found"})
		return
	}

	var loaded []SystemPackageDBLoad
	q := systemPackageQuery(account, inventoryID).Select(PackagesSelect)
	err := q.Find(&loaded).Error
	if gorm.IsRecordNotFoundError(err) {
		LogAndRespNotFound(c, err, "loaded not found")
		return
	}

	if err != nil {
		LogAndRespError(c, err, "database error")
		return
	}
	resp := make(SystemPackageResponse, len(loaded))
	for i, sp := range loaded {
		resp[i].SystemPackagesAttrs = sp.SystemPackagesAttrs
		if sp.Updates.RawMessage == nil {
			continue
		}
		if err := json.Unmarshal(sp.Updates.RawMessage, &resp[i].Updates); err != nil {
			panic(err)
		}
	}

	c.JSON(200, loaded)
}
