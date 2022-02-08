package controllers

import (
	"app/base"
	"app/base/database"
	"app/base/models"
	"app/manager/middlewares"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type BaselineConfig database.BaselineConfig

type UpdateBaselineRequest struct {
	// Updated baseline name (optional)
	Name *string `json:"name" example:"my-changed-baseline-name"`
	// Map of inventories to add to (true) or remove (false) from given baseline (optional)
	InventoryIDs map[string]bool `json:"inventory_ids"`
	// Updated baseline config (optional)
	Config *BaselineConfig `json:"config"`
}

type UpdateBaselineResponse struct {
	BaselineID int `example:"1"` // Updated baseline unique ID, it can not be changed
}

// @Summary Update a baseline for my set of systems
// @Description Update a baseline for my set of systems
// @ID updateBaseline
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    baseline_id path    int                  true "Baseline ID"
// @Param    body        body    UpdateBaselineRequest true "Request body"
// @Success 200 {object} UpdateBaselineResponse
// @Router /api/patch/v1/baselines/{baseline_id} [put]
func BaselineUpdateHandler(c *gin.Context) {
	account := c.GetInt(middlewares.KeyAccount)

	var req UpdateBaselineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		LogAndRespBadRequest(c, err, "Invalid request body: "+err.Error())
		return
	}

	baselineIDstr := c.Param("baseline_id")
	baselineID, err := strconv.Atoi(baselineIDstr)
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

	missingIDs, err := checkInventoryIDs(account, map2list(req.InventoryIDs))
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
		LogAndRespError(c, err, "Database error")
		return
	}

	c.JSON(http.StatusOK, baselineID)
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

func updateSystemsBaselineID(tx *gorm.DB, rhAccountID int, inventoryIDs []string, baselineID interface{}) error {
	err := tx.Model(models.SystemPlatform{}).
		Joins("JOIN inventory.hosts ih ON ih.id = sp.inventory_id").
		Where("rh_account_id = (?) AND inventory_id::text IN (?)", rhAccountID, inventoryIDs).
		Update("baseline_id", baselineID).Error
	return err
}

func buildUpdateBaselineQuery(baselineID int, req UpdateBaselineRequest, newIDs, obsoleteIDs []string,
	account int) error {
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

	tx := database.Db.WithContext(base.Context).Begin()
	defer tx.Rollback()

	if req.Name != nil || req.Config != nil {
		err := tx.Model(models.Baseline{}).
			Where("id = ? AND rh_account_id = ?", baselineID, account).
			Updates(&data).Error
		if err != nil {
			return err
		}
	}

	if len(newIDs) > 0 {
		err := updateSystemsBaselineID(tx, account, newIDs, baselineID)
		if err != nil {
			return err
		}
	}

	if len(obsoleteIDs) > 0 {
		err := updateSystemsBaselineID(tx, account, obsoleteIDs, nil)
		if err != nil {
			return err
		}
	}

	err := tx.Commit().Error
	return err
}
