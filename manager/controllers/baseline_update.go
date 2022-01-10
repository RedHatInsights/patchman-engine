package controllers

import (
	"app/base"
	"app/base/database"
	"app/base/models"
	"app/manager/middlewares"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type UpdateBaselineRequest struct {
	Name         string          `json:"name"`
	InventoryIDs map[string]bool `json:"inventory_ids"`
	Config       BaselineConfig  `json:"config"`
}

type BaselineConfig struct {
	ToTime string `json:"to_time"`
}

type UpdateBaselineResponse struct {
	BaselineID int
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
// @Router /api/patch/v1/baselines/{baseline_id} [post]
func BaselineUpdateHandler(c *gin.Context) {
	account := c.GetInt(middlewares.KeyAccount)

	var req UpdateBaselineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		LogAndRespBadRequest(c, err, "Invalid request body: "+err.Error())
		return
	}

	baselineID, err := strconv.Atoi(c.Param("baseline_id"))
	if err != nil {
		LogAndRespError(c, err, "Invalid baseline id: "+err.Error())
		return
	}

	var exists int64
	err = database.Db.Model(&models.Baseline{}).
		Where("id = ? ", baselineID).Count(&exists).Error
	if err != nil {
		LogAndRespError(c, err, "database error: "+err.Error())
		return
	}
	if exists == 0 {
		LogAndRespNotFound(c, errors.New("Baseline not found"), "Baseline not found")
		return
	}

	newAssociations, obsoleteAssociations, err := sortInventoryIDs(req.InventoryIDs)
	if err != nil {
		c.JSON(http.StatusNotFound, "System(s) do(es) not exist: "+err.Error())
		return
	}

	err = buildUpdateBaselineQuery(baselineID, req, newAssociations, obsoleteAssociations, account)
	if err != nil {
		LogAndRespError(c, err, "Database error: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, baselineID)
}

type SystemPlatform struct {
	*models.SystemPlatform
}

func checkInventoryID(inventoryID string) error {
	var exists int64
	err := database.Db.Model(&models.SystemPlatform{}).
		Where("inventory_id = ? ", inventoryID).Count(&exists).Error
	if err != nil {
		return err
	}
	return nil
}

func sortInventoryIDs(inventoryIDs map[string]bool) (newIDs, obsoleteIDs []string, err error) {
	for key, value := range inventoryIDs {
		err := checkInventoryID(key)
		if err != nil {
			return nil, nil, err
		}

		if value {
			newIDs = append(newIDs, key)
		} else {
			obsoleteIDs = append(obsoleteIDs, key)
		}
	}
	return newIDs, obsoleteIDs, nil
}

func updateSystemsBaselineID(tx *gorm.DB, rhAccountID int, inventoryIDs []string, baselineID interface{}) error {
	err := tx.Model(models.SystemPlatform{}).
		Joins("JOIN inventory.hosts ih ON ih.id = sp.inventory_id").
		Where("rh_account_id = (?) AND inventory_id::text IN (?)", rhAccountID, inventoryIDs).
		Update("baseline_id", baselineID).Error
	return err
}

//nolint:lll
func buildUpdateBaselineQuery(baselineID int, req UpdateBaselineRequest, newIDs, obsoleteIDs []string, account int) error {
	config, err := json.Marshal(&req.Config)
	if err != nil {
		return err
	}

	data := map[string]interface{}{}
	data["name"] = req.Name
	data["config"] = config

	tx := database.Db.WithContext(base.Context).Begin()
	defer tx.Rollback()

	err = tx.Model(models.Baseline{}).
		Where("id = ? AND rh_account_id = ?", baselineID, account).
		Updates(&data).Error
	if err != nil {
		return err
	}

	err = updateSystemsBaselineID(tx, account, newIDs, baselineID)
	if err != nil {
		return err
	}

	err = updateSystemsBaselineID(tx, account, obsoleteIDs, nil)
	if err != nil {
		return err
	}

	query := tx.Commit()

	return query.Error
}
