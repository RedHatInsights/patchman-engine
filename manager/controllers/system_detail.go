package controllers

import (
	"app/base/database"
	"app/base/models"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"net/http"
)

type SystemDetailResponse struct {
	Data  SystemItem     `json:"data"`
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
	inventoryId := c.Param("inventory_id")
	if inventoryId == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{"inventory_id param not found"})
		return
	}

	var inventory models.SystemPlatform
	err := database.Db.Where("inventory_id = ?", inventoryId).First(&inventory).Error
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
			Attributes: SystemItemAttributes{
				LastEvaluation: inventory.LastEvaluation,
				LastUpload: inventory.LastUpload,
				RhsaCount: inventory.AdvisoryCountCache,
				RhbaCount: 0,
				RheaCount: 0,
				Enabled: !inventory.OptOut,
			},
			Id:   inventory.InventoryID,
	        Type: "system",
		}}
	c.JSON(http.StatusOK, &resp)
	return
}
