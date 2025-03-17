package controllers

import (
	"app/base"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"app/manager/config"
	"app/manager/kafka"
	"app/manager/middlewares"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const BaselineMissingNameErr = "missing or invalid required parameter 'name'"
const DuplicateBaselineNameErr = "patch template name already exists"
const InvalidDescription = "invalid 'description'"

type CreateBaselineRequest struct {
	// Baseline name
	Name string `json:"name"`
	// Inventory IDs list of systems to associate with this baseline (optional).
	InventoryIDs []string `json:"inventory_ids"`
	// Baseline config to filter applicable advisories and package updates for the associated systems (optional).
	Config *BaselineConfig `json:"config"`
	// Description of the baseline (optional).
	Description *string `json:"description"`
	// Creator of the template
	Creator *string `json:"-"`
}

type CreateBaselineResponse struct {
	BaselineID int64 `json:"baseline_id" example:"1"` // Updated baseline unique ID, it can not be changed
}

type SystemBaselineDBLookup struct {
	InventoryID      string `query:"sp.inventory_id"`
	SatelliteManaged bool   `query:"sp.satellite_managed"`
	Bootc            bool   `query:"sp.bootc"`
}

// @Summary Create a baseline for my set of systems
// @Description Create a baseline for my set of systems. System cannot be satellite managed.
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
// nolint: funlen
func CreateBaselineHandler(c *gin.Context) {
	accountID := c.GetInt(utils.KeyAccount)
	creator := c.GetString(utils.KeyUser)
	groups := c.GetStringMapString(utils.KeyInventoryGroups)

	var request CreateBaselineRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		LogAndRespBadRequest(c, err, fmt.Sprintf("Invalid request body: %s", err.Error()))
		return
	}
	if len(creator) > 0 {
		request.Creator = &creator
	}

	if !utils.IsParamValid(&request.Name, false, false) {
		LogAndRespBadRequest(c, errors.New(BaselineMissingNameErr), BaselineMissingNameErr)
		return
	}
	if !utils.IsParamValid(request.Description, true, true) {
		LogAndRespBadRequest(c, errors.New(InvalidDescription), InvalidDescription)
		return
	}
	request.Description = utils.EmptyToNil(request.Description)

	db := middlewares.DBFromContext(c)
	err := checkInventoryIDs(db, accountID, request.InventoryIDs, groups)
	if err != nil {
		switch {
		case errors.Is(err, base.ErrBadRequest):
			LogAndRespBadRequest(c, err, err.Error())
			return
		case errors.Is(err, base.ErrNotFound):
			LogAndRespNotFound(c, err, err.Error())
			return
		default:
			LogAndRespError(c, err, "Database error")
			return
		}
	}

	baselineID, err := buildCreateBaselineQuery(db, request, accountID)
	if err != nil {
		if database.IsPgErrorCode(db, err, gorm.ErrDuplicatedKey) {
			LogAndRespBadRequest(c, err, DuplicateBaselineNameErr)
			return
		}
		LogAndRespError(c, err, "Database error")
		return
	}

	configUpdated := request.Config != nil
	inventoryIDs := kafka.GetInventoryIDsToEvaluate(db, &baselineID, accountID, configUpdated, nil)
	kafka.EvaluateBaselineSystems(inventoryIDs)

	resp := CreateBaselineResponse{BaselineID: baselineID}
	c.JSON(http.StatusOK, &resp)
}

func buildCreateBaselineQuery(db *gorm.DB, request CreateBaselineRequest, accountID int) (int64, error) {
	tx := db.Begin()
	defer tx.Rollback()

	now := time.Now()
	baseline := models.Baseline{
		RhAccountID: accountID,
		Name:        request.Name,
		Description: request.Description,
		Creator:     request.Creator,
		Published:   &now,
		LastEdited:  &now,
	}

	if request.Config != nil {
		config, err := sonic.Marshal(request.Config)
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

func checkInventoryIDs(db *gorm.DB, accountID int, inventoryIDs []string, groups map[string]string) (err error) {
	var containingSystems []SystemBaselineDBLookup
	var missingIDs []string
	var satelliteIDs []string
	var bootcIDs []string
	err = database.Systems(db, accountID, groups).
		Where("inventory_id IN (?::uuid)", inventoryIDs).
		Scan(&containingSystems).Error
	if err != nil {
		return errors.Join(base.ErrDatabase, err)
	}

	containingIDsMap := make(map[string]bool, len(containingSystems))
	for _, containingSystem := range containingSystems {
		containingIDsMap[containingSystem.InventoryID] = true

		if containingSystem.SatelliteManaged {
			satelliteIDs = append(satelliteIDs, containingSystem.InventoryID)
		}
		if containingSystem.Bootc {
			bootcIDs = append(bootcIDs, containingSystem.InventoryID)
		}
	}

	for _, inventoryID := range inventoryIDs {
		if _, ok := containingIDsMap[inventoryID]; !ok {
			missingIDs = append(missingIDs, inventoryID)
		}
	}

	sort.Strings(missingIDs)
	sort.Strings(satelliteIDs)
	sort.Strings(bootcIDs)

	switch {
	case config.EnableSatelliteFunctionality && len(satelliteIDs) > 0:
		errIDs := fmt.Errorf("template can not contain satellite managed systems: %v", satelliteIDs)
		err = errors.Join(err, base.ErrBadRequest, errIDs)
	case len(bootcIDs) > 0:
		errIDs := fmt.Errorf("template can not contain bootc systems: %v", bootcIDs)
		err = errors.Join(err, base.ErrBadRequest, errIDs)
	case len(missingIDs) > 0:
		errIDs := fmt.Errorf("unknown inventory_ids: %v", missingIDs)
		err = errors.Join(err, base.ErrNotFound, errIDs)
	}

	return err
}
