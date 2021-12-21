package controllers

import (
	"app/base"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"app/manager/middlewares"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

// @Summary Delete a baseline
// @Description Delete a baseline
// @ID baselineDelete
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param baseline_id path int true "Baseline ID"
// @Success 200 "Ok"
// @Router /api/patch/v1/baselines/{baseline_id} [delete]
func BaselineDeleteHandler(c *gin.Context) {
	account := c.GetInt(middlewares.KeyAccount)

	baselineID := c.Param("baseline_id")
	if baselineID == "" {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{Error: "baseline_id param not found"})
		return
	}

	tx := database.Db.WithContext(base.Context).Begin()

	defer tx.Rollback()

	systemsQuery := tx.Model(models.SystemPlatform{}).
		Joins("JOIN inventory.hosts ih ON ih.id = inventory_id").
		Where("rh_account_id = ? AND baseline_id = ?", account, baselineID).
		Where("stale = false").
		Select("baseline_id").
		Updates(&models.SystemPlatform{BaselineID: nil})

	if systemsQuery.Error != nil {
		LogAndRespError(c, systemsQuery.Error, "Could not delete system")
		return
	}

	deleteQuery := tx.Delete(&models.Baseline{}, baselineID).Where("rh_account_id = (?)", account)

	if deleteQuery.Error != nil {
		LogAndRespError(c, deleteQuery.Error, "Could not delete system")
		return
	}

	if tx.Commit().Error != nil {
		LogAndRespError(c, deleteQuery.Error, "Could not delete system")
		return
	}

	if deleteQuery.RowsAffected > 0 {
		c.Status(http.StatusOK)
	} else {
		LogAndRespNotFound(c, errors.New("no rows returned"), "baseline not found")
		return
	}
}
