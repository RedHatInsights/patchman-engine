package controllers

import (
	"app/base"
	"app/base/database"
	"app/base/models"
	"app/manager/middlewares"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
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

	baselineIDstr := c.Param("baseline_id")
	baselineID, err := strconv.Atoi(baselineIDstr)
	if err != nil {
		LogAndRespError(c, err, "Invalid baseline id: "+baselineIDstr)
		return
	}

	var exists int64
	err = database.Db.Model(&models.Baseline{}).
		Where("id = ? ", baselineID).Count(&exists).Error
	if err != nil {
		LogAndRespError(c, err, "Database error")
		return
	}
	if exists == 0 {
		LogAndRespNotFound(c, errors.New("Baseline not found"), "Baseline not found")
		return
	}

	missingIDs, err := checkInventoryIDs(account, req.InventoryIDs)
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

type SystemPlatform struct {
	*models.SystemPlatform
}

func checkInventoryIDs(accountID int, inventoryIDsMap map[string]bool) (missingIDs []string, err error) {
	inventoryIDs := make([]string, 0, len(inventoryIDsMap))
	for id := range inventoryIDsMap {
		inventoryIDs = append(inventoryIDs, id)
	}

	var containingIDs []string
	err = database.Db.Table("system_platform sp").
		Joins("JOIN inventory.hosts ih ON ih.id = sp.inventory_id").
		Where("rh_account_id = (?) AND inventory_id::text IN (?)", accountID, inventoryIDs).
		Pluck("sp.inventory_id", &containingIDs).Error
	if err != nil {
		return nil, err
	}

	if len(inventoryIDs) == len(containingIDs) {
		return []string{}, nil // all inventoryIDs found in database
	}

	containingIDsMap := map[string]bool{}
	for _, containingID := range containingIDs {
		containingIDsMap[containingID] = true
	}

	for _, inventoryID := range inventoryIDs {
		if _, ok := containingIDsMap[inventoryID]; !ok {
			missingIDs = append(missingIDs, inventoryID)
		}
	}
	sort.Strings(missingIDs)
	return missingIDs, nil
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
