package controllers

import (
	"app/base/utils"
	"app/manager/config"
	"app/manager/kafka"
	"app/manager/middlewares"
	"net/http"

	"github.com/gin-gonic/gin"
)

// @Summary Remove systems from template
// @Description Remove systems from template
// @ID removeTemplateSystems
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    body    body   TemplateSystemsUpdateRequest true "Request body"
// @Success 200
// @Failure 400 {object} 	utils.ErrorResponse
// @Failure 404 {object} 	utils.ErrorResponse
// @Failure 500 {object} 	utils.ErrorResponse
// @Router /templates/systems [DELETE]
func TemplateSystemsDeleteHandler(c *gin.Context) {
	account := c.GetInt(utils.KeyAccount)
	groups := c.GetStringMapString(utils.KeyInventoryGroups)

	var req TemplateSystemsUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		LogAndRespBadRequest(c, err, "Invalid template delete request "+err.Error())
		return
	}

	db := middlewares.DBFromContext(c)

	err := checkTemplateSystems(c, db, account, nil, req.Systems, groups)
	if err != nil {
		return
	}

	modified, _ := assignCandlepinEnvironment(db, account, nil, req.Systems, groups)

	// unassign system from template => assign NULL as template_id
	err = assignTemplateSystems(c, db, account, nil, modified)
	if err != nil {
		return
	}

	// re-evaluate systems removed from templates
	if config.EnableTemplateChangeEval {
		inventoryAIDs := kafka.InventoryIDs2InventoryAIDs(account, req.Systems)
		kafka.EvaluateBaselineSystems(inventoryAIDs)
	}
	c.Status(http.StatusOK)
}
