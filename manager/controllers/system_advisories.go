package controllers

import (
	"app/base/database"
	"app/base/utils"
	"app/manager/middlewares"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"net/http"
	"time"
)

var SystemAdvisoriesFields = database.MustGetQueryAttrs(&SystemAdvisoriesDBLookup{})
var SystemAdvisoriesSelect = database.MustGetSelect(&SystemAdvisoriesDBLookup{})

type SystemAdvisoriesDBLookup struct {
	ID string `query:"am.name"`
	SystemAdvisoryItemAttributes
}

type SystemAdvisoryItemAttributes struct {
	Description  string    `json:"description" query:"am.description"`
	PublicDate   time.Time `json:"public_date" query:"am.public_date"`
	Synopsis     string    `json:"synopsis" query:"am.synopsis"`
	AdvisoryType int       `json:"advisory_type" query:"am.advisory_type_id"`
	Severity     *int      `json:"severity,omitempty" query:"am.severity_id"`
}

type SystemAdvisoryItem struct {
	Attributes SystemAdvisoryItemAttributes `json:"attributes"`
	ID         string                       `json:"id"`
	Type       string                       `json:"type"`
}

type SystemAdvisoriesResponse struct {
	Data  []SystemAdvisoryItem `json:"data"` // advisories items
	Links Links                `json:"links"`
	Meta  ListMeta             `json:"meta"`
}

// nolint:lll
// @Summary Show me advisories for a system by given inventory id
// @Description Show me advisories for a system by given inventory id
// @ID listSystemAdvisories
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    inventory_id   path    string  true    "Inventory ID"
// @Param    limit          query   int     false   "Limit for paging"
// @Param    offset         query   int     false   "Offset for paging"
// @Param    sort           query   string  false   "Sort field"    Enums(id,name,type,synopsis,public_date)
// @Success 200 {object} SystemAdvisoriesResponse
// @Router /api/patch/v1/systems/{inventory_id}/advisories [get]
func SystemAdvisoriesHandler(c *gin.Context) {
	account := c.GetString(middlewares.KeyAccount)

	inventoryID := c.Param("inventory_id")
	if inventoryID == "" {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{Error: "inventory_id param not found"})
		return
	}

	query := database.SystemAdvisoriesQueryName(database.Db, inventoryID).
		Select(SystemAdvisoriesSelect).
		Joins("INNER JOIN rh_account ra on sp.rh_account_id = ra.id").
		Where("ra.name = ?", account)

	path := fmt.Sprintf("/api/patch/v1/systems/%v/advisories", inventoryID)
	query, meta, links, err := ListCommon(query, c, path, SystemAdvisoriesFields, nil)
	if err != nil {
		// Error handling and setting of result code & content is done in ListCommon
		return
	}

	var dbItems []SystemAdvisoriesDBLookup
	err = query.Find(&dbItems).Error
	if gorm.IsRecordNotFoundError(err) {
		LogAndRespNotFound(c, err, "no systems found")
		return
	}

	if err != nil {
		LogAndRespError(c, err, "database error")
		return
	}

	data := buildSystemAdvisoriesData(dbItems)
	var resp = SystemAdvisoriesResponse{
		Data:  *data,
		Links: *links,
		Meta:  *meta,
	}
	c.JSON(http.StatusOK, &resp)
}

func buildSystemAdvisoriesData(models []SystemAdvisoriesDBLookup) *[]SystemAdvisoryItem {
	data := make([]SystemAdvisoryItem, len(models))
	for i, advisory := range models {
		item := SystemAdvisoryItem{
			ID:         advisory.ID,
			Type:       "advisory",
			Attributes: advisory.SystemAdvisoryItemAttributes,
		}
		data[i] = item
	}
	return &data
}
