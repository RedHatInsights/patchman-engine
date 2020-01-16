package controllers

import (
	"app/base/core"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"app/manager/middlewares"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"net/http"
)

type SystemAdvisoriesResponse struct {
	Data  []AdvisoryItem `json:"data"` // advisories items
	Links Links          `json:"links"`
	Meta  AdvisoryMeta   `json:"meta"`
}

// @Summary Show me advisories for a system by given inventory id
// @Description Show me advisories for a system by given inventory id
// @ID listSystemAdvisories
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    inventory_id    path    string   true "Inventory ID"
// @Success 200 {object} SystemAdvisoriesResponse
// @Router /api/patch/v1/systems/{inventory_id}/advisories [get]
func SystemAdvisoriesHandler(c *gin.Context) {
	account := c.GetString(middlewares.KeyAccount)

	limit, offset, err := utils.LoadLimitOffset(c, core.DefaultLimit)
	if err != nil {
		LogAndRespBadRequest(c, err, err.Error())
		return
	}

	inventoryID := c.Param("inventory_id")
	if inventoryID == "" {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{Error: "inventory_id param not found"})
		return
	}

	query := database.SystemAdvisoriesQueryName(database.Db, inventoryID).
		Joins("inner join rh_account ra on sp.rh_account_id = ra.id").
		Where("ra.name = ?", account)

	query, err = ApplySort(c, query)
	if err != nil {
		LogAndRespError(c, err, err.Error())
		return
	}

	var total int
	err = query.Count(&total).Error
	if err != nil {
		LogAndRespError(c, err, "error getting items count from db")
		return
	}

	if offset > total {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{Error: "too big offset"})
		return
	}

	var dbItems []models.SystemAdvisories
	err = query.Limit(limit).Offset(offset).Preload("Advisory").Find(&dbItems).Error
	if gorm.IsRecordNotFoundError(err) {
		LogAndRespNotFound(c, err, "no systems found")
		return
	}

	if err != nil {
		LogAndRespError(c, err, "database error")
		return
	}

	data := buildSystemAdvisoriesData(&dbItems)
	meta := buildAdvisoriesMeta(limit, offset, total)
	links := CreateLinks("/api/patch/v1/systems/$INVENTORY_ID/advisories", offset, limit, total,
		"&data_format=json")
	var resp = SystemAdvisoriesResponse{
		Data:  *data,
		Links: links,
		Meta:  *meta,
	}
	c.JSON(http.StatusOK, &resp)
}

func buildSystemAdvisoriesData(models *[]models.SystemAdvisories) *[]AdvisoryItem {
	var data []AdvisoryItem
	for _, systemAdvisories := range *models {
		advisory := systemAdvisories.Advisory
		item := AdvisoryItem{ID: advisory.Name, Type: "advisory", Attributes: AdvisoryItemAttributes{
			Description: advisory.Description, Severity: "", PublicDate: advisory.PublicDate, Synopsis: advisory.Synopsis,
			AdvisoryType: advisory.AdvisoryTypeID, ApplicableSystems: 0,
		}}
		data = append(data, item)
	}
	return &data
}
