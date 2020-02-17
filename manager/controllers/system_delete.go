package controllers

import (
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"app/manager/middlewares"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
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

	del := database.Db.
		Joins("inner join rh_account ra on system_platform.rh_account_id = ra.id").
		Where("ra.name = ?", account).
		Where("inventory_id = ?", inventoryID).
		Delete(&models.SystemPlatform{})

	if gorm.IsRecordNotFoundError(del.Error) {
		LogAndRespNotFound(c, del.Error, "inventory not found")
		return
	}

	if del.RowsAffected > 0 {
		c.Status(http.StatusNoContent)
	} else {
		c.Status(http.StatusBadRequest)
	}
}
