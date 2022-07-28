package controllers

import (
	"app/base"
	"app/base/database"
	"app/base/models"
	"app/manager/kafka"
	"app/manager/middlewares"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const ForeignBaselineViolationErr = "unable to update systems of another baseline"

type BaselineConfig database.BaselineConfig

type UpdateBaselineRequest struct {
	// Updated baseline name (optional)
	Name *string `json:"name" example:"my-changed-baseline-name"`
	// Map of inventories to add to (true) or remove (false) from given baseline (optional)
	InventoryIDs map[string]bool `json:"inventory_ids"`
	// Updated baseline config (optional)
	Config *BaselineConfig `json:"config"`
	// Description of the baseline (optional).
	Description *string `json:"description,omitempty"`
}

type UpdateBaselineResponse struct {
	BaselineID int64 `json:"baseline_id" example:"1"` // Updated baseline unique ID, it can not be changed
}

// nolint: funlen
// @Summary Update a baseline for my set of systems
// @Description Update a baseline for my set of systems
// @ID updateBaseline
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    baseline_id path    int                  true "Baseline ID"
// @Param    body        body    UpdateBaselineRequest true "Request body"
// @Success 200 {object} UpdateBaselineResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /baselines/{baseline_id} [put]
func BaselineUpdateHandler(c *gin.Context) {
	account := c.GetInt64(middlewares.KeyAccount)

	var req UpdateBaselineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		LogAndRespBadRequest(c, err, "Invalid request body: "+err.Error())
		return
	}

	baselineIDstr := c.Param("baseline_id")
	baselineID, err := strconv.ParseInt(baselineIDstr, 10, 64)
	if err != nil {
		LogAndRespBadRequest(c, err, "Invalid baseline_id: "+baselineIDstr)
		return
	}

	var exists int64
	err = database.Db.Model(&models.Baseline{}).
		Where("id = ? AND rh_account_id = ?", baselineID, account).Count(&exists).Error
	if err != nil {
		LogAndRespError(c, err, "Database error")
		return
	}
	if exists == 0 {
		LogAndRespNotFound(c, errors.New("Baseline not found"), "Baseline not found")
		return
	}

	inventoryIDsList := map2list(req.InventoryIDs)
	missingIDs, err := checkInventoryIDs(account, inventoryIDsList)
	if err != nil {
		LogAndRespError(c, err, "Database error")
		return
	}

	if len(missingIDs) > 0 {
		msg := fmt.Sprintf("Missing inventory_ids: %v", missingIDs)
		LogAndRespNotFound(c, errors.New(msg), msg)
		return
	}

	newAssociations, obsoleteAssociations := sortInventoryIDs(req.InventoryIDs)
	err = buildUpdateBaselineQuery(baselineID, req, newAssociations, obsoleteAssociations, account)
	if err != nil {
		if e := err.Error(); e == ForeignBaselineViolationErr {
			LogAndRespBadRequest(c, err, "Invalid inventory IDs: "+e)
			return
		}
		if database.IsPgErrorCode(err, database.PgErrorDuplicateKey) {
			LogAndRespBadRequest(c, err, DuplicateBaselineNameErr)
			return
		}
		LogAndRespError(c, err, "Database error")
		return
	}

	inventoryAIDs := kafka.GetInventoryIDsToEvaluate(&baselineID, account, req.Config != nil, inventoryIDsList)
	kafka.EvaluateBaselineSystems(inventoryAIDs)

	resp := UpdateBaselineResponse{BaselineID: baselineID}
	c.JSON(http.StatusOK, &resp)
}

func map2list(m map[string]bool) []string {
	l := make([]string, 0, len(m))
	for key := range m {
		l = append(l, key)
	}
	return l
}

func sortInventoryIDs(inventoryIDs map[string]bool) (newIDs, obsoleteIDs []string) {
	for key, value := range inventoryIDs {
		if value {
			newIDs = append(newIDs, key)
		} else {
			obsoleteIDs = append(obsoleteIDs, key)
		}
	}
	return newIDs, obsoleteIDs
}

func updateSystemsBaselineID(tx *gorm.DB, rhAccountID int64, inventoryIDs []string,
	newBaselineID, oldBaselineID *int64) error {
	updateFields := map[string]interface{}{"baseline_id": newBaselineID, "unchanged_since": time.Now()}
	tx = tx.Model(models.SystemPlatform{}).
		Where("rh_account_id = (?) AND inventory_id::text IN (?)", rhAccountID, inventoryIDs)

	// oldBaselineID is used to prevent overwriting inventory IDs of another baseline
	if oldBaselineID != nil {
		tx = tx.Where("baseline_id = (?) OR baseline_id is NULL", oldBaselineID)
	}

	tx = tx.Updates(updateFields)
	if tx.Error != nil {
		return tx.Error
	}

	if int(tx.RowsAffected) < len(inventoryIDs) {
		return errors.New(ForeignBaselineViolationErr)
	}

	return nil
}

func buildUpdateBaselineQuery(baselineID int64, req UpdateBaselineRequest, newIDs, obsoleteIDs []string,
	account int64) error {
	data := map[string]interface{}{}
	if req.Name != nil {
		data["name"] = req.Name
	}

	if req.Config != nil {
		config, err := json.Marshal(req.Config)
		if err != nil {
			return err
		}
		data["config"] = config
	}

	if req.Description != nil {
		data["description"] = req.Description
	}

	tx := database.Db.WithContext(base.Context).Begin()
	defer tx.Rollback()

	if req.Name != nil || req.Config != nil || req.Description != nil {
		err := tx.Model(models.Baseline{}).
			Where("id = ? AND rh_account_id = ?", baselineID, account).
			Updates(&data).Error
		if err != nil {
			return err
		}
	}

	if len(newIDs) > 0 {
		err := updateSystemsBaselineID(tx, account, newIDs, &baselineID, nil)
		if err != nil {
			return err
		}
	}

	if len(obsoleteIDs) > 0 {
		err := updateSystemsBaselineID(tx, account, obsoleteIDs, nil, &baselineID)
		if err != nil {
			return err
		}
	}

	err := tx.Commit().Error
	return err
}
