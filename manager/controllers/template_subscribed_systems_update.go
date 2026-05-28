package controllers

import (
	"app/base/utils"
	"app/manager/config"
	"app/manager/kafka"
	"app/manager/middlewares"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

// @Summary Add a system to a template
// @Description Add a system authenticated by  its client identity certificate to a template
// @ID addTemplateSubscribedSystem
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    template_id    path  string   true  "Template ID"
// @Success 200
// @Failure 400 {object} 	utils.ErrorResponse
// @Failure 404 {object} 	utils.ErrorResponse
// @Failure 500 {object} 	utils.ErrorResponse
// @Router /templates/{template_id}/subscribed-systems [PATCH]
func TemplateSubscribedSystemsUpdateHandler(c *gin.Context) {
	templateUUID := c.Param("template_id")

	db := middlewares.DBFromContext(c)

	account, orgID, systemUUID, err := getSubscribedSystem(c, db)
	if err != nil {
		// respose set in getTemplateID()
		return
	}
	template, err := getTemplate(c, db, account, templateUUID)
	if err != nil {
		// respose set in getTemplateID()
		return
	}

	systemList := []uuid.UUID{systemUUID}
	err = checkTemplateSystems(c, db, account, template, systemList, nil)
	if err != nil {
		return
	}

	err = assignCandlepinEnvironment(c, db, account, &template.EnvironmentID, systemList, nil)
	if err != nil {
		return
	}

	err = assignTemplateSystems(c, db, account, template, systemList)
	if err != nil {
		return
	}

	// re-evaluate systems added/removed from templates
	if config.EnableTemplateChangeEval {
		evalDataList := kafka.InventoryIDs2EvalData(account, orgID, systemList)
		kafka.RecalcSystems(evalDataList)
	}
	c.Status(http.StatusOK)
}

func getSubscribedSystem(c *gin.Context, tx *gorm.DB) (int, string, uuid.UUID, error) {
	account := c.GetInt(utils.KeyAccount)
	orgID := c.GetString(utils.KeyOrgID)
	systemCn := c.GetString(utils.KeySystem)

	var inventoryIDStr string
	err := tx.Select("inventory_id").
		Table("system_inventory").
		Where("subscription_manager_id = ? AND rh_account_id = ?", systemCn, account).
		Limit(1).Find(&inventoryIDStr).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		utils.LogAndRespError(c, err, "database error")
		return 0, "", uuid.Nil, err
	}
	inventoryID, _ := uuid.Parse(inventoryIDStr)
	if inventoryID == uuid.Nil {
		err := errors.Errorf("System %s not found", systemCn)
		utils.LogAndRespNotFound(c, err, err.Error())
		return 0, "", uuid.Nil, err
	}
	return account, orgID, inventoryID, err
}
