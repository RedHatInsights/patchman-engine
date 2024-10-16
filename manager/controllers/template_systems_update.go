package controllers

import (
	"app/base"
	"app/base/candlepin"
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"app/manager/config"
	"app/manager/kafka"
	"app/manager/middlewares"
	"context"
	"fmt"
	"net/http"

	errors2 "errors"

	"github.com/pkg/errors"
	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
)

var errCandlepin = errors.New("candlepin error")
var candlepinClient = config.CreateCandlepinClient()

type TemplateSystemsUpdateRequest struct {
	// List of inventory IDs to have templates removed
	Systems []string `json:"systems" example:"system1-uuid, system2-uuid, ..."`
}

// @Summary Add systems to a template
// @Description Add systems to a template
// @ID updateTemplateSystems
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    body    body   TemplateSystemsUpdateRequest true "Request body"
// @Param    template_id    path  string   true  "Template ID"
// @Success 200
// @Failure 400 {object} 	utils.ErrorResponse
// @Failure 404 {object} 	utils.ErrorResponse
// @Failure 500 {object} 	utils.ErrorResponse
// @Router /templates/{template_id}/systems [PUT]
func TemplateSystemsUpdateHandler(c *gin.Context) {
	account := c.GetInt(utils.KeyAccount)
	templateUUID := c.Param("template_id")
	groups := c.GetStringMapString(utils.KeyInventoryGroups)

	var req TemplateSystemsUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		LogAndRespBadRequest(c, err, "Invalid template update request "+err.Error())
		return
	}

	db := middlewares.DBFromContext(c)
	template, err := getTemplate(c, db, account, templateUUID)
	if err != nil {
		// respose set in getTemplateID()
		return
	}

	err = checkTemplateSystems(c, db, account, template, req.Systems, groups)
	if err != nil {
		return
	}

	modified, _ := assignCandlepinEnvironment(db, account, &template.EnvironmentID, req.Systems, groups)

	err = assignTemplateSystems(c, db, account, template, modified)
	if err != nil {
		return
	}

	// re-evaluate systems added/removed from templates
	if config.EnableTemplateChangeEval {
		inventoryAIDs := kafka.InventoryIDs2InventoryAIDs(account, req.Systems)
		kafka.EvaluateBaselineSystems(inventoryAIDs)
	}
	c.Status(http.StatusOK)
}

func checkTemplateSystems(c *gin.Context, db *gorm.DB, accountID int, template *models.Template,
	inventoryIDs []string, groups map[string]string) error {
	if len(inventoryIDs) == 0 {
		err := errors.New(InvalidInventoryIDsErr)
		LogAndRespBadRequest(c, err, InvalidInventoryIDsErr)
		return err
	}

	err := checkInventoryIDs(db, accountID, inventoryIDs, groups)
	if err != nil {
		switch {
		case errors.Is(err, base.ErrBadRequest):
			LogAndRespBadRequest(c, err, err.Error())
			return err
		case errors.Is(err, base.ErrNotFound):
			LogAndRespNotFound(c, err, err.Error())
			return err
		default:
			LogAndRespError(c, err, "Database error")
			return err
		}
	}

	if err := templateArchVersionMatch(db, inventoryIDs, template, accountID, groups); err != nil {
		msg := fmt.Sprintf("Incompatible template and system version or architecture: %s", err.Error())
		LogAndRespBadRequest(c, err, msg)
		return err
	}

	return nil
}

func assignTemplateSystems(c *gin.Context, db *gorm.DB, accountID int, template *models.Template,
	inventoryIDs []string) error {
	tx := db.Begin()
	defer tx.Rollback()

	// if we want to unassign system from template, we need to set template_id=null
	var templateID *int64
	if template != nil && template.ID != 0 {
		templateID = &template.ID
	}

	tx = tx.Model(models.SystemPlatform{}).
		Where("rh_account_id = ? AND inventory_id IN (?::uuid)",
			accountID, inventoryIDs).
		Update("template_id", templateID)
	if err := tx.Error; err != nil {
		LogAndRespError(c, err, "Database error")
		return err
	}
	if int(tx.RowsAffected) != len(inventoryIDs) {
		err := errors.New(InvalidInventoryIDsErr)
		LogAndRespBadRequest(c, err, InvalidInventoryIDsErr)
		return err
	}

	err := tx.Commit().Error
	if e := tx.Error; e != nil {
		LogAndRespError(c, err, "Database error")
		return err
	}
	return nil
}

