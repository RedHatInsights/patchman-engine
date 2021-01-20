package controllers

import (
	"app/base/database"
	"app/base/utils"
	"app/manager/middlewares"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"net/http"
)

type SystemDetailResponse struct {
	Data SystemItem `json:"data"`
}

// @Summary Show me details about a system by given inventory id
// @Description Show me details about a system by given inventory id
// @ID detailSystem
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    inventory_id    path    string   true "Inventory ID"
// @Success 200 {object} SystemDetailResponse
// @Router /api/patch/v1/systems/{inventory_id} [get]
func SystemDetailHandler(c *gin.Context) {
	account := c.GetInt(middlewares.KeyAccount)

	inventoryID := c.Param("inventory_id")
	if inventoryID == "" {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{Error: "inventory_id param not found"})
		return
	}

	var systemItemAttributes SystemItemAttributes
	query := database.Systems(database.Db, account).
		Select(database.MustGetSelect(&systemItemAttributes)).
		Where("sp.inventory_id = ?::uuid", inventoryID)

	if applyInventoryHosts {
		query = query.Joins("JOIN inventory.hosts ih ON ih.id = inventory_id")
	}

	err := query.First(&systemItemAttributes).Error
	if gorm.IsRecordNotFoundError(err) {
		LogAndRespNotFound(c, err, "inventory not found")
		return
	}

	if err != nil {
		LogAndRespError(c, err, "database error")
		return
	}

	var resp = SystemDetailResponse{
		Data: SystemItem{
			Attributes: systemItemAttributes,
			ID:         inventoryID,
			Type:       "system",
		}}
	c.JSON(http.StatusOK, &resp)
}
