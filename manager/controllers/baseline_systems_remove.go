package controllers

import (
	"app/base"
	"app/base/database"
	"app/base/models"
	"app/manager/middlewares"
	"net/http"

	"github.com/pkg/errors"

	"github.com/gin-gonic/gin"
)

const InvalidInventoryIDsErr = "invalid list of inventory IDs"

type InventoryDBLookUp struct {
	InventoryID string `gorm:"column:inventory_id"`
}

type BaselineSystemsRemoveRequest struct {
	// List of inventory IDs to have baselines removed
	InventoryIDs []string `json:"inventory_ids"`
}

type BaselineSystemsRemoveResponse struct {
	// List of inventory IDs with successfully removed baselines
	InventoryIDs []string `json:"inventory_ids"`
}

// @Summary Remove systems from baseline
// @Description Remove systems from baseline
// @ID removeBaselineSystems
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    body    body   BaselineSystemsRemoveRequest true "Request body"
// @Success 200 {object}	BaselineSystemsRemoveResponse
// @Failure 400 {object} 	utils.ErrorResponse
// @Failure 404 {object} 	utils.ErrorResponse
// @Failure 500 {object} 	utils.ErrorResponse
// @Router /api/patch/v1/baselines/systems/remove [POST]
func BaselineSystemsRemoveHandler(c *gin.Context) {
	account := c.GetInt(middlewares.KeyAccount)

	var req BaselineSystemsRemoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		LogAndRespBadRequest(c, err, "Invalid request body: "+err.Error())
		return
	}

	resp, err := buildBaselineSystemsRemoveQuery(req.InventoryIDs, account)
	if err != nil {
		switch e := err.Error(); e {
		case InvalidInventoryIDsErr:
			LogAndRespBadRequest(c, err, e)
		default:
			LogAndRespError(c, err, "database error")
		}
		return
	}

	c.JSON(http.StatusOK, resp)
}

func buildBaselineSystemsRemoveQuery(inventoryIDs []string,
	accountID int) (*BaselineSystemsRemoveResponse, error) {
	tx := database.Db.WithContext(base.Context).Begin()
	defer tx.Rollback()

	resp := &BaselineSystemsRemoveResponse{make([]string, 0)}
	var ivs []InventoryDBLookUp

	tx = tx.Model(models.SystemPlatform{}).
		Where("rh_account_id = ? AND "+
			"baseline_id is NOT NULL AND "+
			"inventory_id::uuid IN (?)",
			accountID, inventoryIDs).
		Find(&ivs).
		Update("baseline_id", nil)
	if tx.Error != nil {
		return nil, tx.Error
	}
	if int(tx.RowsAffected) != len(inventoryIDs) {
		return nil, errors.New(InvalidInventoryIDsErr)
	}

	for _, i := range ivs {
		resp.InventoryIDs = append(resp.InventoryIDs, i.InventoryID)
	}

	err := tx.Commit().Error
	return resp, err
}
