package controllers

import (
	"app/base"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"app/manager/middlewares"
	"net/http"

	"github.com/pkg/errors"

	"github.com/gin-gonic/gin"
)

const InvalidInventoryIDsErr = "invalid list of inventory IDs"

type BaselineSystemsRemoveRequest struct {
	// List of inventory IDs to have baselines removed
	InventoryIDs []string `json:"inventory_ids"`
}

// @Summary Remove systems from baseline
// @Description Remove systems from baseline
// @ID removeBaselineSystems
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    body    body   BaselineSystemsRemoveRequest true "Request body"
// @Success 200 {int}		http.StatusOK
// @Failure 400 {object} 	utils.ErrorResponse
// @Failure 404 {object} 	utils.ErrorResponse
// @Failure 500 {object} 	utils.ErrorResponse
// @Router /baselines/systems/remove [POST]
func BaselineSystemsRemoveHandler(c *gin.Context) {
	account := c.GetInt(middlewares.KeyAccount)

	var req BaselineSystemsRemoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		LogAndRespBadRequest(c, err, "Invalid request body: "+err.Error())
		return
	}

	err := buildBaselineSystemsRemoveQuery(req.InventoryIDs, account)
	if err != nil {
		switch e := err.Error(); e {
		case InvalidInventoryIDsErr:
			LogAndRespBadRequest(c, err, e)
		default:
			LogAndRespError(c, err, "database error")
		}
		return
	}

	c.Status(http.StatusOK)
}

func buildBaselineSystemsRemoveQuery(inventoryIDs []string,
	accountID int) error {
	if len(inventoryIDs) == 0 {
		return errors.New(InvalidInventoryIDsErr)
	}
	for _, invID := range inventoryIDs {
		if !utils.IsValidUUID(invID) {
			return errors.New(InvalidInventoryIDsErr)
		}
	}
	tx := database.Db.WithContext(base.Context).Begin()
	defer tx.Rollback()

	tx = tx.Model(models.SystemPlatform{}).
		Where("rh_account_id = ? AND "+
			"baseline_id is NOT NULL AND "+
			"inventory_id::uuid IN (?)",
			accountID, inventoryIDs).
		Update("baseline_id", nil)
	if e := tx.Error; e != nil {
		return e
	}
	if int(tx.RowsAffected) != len(inventoryIDs) {
		return errors.New(InvalidInventoryIDsErr)
	}

	err := tx.Commit().Error
	return err
}