func templateArchVersionMatch(
	db *gorm.DB, inventoryIDs []string, template *models.Template, acc int, groups map[string]string,
) error {
	if template == nil {
		return nil
	}
	var sysArchVersions = []struct {
		InventoryID string
		Arch        string
		Version     string
	}{}
	var err error
	err = database.Systems(db, acc, groups).
		Select("ih.id as inventory_id, ih.system_profile->'operating_system'->>'major' as version, sp.arch as arch").
		Where("ih.id in (?)", inventoryIDs).Find(&sysArchVersions).Error
	if err != nil {
		return err
	}

	for _, sys := range sysArchVersions {
		if sys.Version != template.Version || sys.Arch != template.Arch {
			systemErr := fmt.Errorf("system uuid: %s, arch: %s, version: %s", sys.InventoryID, sys.Arch, sys.Version)
			err = errors2.Join(err, systemErr)
		}
	}
	if err != nil {
		err = errors2.Join(fmt.Errorf("template arch: %s, version: %s", template.Arch, template.Version), err)
	}
	return err
}

func callCandlepin(ctx context.Context, consumer string, request *candlepin.ConsumersUpdateRequest) (
	*candlepin.ConsumersUpdateResponse, error) {
	candlepinEnvConsumersURL := utils.CoreCfg.CandlepinAddress + "/consumers/" + consumer
	candlepinFunc := func() (interface{}, *http.Response, error) {
		candlepinResp := candlepin.ConsumersUpdateResponse{}
		resp, err := candlepinClient.Request(&ctx, http.MethodPut, candlepinEnvConsumersURL, request, &candlepinResp)
		statusCode := utils.TryGetStatusCode(resp)
		utils.LogDebug("request", *request, "candlepin_url", candlepinEnvConsumersURL,
			"status_code", statusCode, "err", err)
		if err != nil || statusCode != http.StatusOK {
			err = errors.Wrap(errCandlepin, err.Error())
		}
		return &candlepinResp, resp, err
	}

	candlepinRespPtr, err := utils.HTTPCallRetry(base.Context, candlepinFunc, config.CandlepinExpRetries,
		config.CandlepinRetries, http.StatusServiceUnavailable)
	if err != nil {
		return nil, errors.Wrap(err, "candlepin /consumers call failed")
	}
	return candlepinRespPtr.(*candlepin.ConsumersUpdateResponse), nil
}

func assignCandlepinEnvironment(db *gorm.DB, accountID int, env *string, inventoryIDs []string,
	groups map[string]string) ([]string, error) {
	var assignedIDs []string
	var consumers = []struct {
		InventoryID string
		Consumer    *string
	}{}

	err := database.Systems(db, accountID, groups).
		Select("ih.id as inventory_id, ih.system_profile->>'owner_id' as consumer").
		Where("ih.id in (?)", inventoryIDs).Find(&consumers).Error
	if err != nil {
		return nil, err
	}

	environments := []candlepin.ConsumersUpdateEnvironment{}
	if env != nil {
		environments = []candlepin.ConsumersUpdateEnvironment{{ID: *env}}
	}
	updateReq := candlepin.ConsumersUpdateRequest{
		Environments: environments,
	}
	for _, consumer := range consumers {
		if consumer.Consumer == nil {
			err = errors2.Join(err, errors.Errorf("Missing owner_id for '%s'", consumer.InventoryID))
			continue
		}
		resp, apiErr := callCandlepin(base.Context, *consumer.Consumer, &updateReq)
		// check response
		if apiErr != nil {
			err = errors2.Join(err, apiErr, errors.New(resp.Message))
		} else {
			assignedIDs = append(assignedIDs, consumer.InventoryID)
		}
	}

	return assignedIDs, err
}
