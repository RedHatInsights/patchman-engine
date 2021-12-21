package controllers

import (
	"app/base"
	"app/base/database"
	"app/base/models"
	"app/manager/middlewares"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
)

type CreateBaselineRequst struct {
	Name         string     `json:"name"`
	InventoryIDs []SystemID `json:"inventoryIDs"`
	ToTime       string     `json:"toTime"`
}

type BaselineConfig struct {
	ToTime string `json:"to_time"`
}

type AccountID struct {
	RhAccountID int
}
type CreateBaselineResponse struct {
	BaselineID int
}

// @Summary Create a baseline for my set of systems
// @Description Create a baseline for my set of systems
// @ID createBaseline
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    body    body    CreateBaselineRequst true "Request body"
// @Success 200 {object} CreateBaselineResponse
// @Router /api/patch/v1/baselines [put]
func CreateBaselineHandler(c *gin.Context) {
	account := AccountID{
		RhAccountID: c.GetInt(middlewares.KeyAccount),
	}

	var req CreateBaselineRequst
	if err := c.ShouldBindJSON(&req); err != nil {
		LogAndRespBadRequest(c, err, "Invalid request body")
	}

	baselineConfig := BaselineConfig{
		ToTime: req.ToTime,
	}

	config, err := json.Marshal(&baselineConfig)
	if err != nil {
		LogAndRespError(c, err, "Invalid config")
	}

	baseline := models.Baseline{
		Name:        req.Name,
		Config:      config,
		RhAccountID: account.RhAccountID,
	}

	baselineID, err := buildCreateBaselineQuery(baseline, req.InventoryIDs, account)
	if err != nil {
		LogAndRespError(c, err, "Database error")
	}

	c.JSON(http.StatusOK, baselineID)
}

func buildCreateBaselineQuery(baseline models.Baseline, inventoryIDs []SystemID,
	account AccountID) (int, error) {
	tx := database.Db.WithContext(base.Context).Begin()
	defer tx.Rollback()

	if err := tx.Model(models.Baseline{}).Create(&baseline).Error; err != nil {
		return baseline.ID, err
	}

	if err := tx.Model(models.SystemPlatform{}).
		Joins("JOIN inventory.hosts ih ON ih.id = sp.inventory_id").
		Where("rh_account_id = (?) AND inventory_id::text IN (?)", account.RhAccountID, inventoryIDs).
		Update("baseline_id", baseline.ID).Error; err != nil {
		return baseline.ID, err
	}

	query := tx.Commit()

	return baseline.ID, query.Error
}
