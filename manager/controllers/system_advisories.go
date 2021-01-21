package controllers

import (
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"app/manager/middlewares"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"net/http"
	"time"
)

var SystemAdvisoriesFields = database.MustGetQueryAttrs(&SystemAdvisoriesDBLookup{})
var SystemAdvisoriesSelect = database.MustGetSelect(&SystemAdvisoriesDBLookup{})
var SystemAdvisoriesOpts = ListOpts{
	Fields:         SystemAdvisoriesFields,
	DefaultFilters: nil,
	DefaultSort:    "-public_date",
	SearchFields:   []string{"am.name", "am.synopsis", "am.description"},
}

type SystemAdvisoriesDBLookup struct {
	ID string `query:"am.name"`
	SystemAdvisoryItemAttributes
}

// nolint: lll
type SystemAdvisoryItemAttributes struct {
	Description  string    `json:"description" csv:"description" query:"am.description" gorm:"column:description"`
	PublicDate   time.Time `json:"public_date" csv:"public_date" query:"am.public_date" gorm:"column:public_date"`
	Synopsis     string    `json:"synopsis" csv:"synopsis" query:"am.synopsis" gorm:"column:synopsis"`
	AdvisoryType int       `json:"advisory_type" csv:"advisory_type" query:"am.advisory_type_id" gorm:"column:advisory_type"`
	Severity     *int      `json:"severity,omitempty" csv:"severity" query:"am.severity_id" gorm:"column:severity"`
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
// @Param    limit          query   int     false   "Limit for paging, set -1 to return all"
// @Param    offset         query   int     false   "Offset for paging"
// @Param    sort           query   string  false   "Sort field"    Enums(id,name,type,synopsis,public_date)
// @Param    search         query   string  false   "Find matching text"
// @Param    filter[id]              query   string  false "Filter"
// @Param    filter[description]     query   string  false "Filter"
// @Param    filter[public_date]     query   string  false "Filter"
// @Param    filter[synopsis]        query   string  false "Filter"
// @Param    filter[advisory_type]   query   string  false "Filter"
// @Param    filter[severity]        query   string  false "Filter"
// @Success 200 {object} SystemAdvisoriesResponse
// @Router /api/patch/v1/systems/{inventory_id}/advisories [get]
func SystemAdvisoriesHandler(c *gin.Context) {
	account := c.GetInt(middlewares.KeyAccount)

	inventoryID := c.Param("inventory_id")
	if inventoryID == "" {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{Error: "inventory_id param not found"})
		return
	}

	var exists int64
	err := database.Db.Model(&models.SystemPlatform{}).Where("inventory_id = ?::uuid ", inventoryID).
		Count(&exists).Error

	if err != nil {
		LogAndRespError(c, err, "database error")
	}
	if exists == 0 {
		LogAndRespNotFound(c, errors.New("System not found"), "Systems not found")
		return
	}

	query := database.SystemAdvisoriesQueryName(database.Db, inventoryID).
		Select(SystemAdvisoriesSelect).
		Where("sp.rh_account_id = ?", account).
		Where("when_patched IS NULL")

	if applyInventoryHosts {
		query = query.Joins("JOIN inventory.hosts ih ON ih.id = sp.inventory_id")
	}

	path := fmt.Sprintf("/api/patch/v1/systems/%v/advisories", inventoryID)
	query, meta, links, err := ListCommon(query, c, path, SystemAdvisoriesOpts)
	if err != nil {
		// Error handling and setting of result code & content is done in ListCommon
		return
	}

	var dbItems []SystemAdvisoriesDBLookup

	if err = query.Find(&dbItems).Error; err != nil {
		LogAndRespError(c, err, "database error")
		return
	}

	data := buildSystemAdvisoriesData(dbItems)
	var resp = SystemAdvisoriesResponse{
		Data:  data,
		Links: *links,
		Meta:  *meta,
	}
	c.JSON(http.StatusOK, &resp)
}

func buildSystemAdvisoriesData(models []SystemAdvisoriesDBLookup) []SystemAdvisoryItem {
	data := make([]SystemAdvisoryItem, len(models))
	for i, advisory := range models {
		item := SystemAdvisoryItem{
			ID:         advisory.ID,
			Type:       "advisory",
			Attributes: advisory.SystemAdvisoryItemAttributes,
		}
		data[i] = item
	}
	return data
}
