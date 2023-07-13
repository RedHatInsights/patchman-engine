package controllers

import (
	"app/base/database"
	"app/base/utils"
	"app/manager/middlewares"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var BaselineFields = database.MustGetQueryAttrs(&BaselinesDBLookup{})
var BaselineSelect = database.MustGetSelect(&BaselinesDBLookup{})
var BaselineOpts = ListOpts{
	Fields:         BaselineFields,
	DefaultFilters: nil,
	DefaultSort:    "name",
	StableSort:     "id",
	SearchFields:   []string{"bl.name"},
}

type BaselinesDBLookup struct {
	ID int `query:"bl.id" gorm:"column:id"`
	// a helper to get total number of systems
	MetaTotalHelper

	BaselineItemAttributes
}

type BaselineItemAttributes struct {
	// Baseline name
	Name string `json:"name" csv:"name" query:"bl.name" gorm:"column:name" example:"my-baseline"`
	// Count of the systems associated with the baseline
	Systems int `json:"systems" csv:"systems" query:"systems" gorm:"column:systems" example:"22"`
	// Created and updated dates
	Published  *time.Time `json:"published,omitempty" csv:"published" query:"published" gorm:"column:published"`
	LastEdited *time.Time `json:"last_edited,omitempty" csv:"last_edited" query:"last_edited" gorm:"column:last_edited"`
	Creator    *string    `json:"creator,omitempty" csv:"creator" query:"creator" gorm:"column:creator"`
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

type BaselinesMeta struct {
	ListMeta
	Creators []*string `json:"creators,omitempty"`
}

type BaselinesResponse struct {
	Data  []BaselineItem `json:"data"`  // Baseline items
	Links Links          `json:"links"` // Pagination links
	Meta  BaselinesMeta  `json:"meta"`  // Generic response fields (pagination params, filters etc.)
}

// @Summary Show me all baselines for all my systems
// @Description  Show me all baselines for all my systems
// @ID listBaseline
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    limit          query   int     false   "Limit for paging, set -1 to return all"
// @Param    offset         query   int     false   "Offset for paging"
// @Param    sort           query   string  false   "Sort field"    Enums(id,name,systems,published,last_edited,creator)
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
	apiver := c.GetInt(middlewares.KeyApiver)
	groups := c.GetStringMapString(middlewares.KeyInventoryGroups)
	filters, err := ParseTagsFilters(c)
	if err != nil {
		return
	}

	db := middlewares.DBFromContext(c)
	query := buildQueryBaselines(db, filters, account, groups)
	if err != nil {
		return
	} // Error handled in method itself

	query, meta, params, err := ListCommon(query, c, filters, BaselineOpts)
	if err != nil {
		// Error handling and setting of result code & content is done in ListCommon
		return
	}

	var baselines []BaselinesDBLookup
	err = query.Find(&baselines).Error
	if err != nil {
		LogAndRespError(c, err, err.Error())
		return
	}

	baselinesMeta, err := creatorsMeta(c, db, account)
	if err != nil {
		LogAndRespError(c, err, err.Error())
		return
	}

	data, total := buildBaselinesData(baselines, apiver)
	meta, links, err := UpdateMetaLinks(c, meta, total, nil, params...)
	if err != nil {
		return // Error handled in method itself
	}

	baselinesMeta.ListMeta = *meta
	var resp = BaselinesResponse{
		Data:  data,
		Links: *links,
		Meta:  baselinesMeta,
	}
	c.JSON(http.StatusOK, &resp)
}

func buildQueryBaselines(db *gorm.DB, filters map[string]FilterData, account int, groups map[string]string) *gorm.DB {
	subq := database.Systems(db, account, groups).
		Select("sp.baseline_id, count(sp.inventory_id) as systems").
		Group("sp.baseline_id")

	subq, _ = ApplyTagsFilter(filters, subq, "sp.inventory_id")

	query := db.Table("baseline as bl").
		Select(BaselineSelect).
		Joins("LEFT JOIN (?) sp ON sp.baseline_id = bl.id", subq).
		Where("bl.rh_account_id = ?", account)

	return query
}

func buildBaselinesData(baselines []BaselinesDBLookup, apiver int) ([]BaselineItem, int) {
	var total int
	if len(baselines) > 0 {
		total = baselines[0].Total
	}
	data := make([]BaselineItem, len(baselines))
	for i := 0; i < len(baselines); i++ {
		baseline := baselines[i]
		data[i] = BaselineItem{
			Attributes: BaselineItemAttributes{
				Name:       baseline.Name,
				Systems:    baseline.Systems,
				Published:  APIV3Compat(baseline.Published, apiver),
				LastEdited: APIV3Compat(baseline.LastEdited, apiver),
				Creator:    APIV3Compat(baseline.Creator, apiver),
			},
			ID:   baseline.ID,
			Type: "baseline",
		}
	}
	return data, total
}

func creatorsMeta(c *gin.Context, db *gorm.DB, account int) (BaselinesMeta, error) {
	apiver := c.GetInt(middlewares.KeyApiver)
	// list of creators for account
	baselinesMeta := BaselinesMeta{}
	err := db.Table("baseline bl").
		Distinct("COALESCE(creator, '')").
		Joins("JOIN rh_account acc ON bl.rh_account_id = acc.id").
		Where("bl.rh_account_id = ?", account).
		Scan(&baselinesMeta.Creators).Error
	if err != nil {
		return baselinesMeta, err
	}

	// remove "" from baselinesMeta.Creators
	creators := make([]*string, 0, len(baselinesMeta.Creators))
	for _, c := range baselinesMeta.Creators {
		creators = append(creators, utils.EmptyToNil(c))
	}
	if len(creators) == 1 && creators[0] == nil {
		creators = []*string{}
	}
	baselinesMeta.Creators = creators
	if apiver < 3 {
		baselinesMeta.Creators = nil
	}
	return baselinesMeta, nil
}
