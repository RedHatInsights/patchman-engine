package controllers

import (
	"app/base/database"
	"app/manager/middlewares"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var BaselineFields = database.MustGetQueryAttrs(&BaselinesDBLookup{})
var BaselineSelect = database.MustGetSelect(&BaselinesDBLookup{})
var BaselineOpts = ListOpts{
	Fields:         BaselineFields,
	DefaultFilters: nil,
	DefaultSort:    "-name",
	StableSort:     "id",
	SearchFields:   []string{"bl.name"},
	TotalFunc:      CountRows,
}

type BaselinesDBLookup struct {
	ID int `query:"bl.id" gorm:"column:id"`
	// a helper to get total number of systems
	Total int `json:"-" csv:"-" query:"count(bl.id) over ()" gorm:"column:total"`

	BaselineItemAttributes
}

type BaselineItemAttributes struct {
	// Baseline name
	Name string `json:"name" csv:"name" query:"bl.name" gorm:"column:name" example:"my-baseline"`
	// Count of the systems associated with the baseline
	Systems int `json:"systems" csv:"systems" query:"systems" gorm:"column:systems" example:"22"`
}

type BaselineItem struct {
	Attributes BaselineItemAttributes `json:"attributes"`              // Additional baseline attributes
	ID         int                    `json:"id" example:"10"`         // Unique baseline id
	Type       string                 `json:"type" example:"baseline"` // Document type name
}

type BaselineInlineItem struct {
	ID string `json:"id" csv:"id"`
	BaselineItemAttributes
}

type BaselinesResponse struct {
	Data  []BaselineItem `json:"data"`  // Baseline items
	Links Links          `json:"links"` // Pagination links
	Meta  ListMeta       `json:"meta"`  // Generic response fields (pagination params, filters etc.)
}

// @Summary Show me all baselines for all my systems
// @Description  Show me all baselines for all my systems
// @ID listBaseline
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    limit          query   int     false   "Limit for paging, set -1 to return all"
// @Param    offset         query   int     false   "Offset for paging"
// @Param    sort           query   string  false   "Sort field"    Enums(id,name,config)
// @Param    search         query   string  false   "Find matching text"
// @Param    filter[id]           query   string  false "Filter "
// @Param    filter[name]         query   string  false "Filter"
// @Param    filter[systems]      query   string  false "Filter"
// @Param    tags           query   []string  false "Tag filter"
// @Success 200 {object} BaselinesResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /baselines [get]
func BaselinesListHandler(c *gin.Context) {
	account := c.GetInt(middlewares.KeyAccount)
	filters, err := ParseTagsFilters(c)
	if err != nil {
		return
	}

	db := middlewares.DBFromContext(c)
	query := buildQueryBaselines(db, filters, account)
	if err != nil {
		return
	} // Error handled in method itself

	query, meta, params, err := ListCommonWithoutCount(query, c, filters, BaselineOpts)
	if err != nil {
		// Error handling and setting of result code & content is done in ListCommon
		return
	}

	var baselines []BaselinesDBLookup
	err = query.Find(&baselines).Error
	if err != nil {
		LogAndRespError(c, err, err.Error())
	}

	data, total := buildBaselinesData(baselines)
	meta, links, err := UpdateMetaLinks(c, meta, total, nil, params...)
	if err != nil {
		return // Error handled in method itself
	}
	var resp = BaselinesResponse{
		Data:  data,
		Links: *links,
		Meta:  *meta,
	}
	c.JSON(http.StatusOK, &resp)
}

func buildQueryBaselines(db *gorm.DB, filters map[string]FilterData, account int) *gorm.DB {
	subq := db.Table("system_platform sp").
		Select("sp.baseline_id, count(sp.inventory_id) as systems").
		Joins("JOIN inventory.hosts ih ON ih.id = sp.inventory_id").
		Where("sp.rh_account_id = ?", account).
		Where("sp.stale = false").
		Group("sp.baseline_id")

	subq, _ = ApplyTagsFilter(filters, subq, "sp.inventory_id")

	query := db.Table("baseline as bl").
		Select(BaselineSelect).
		Joins("LEFT JOIN (?) sp ON sp.baseline_id = bl.id", subq).
		Where("bl.rh_account_id = ?", account).Order("bl.name asc")

	return query
}

func buildBaselinesData(baselines []BaselinesDBLookup) ([]BaselineItem, int) {
	var total int
	if len(baselines) > 0 {
		total = baselines[0].Total
	}
	data := make([]BaselineItem, len(baselines))
	for i := 0; i < len(baselines); i++ {
		baseline := baselines[i]
		data[i] = BaselineItem{
			Attributes: BaselineItemAttributes{
				Name:    baseline.Name,
				Systems: baseline.Systems,
			},
			ID:   baseline.ID,
			Type: "baseline",
		}
	}
	return data, total
}
