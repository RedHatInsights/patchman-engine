package controllers

import (
	"app/base"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"app/manager/kafka"
	"app/manager/middlewares"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
)

const BaselineMissingNameErr = "missing required parameter 'name'"
const DuplicateBaselineNameErr = "patch template name already exists"

type CreateBaselineRequest struct {
	// Baseline name
	Name string `json:"name"`
	// Inventory IDs list of systems to associate with this baseline (optional).
	InventoryIDs []string `json:"inventory_ids"`
	// Baseline config to filter applicable advisories and package updates for the associated systems (optional).
	Config *BaselineConfig `json:"config"`
	// Description of the baseline (optional).
	Description *string `json:"description"`
}

type CreateBaselineResponse struct {
	BaselineID int `json:"baseline_id" example:"1"` // Updated baseline unique ID, it can not be changed
}

// @Summary Create a baseline for my set of systems
// @Description Create a baseline for my set of systems
// @ID createBaseline
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    body    body    CreateBaselineRequest true "Request body"
// @Success 200 {object} CreateBaselineResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /baselines [put]
func CreateBaselineHandler(c *gin.Context) {
	accountID := c.GetInt(middlewares.KeyAccount)

	var request CreateBaselineRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		LogAndRespBadRequest(c, err, "Invalid request body: "+err.Error())
		return
	}

	if request.Name == "" {
		LogAndRespBadRequest(c, errors.New(BaselineMissingNameErr), BaselineMissingNameErr)
		return
	}
	request.Description = utils.EmptyToNil(request.Description)

	missingIDs, err := checkInventoryIDs(accountID, request.InventoryIDs)
	if err != nil {
		LogAndRespError(c, err, "Database error")
		return
	}

	if len(missingIDs) > 0 {
		msg := fmt.Sprintf("Missing inventory_ids: %v", missingIDs)
		LogAndRespNotFound(c, errors.New(msg), msg)
		return
	}

	baselineID, err := buildCreateBaselineQuery(request, accountID)
	if err != nil {
		if database.IsPgErrorCode(err, database.PgErrorDuplicateKey) {
			LogAndRespBadRequest(c, err, DuplicateBaselineNameErr)
			return
		}
		LogAndRespError(c, err, "Database error")
		return
	}

	configUpdated := request.Config != nil
	inventoryIDs := kafka.GetInventoryIDsToEvaluate(&baselineID, accountID, configUpdated, nil)
	kafka.EvaluateBaselineSystems(inventoryIDs)

	resp := CreateBaselineResponse{BaselineID: baselineID}
	c.JSON(http.StatusOK, &resp)
}

func buildCreateBaselineQuery(request CreateBaselineRequest, accountID int) (int, error) {
	tx := database.Db.WithContext(base.Context).Begin()
	defer tx.Rollback()

	baseline := models.Baseline{
		RhAccountID: accountID,
		Name:        request.Name,
		Description: request.Description,
	}

	if request.Config != nil {
		config, err := json.Marshal(request.Config)
		if err != nil {
			return 0, err
		}
		baseline.Config = config
	}

	if err := tx.Model(models.Baseline{}).Create(&baseline).Error; err != nil {
		return baseline.ID, err
	}

	if len(request.InventoryIDs) > 0 {
		err := updateSystemsBaselineID(tx, accountID, request.InventoryIDs, &baseline.ID, nil)
		if err != nil {
			return baseline.ID, err
		}
	}

	err := tx.Commit().Error
	return baseline.ID, err
}

func checkInventoryIDs(accountID int, inventoryIDs []string) (missingIDs []string, err error) {
	var containingIDs []string
	err = database.Db.Table("system_platform sp").
		Where("rh_account_id = ? AND inventory_id::text IN (?)", accountID, inventoryIDs).
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
