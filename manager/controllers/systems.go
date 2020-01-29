package controllers

import (
	"app/base/database"
	"app/base/models"
	"app/manager/middlewares"
	"github.com/gin-gonic/gin"
	"net/http"
)

var SystemsSortFields = []string{"last_updated", "last_evaluation"}

type SystemsResponse struct {
	Data  []SystemItem `json:"data"`
	Links Links        `json:"links"`
	Meta  ListMeta     `json:"meta"`
}

// @Summary Show me all my systems
// @Description Show me all my systems
// @ID listSystems
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    limit          query   int     false   "Limit for paging"
// @Param    offset         query   int     false   "Offset for paging"
// @Param    sort           query   string  false   "Sort field"    Enums(id,last_updated,last_evaluation)
// @Success 200 {object} SystemsResponse
// @Router /api/patch/v1/systems [get]
func SystemsListHandler(c *gin.Context) {
	account := c.GetString(middlewares.KeyAccount)

	query := database.Db.Model(models.SystemPlatform{}).
		Joins("inner join rh_account ra on system_platform.rh_account_id = ra.id").
		Where("ra.name = ?", account)

	query, meta, links, err := ListCommon(query, c, SystemsSortFields, "/api/patch/v1/systems")
	if err != nil {
		LogAndRespError(c, err, err.Error())
		return
	}

	var systems []models.SystemPlatform
	err = query.Find(&systems).Error
	if err != nil {
		LogAndRespError(c, err, "db error")
		return
	}

	data := buildData(&systems)
	var resp = SystemsResponse{
		Data:  *data,
		Links: *links,
		Meta:  *meta,
	}
	c.JSON(http.StatusOK, &resp)
}

func buildData(systems *[]models.SystemPlatform) *[]SystemItem {
	data := make([]SystemItem, len(*systems))
	for i := 0; i < len(*systems); i++ {
		system := (*systems)[i]
		data[i] = SystemItem{
			Attributes: SystemItemAttributes{
				LastEvaluation: system.LastEvaluation,
				LastUpload:     system.LastUpload,
				RhsaCount:      system.AdvisorySecCountCache,
				RheaCount:      system.AdvisoryEnhCountCache,
				RhbaCount:      system.AdvisoryBugCountCache,
				Enabled:        !system.OptOut,
			},
			ID:   system.InventoryID,
			Type: "system",
		}
	}
	return &data
}
