package controllers

import (
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"app/base/vmaas"
	"app/manager/middlewares"
	"encoding/json"
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
	GroupsStrHelper
}

type SystemVmaasJSONResponse struct {
	Data vmaas.UpdatesV3Request `json:"data"`
}

type SystemYumUpdatesResponse struct {
	Data vmaas.UpdatesV2Response `json:"data"`
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
	groups := c.GetStringMapString(middlewares.KeyInventoryGroups)

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
	query := database.Systems(db, account, groups).
		Select(database.MustGetSelect(&systemDetail)).
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

	if err := parseSystemItems(systemDetail.TagsStr, &systemDetail.Tags); err != nil {
		utils.LogDebug("err", err.Error(), "inventory_id", inventoryID, "system tags parsing failed")
	}

	if err := parseSystemItems(systemDetail.GroupsStr, &systemDetail.Groups); err != nil {
		utils.LogDebug("err", err.Error(), "inventory_id", inventoryID, "system groups parsing failed")
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

// @Summary Show me system's json request for VMaaS
// @Description Show me system's json request for VMaaS
// @ID systemVmaasJson
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    inventory_id    path    string   true "Inventory ID"
// @Success 200 {object} SystemVmaasJSONResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /systems/{inventory_id}/vmaas_json [get]
func SystemVmaasJSONHandler(c *gin.Context) {
	system := systemJSONsCommon(c, "sp.vmaas_json")
	if system == nil {
		return // abort handled by `systemJSONsCommon`
	}
	data, err := utils.ParseVmaasJSON(system)
	if err != nil {
		LogAndRespError(c, err, "couldn't parse vmaas json")
		return
	}

	resp := SystemVmaasJSONResponse{data}
	c.JSON(http.StatusOK, &resp)
}

// @Summary Show me system's yum_updates (client side evaluation data)
// @Description Show me system's yum_updates (client side evaluation data)
// @ID systemYumUpdates
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    inventory_id    path    string   true "Inventory ID"
// @Success 200 {object} SystemYumUpdatesResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /systems/{inventory_id}/yum_updates [get]
func SystemYumUpdatesHandler(c *gin.Context) {
	system := systemJSONsCommon(c, "sp.yum_updates")
	if system == nil {
		return // abort handled by `systemJSONsCommon`
	}

	var resp SystemYumUpdatesResponse
	if system.YumUpdates == nil {
		c.JSON(http.StatusOK, &resp)
		return
	}

	err := json.Unmarshal(system.YumUpdates, &resp.Data)
	if err != nil {
		LogAndRespError(c, err, "unable to unmarshall yum updates")
		return
	}

	c.JSON(http.StatusOK, &resp)
}

func systemJSONsCommon(c *gin.Context, column string) *models.SystemPlatform {
	account := c.GetInt(middlewares.KeyAccount)
	groups := c.GetStringMapString(middlewares.KeyInventoryGroups)

	inventoryID := c.Param("inventory_id")
	if inventoryID == "" {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{Error: "inventory_id param not found"})
		return nil
	}

	if !utils.IsValidUUID(inventoryID) {
		LogAndRespBadRequest(c, errors.New("bad request"), "incorrect inventory_id format")
		return nil
	}

	if !isFilterInURLValid(c) {
		return nil
	}

	var system models.SystemPlatform
	db := middlewares.DBFromContext(c)
	query := database.Systems(db, account, groups).
		Select(column).
		Where("sp.inventory_id = ?::uuid", inventoryID)

	err := query.Take(&system).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		LogAndRespNotFound(c, err, "inventory not found")
		return nil
	}

	if err != nil {
		LogAndRespError(c, err, "database error")
		return nil
	}
	return &system
}
