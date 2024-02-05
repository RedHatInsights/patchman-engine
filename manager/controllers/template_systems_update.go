package controllers

import (
	"app/base/models"
	"app/base/utils"
	"app/manager/middlewares"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
)

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

	if !utils.IsValidUUID(templateUUID) {
		errmsg := "Invalid template uuid: " + templateUUID
		LogAndRespNotFound(c, errors.New(errmsg), errmsg)
		return
	}
	var req TemplateSystemsUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		LogAndRespBadRequest(c, err, "Invalid template update request "+err.Error())
		return
	}

	db := middlewares.DBFromContext(c)
	templateID, err := getTemplateID(db, account, templateUUID)
	if err != nil {
		LogAndRespError(c, err, "database error")
		return
	}
	if templateID == 0 {
		LogAndRespNotFound(c, errors.New("Template not found"), "Template not found")
		return
	}

	missingIDs, satelliteManagedIDs, err := checkInventoryIDs(db, account, req.Systems, groups)
	if err != nil {
		LogAndRespError(c, err, "Database error")
		return
	}

	if enableSatelliteFunctionality && len(satelliteManagedIDs) > 0 {
		msg := fmt.Sprintf("Attempting to add satellite managed systems to baseline: %v", satelliteManagedIDs)
		LogAndRespBadRequest(c, errors.New(msg), msg)
		return
	} else if len(missingIDs) > 0 {
		msg := fmt.Sprintf("Missing inventory_ids: %v", missingIDs)
		LogAndRespNotFound(c, errors.New(msg), msg)
		return
	}
	err = assignTemplateSystems(db, account, templateID, req.Systems)
	if err != nil {
		switch e := err.Error(); e {
		case InvalidInventoryIDsErr:
			LogAndRespBadRequest(c, err, e)
		default:
			LogAndRespError(c, err, "database error")
		}
		return
	}

	// TODO: re-evaluate systems added/removed from templates
	// inventoryAIDs := kafka.InventoryIDs2InventoryAIDs(account, req.InventoryIDs)
	// kafka.EvaluateBaselineSystems(inventoryAIDs)

	c.Status(http.StatusOK)
}

func assignTemplateSystems(db *gorm.DB, accountID int, templateID int64, inventoryIDs []string) error {
	if len(inventoryIDs) == 0 {
		return errors.New(InvalidInventoryIDsErr)
	}
	tx := db.Begin()
	defer tx.Rollback()

	tx = tx.Model(models.SystemPlatform{}).
		Where("rh_account_id = ? AND inventory_id IN (?::uuid)",
			accountID, inventoryIDs).
		Update("template_id", templateID)
	if e := tx.Error; e != nil {
		return e
	}
	if int(tx.RowsAffected) != len(inventoryIDs) {
		return errors.New(InvalidInventoryIDsErr)
	}

	err := tx.Commit().Error
	return err
}
