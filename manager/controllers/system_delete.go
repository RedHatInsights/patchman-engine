package controllers

import (
	"app/base"
	"app/base/database"
	"app/base/utils"
	"app/manager/middlewares"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// @Summary Delete system by inventory id
// @Description Delete system by inventory id
// @ID deletesystem
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    inventory_id    path    string   true "Inventory ID"
// @Success 200 {int}		http.StatusOK
// @Failure 400 {object} 	utils.ErrorResponse
// @Failure 404 {object} 	utils.ErrorResponse
// @Failure 500 {object} 	utils.ErrorResponse
// @Router /systems/{inventory_id} [delete]
func SystemDeleteHandler(c *gin.Context) {
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

	var systemInventoryID []string
	tx := database.Db.WithContext(base.Context).Begin()

	defer tx.Rollback()

	err := tx.Set("gorm:query_option", "FOR UPDATE OF system_platform").
		Table("system_platform").
		Where("rh_account_id = ?", account).
		Where("inventory_id = ?::uuid", inventoryID).
		Pluck("inventory_id", &systemInventoryID).Error

	if err != nil {
		LogAndRespError(c, err, "could not query database for system")
		return
	}

	if len(systemInventoryID) == 0 {
		LogAndRespNotFound(c, errors.New("no rows returned"), "system not found")
		return
	}

	query := tx.Exec("select deleted_inventory_id from delete_system(?::uuid)", systemInventoryID[0])

	if query.Error != nil {
		LogAndRespError(c, err, "Could not delete system")
		return
	}

	if tx.Commit().Error != nil {
		LogAndRespError(c, err, "Could not delete system")
		return
	}

	if query.RowsAffected > 0 {
		c.Status(http.StatusOK)
	} else {
		LogAndRespNotFound(c, errors.New("no rows returned"), "system not found")
		return
	}
}
