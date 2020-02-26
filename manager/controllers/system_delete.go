package controllers

import (
	"app/base/database"
	"app/base/utils"
	"app/manager/middlewares"
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
)

// @Summary Delete system by inventory id
// @Description Delete system by inventory id
// @ID deletesystem
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    inventory_id    path    string   true "Inventory ID"
// @Success 200
// @Router /api/patch/v1/systems/{inventory_id} [delete]
func SystemDeleteHandler(c *gin.Context) {
	account := c.GetString(middlewares.KeyAccount)

	inventoryID := c.Param("inventory_id")
	if inventoryID == "" {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{Error: "inventory_id param not found"})
		return
	}
	var systemInventoryID []string
	err := database.Db.Set("gorm:query_option", "FOR UPDATE").
		Table("system_platform").
		Joins("inner join rh_account ra on system_platform.rh_account_id = ra.id").
		Where("ra.name = ?", account).
		Where("inventory_id = ?", inventoryID).
		Pluck("inventory_id", &systemInventoryID).Error

	if err != nil {
		LogAndRespError(c, err, "could not query inventory")
		return
	}

	if len(systemInventoryID) == 0 {
		LogAndRespNotFound(c, errors.New("no rows returned"), "system not found")
		return
	}

	query := database.Db.Exec("select deleted_inventory_id from delete_system(?)", systemInventoryID[0])

	if query.Error != nil {
		LogAndRespError(c, err, "Could not delete system")
		return
	}
	if query.RowsAffected > 0 {
		c.Status(http.StatusOK)
	} else {
		c.Status(http.StatusNotFound)
	}
}
