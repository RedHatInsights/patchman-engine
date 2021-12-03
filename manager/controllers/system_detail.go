package controllers

import (
	"app/base/database"
	"app/base/utils"
	"app/manager/middlewares"
	"errors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
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

	if !utils.IsValidUUID(inventoryID) {
		LogAndRespBadRequest(c, errors.New("bad request"), "incorrect inventory_id format")
		return
	}

	if !isFilterInURLValid(c) {
		return
	}

	var systemItemAttributes SystemItemAttributes
	query := database.Systems(database.Db, account).
		Select(database.MustGetSelect(&systemItemAttributes)).
		Joins("JOIN inventory.hosts ih ON ih.id = inventory_id").
		Joins("LEFT JOIN baseline bl ON sp.baseline_id = bl.id AND sp.rh_account_id = bl.rh_account_id").
		Where("sp.inventory_id = ?::uuid", inventoryID)

	err := query.Take(&systemItemAttributes).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
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
