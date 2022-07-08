package controllers

import (
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"app/manager/middlewares"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

var SystemAdvisoriesFields = database.MustGetQueryAttrs(&SystemAdvisoriesDBLookup{})
var SystemAdvisoriesSelect = database.MustGetSelect(&SystemAdvisoriesDBLookup{})
var SystemAdvisoriesOpts = ListOpts{
	Fields:         SystemAdvisoriesFields,
	DefaultFilters: nil,
	DefaultSort:    "-public_date",
	SearchFields:   []string{"am.name", "am.synopsis"},
	TotalFunc:      CountRows,
}

type RelList []string

type SystemAdvisoriesDBLookup struct {
	ID string `json:"id" csv:"id" query:"am.name" gorm:"column:id"`
	SystemAdvisoryItemAttributes
}

// nolint:lll
type SystemAdvisoryItemAttributes struct {
	Description      string    `json:"description" csv:"description" query:"am.description" gorm:"column:description"`
	PublicDate       time.Time `json:"public_date" csv:"public_date" query:"am.public_date" gorm:"column:public_date"`
	Synopsis         string    `json:"synopsis" csv:"synopsis" query:"am.synopsis" gorm:"column:synopsis"`
	AdvisoryType     int       `json:"advisory_type" csv:"advisory_type" query:"am.advisory_type_id" gorm:"column:advisory_type"`                                // Deprecated, not useful database ID (0 - unknown, 1 -, enhancement, 2 - bugfix, 3 - security, 4 - unspecified)
	AdvisoryTypeName string    `json:"advisory_type_name" csv:"advisory_type_name" query:"at.name" order_query:"at.preference" gorm:"column:advisory_type_name"` // Advisory type name, proper ordering ensured (unknown, unspecified, other, enhancement, bugfix, security)
	Severity         *int      `json:"severity,omitempty" csv:"severity" query:"am.severity_id" gorm:"column:severity"`
	CveCount         int       `json:"cve_count" csv:"cve_count" query:"CASE WHEN jsonb_typeof(am.cve_list) = 'array' THEN jsonb_array_length(am.cve_list) ELSE 0 END" gorm:"column:cve_count"`
	RebootRequired   bool      `json:"reboot_required" csv:"reboot_required" query:"am.reboot_required" gorm:"column:reboot_required"`
	ReleaseVersions  RelList   `json:"release_versions" csv:"release_versions" query:"null" gorm:"-"`

	// helper field to get release_version json from db and parse it to ReleaseVersions field
	ReleaseVersionsJSONB []byte `json:"-" csv:"-" query:"am.release_versions" gorm:"column:release_versions_json"`
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

func (v RelList) String() string {
	return strings.Join(v, ",")
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
// @Param    filter[id]                  query   string  false "Filter"
// @Param    filter[description]         query   string  false "Filter"
// @Param    filter[public_date]         query   string  false "Filter"
// @Param    filter[synopsis]            query   string  false "Filter"
// @Param    filter[advisory_type]       query   string  false "Filter"
// @Param    filter[advisory_type_name]  query   string  false "Filter"
// @Param    filter[severity]            query   string  false "Filter"
// @Success 200 {object} SystemAdvisoriesResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /systems/{inventory_id}/advisories [get]
func SystemAdvisoriesHandler(c *gin.Context) {
	account := c.GetInt(middlewares.KeyAccount)

	inventoryID := c.Param("inventory_id")
	if inventoryID == "" {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{Error: "inventory_id param not found"})
		return
	}

	if !utils.IsValidUUID(inventoryID) {
		LogAndRespBadRequest(c, errors.New("bad request"), "incorrect inventory_id format")
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

	query := buildSystemAdvisoriesQuery(account, inventoryID)
	query, meta, links, err := ListCommon(query, c, nil, SystemAdvisoriesOpts)
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

func buildSystemAdvisoriesQuery(account int, inventoryID string) *gorm.DB {
	query := database.SystemAdvisoriesByInventoryID(database.Db, account, inventoryID).
		Joins("JOIN advisory_metadata am on am.id = sa.advisory_id").
		Joins("JOIN advisory_type at ON am.advisory_type_id = at.id").
		Select(SystemAdvisoriesSelect)
	return query
}

func buildSystemAdvisoriesData(models []SystemAdvisoriesDBLookup) []SystemAdvisoryItem {
	data := make([]SystemAdvisoryItem, len(models))
	for i, advisory := range models {
		advisory.SystemAdvisoryItemAttributes = systemAdvisoryItemAttributeParse(advisory.SystemAdvisoryItemAttributes)
		item := SystemAdvisoryItem{
			ID:         advisory.ID,
			Type:       "advisory",
			Attributes: advisory.SystemAdvisoryItemAttributes,
		}
		data[i] = item
	}
	return data
}
