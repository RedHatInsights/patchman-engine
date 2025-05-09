package controllers

import (
	"app/base/models"
	"app/base/utils"
	"app/manager/kafka"
	"app/manager/middlewares"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

const InvalidBaselineID = "Invalid baseline_id: "

type DeleteBaselineResponse struct {
	BaselineID int64 `json:"baseline_id" example:"1"` // Updated baseline unique ID, it can not be changed
}

// @Summary Delete a baseline
// @Description Delete a baseline
// @ID baselineDelete
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param baseline_id path int true "Baseline ID"
// @Success 200 {object} DeleteBaselineResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /baselines/{baseline_id} [delete]
// @Deprecated
func BaselineDeleteHandler(c *gin.Context) {
	account := c.GetInt(utils.KeyAccount)

	baselineIDstr := c.Param("baseline_id")
	baselineID, err := strconv.ParseInt(baselineIDstr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{Error: InvalidBaselineID + baselineIDstr})
		return
	}

	tx := middlewares.DBFromContext(c).Begin()
	defer tx.Rollback()

	inventoryAIDs := kafka.GetInventoryIDsToEvaluate(tx, &baselineID, account, true, nil)
	err = tx.Model(models.SystemPlatform{}).
		Where("rh_account_id = ? AND baseline_id = ?", account, baselineID).
		Select("baseline_id").
		Updates(&models.SystemPlatform{BaselineID: nil}).Error

	if err != nil {
		LogAndRespError(c, err, "Could not delete system")
		return
	}

	deleteQuery := tx.Where("rh_account_id = ? AND id = ?", account, baselineID).
		Delete(&models.Baseline{})
	err = deleteQuery.Error
	if err != nil {
		LogAndRespError(c, err, "Could not delete baseline")
		return
	}

	err = tx.Commit().Error
	if err != nil {
		LogAndRespError(c, errors.Wrap(err, "Could not commit baseline delete"), err.Error())
		return
	}

	if deleteQuery.RowsAffected == 0 {
		LogAndRespNotFound(c, errors.New("no rows returned"), "baseline not found")
		return
	}

	kafka.EvaluateBaselineSystems(inventoryAIDs)

	resp := DeleteBaselineResponse{BaselineID: baselineID}
	c.JSON(http.StatusOK, &resp)
}
