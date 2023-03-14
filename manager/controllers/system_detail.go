package controllers

import (
	"app/base/database"
	"app/base/utils"
	"app/manager/middlewares"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SystemDetailResponseV2 struct {
	Data SystemItemV2 `json:"data"`
}

type SystemDetailResponse struct {
	// use SystemItem not SystemItemV3 to display more info about system
	Data SystemItem `json:"data"`
}

type SystemDetailLookup struct {
	SystemItemAttributesAll
	TagsStrHelper
}

// nolint: funlen
// @Summary Show me details about a system by given inventory id
// @Description Show me details about a system by given inventory id
// @ID detailSystem
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    inventory_id    path    string   true "Inventory ID"
// @Success 200 {object} SystemDetailResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /systems/{inventory_id} [get]
func SystemDetailHandler(c *gin.Context) {
	account := c.GetInt(middlewares.KeyAccount)
	apiver := c.GetInt(middlewares.KeyApiver)

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

	var systemDetail SystemDetailLookup
	db := middlewares.DBFromContext(c)
	query := database.Systems(db, account).
		Select(database.MustGetSelect(&systemDetail)).
		Joins("JOIN inventory.hosts ih ON ih.id = inventory_id").
		Joins("LEFT JOIN baseline bl ON sp.baseline_id = bl.id AND sp.rh_account_id = bl.rh_account_id").
		Where("sp.inventory_id = ?::uuid", inventoryID)

	err := query.Take(&systemDetail).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		LogAndRespNotFound(c, err, "inventory not found")
		return
	}

	if err != nil {
		LogAndRespError(c, err, "database error")
		return
	}

	systemDetail.Tags, err = parseSystemTags(systemDetail.TagsStr)
	if err != nil {
		utils.LogDebug("err", err.Error(), "inventory_id", inventoryID, "system tags parsing failed")
	}

	if apiver < 3 {
		resp := SystemDetailResponseV2{
			Data: SystemItemV2{
				Attributes: SystemItemAttributesV2{
					systemDetail.SystemItemAttributesCommon,
					systemDetail.SystemItemAttributesV2Only,
				},
				ID:   inventoryID,
				Type: "system",
			}}
		c.JSON(http.StatusOK, &resp)
		return
	}
	resp := SystemDetailResponse{
		Data: SystemItem{
			Attributes: systemDetail.SystemItemAttributesAll,
			ID:         inventoryID,
			Type:       "system",
		}}
	c.JSON(http.StatusOK, &resp)
}
