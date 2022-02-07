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
	SearchFields:   []string{"bl.name"},
	TotalFunc:      CountRows,
}

type BaselinesDBLookup struct {
	ID int `query:"bl.id" gorm:"column:id"`
	BaselineItemAttributes
}

type BaselineItemAttributes struct {
	Name    string `json:"name" csv:"name" query:"bl.name" gorm:"column:name"`
	Systems int    `json:"systems" csv:"systems" query:"systems" gorm:"column:systems"`
}

type BaselineItem struct {
	Attributes BaselineItemAttributes `json:"attributes"`
	ID         int                    `json:"id"`
	Type       string                 `json:"type"`
}

type BaselineInlineItem struct {
	ID string `json:"id" csv:"id"`
	BaselineItemAttributes
}

type BaselinesResponse struct {
	Data  []BaselineItem `json:"data"`
	Links Links          `json:"links"`
	Meta  ListMeta       `json:"meta"`
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
// @Router /api/patch/v1/baselines [get]
func BaselinesListHandler(c *gin.Context) {
	account := c.GetInt(middlewares.KeyAccount)
	var query *gorm.DB

	query, err := buildQueryBaselines(c, account)
	if err != nil {
		return
	} // Error handled in method itself

	query, meta, links, err := ListCommon(query, c, "/api/patch/v1/baselines", BaselineOpts)
	if err != nil {
		// Error handling and setting of result code & content is done in ListCommon
		return
	}

	var baselines []BaselinesDBLookup
	err = query.Find(&baselines).Error
	if err != nil {
		LogAndRespError(c, err, err.Error())
	}

	data := buildBaselinesData(baselines)
	var resp = BaselinesResponse{
		Data:  data,
		Links: *links,
		Meta:  *meta,
	}
	c.JSON(http.StatusOK, &resp)
}

func buildQueryBaselines(c *gin.Context, account int) (*gorm.DB, error) {
	subq := database.Db.Table("system_platform sp").
		Select("sp.baseline_id, count(sp.inventory_id) as systems").
		Joins("JOIN inventory.hosts ih ON ih.id = sp.inventory_id").
		Where("sp.rh_account_id = ?", account).
		Where("sp.stale = false").
		Group("sp.baseline_id")

	subq, _, err := ApplyTagsFilter(c, subq, "sp.inventory_id")
	if err != nil {
		return nil, err
	}

	query := database.Db.Table("baseline as bl").
		Select(BaselineSelect).
		Joins("LEFT JOIN (?) sp ON sp.baseline_id = bl.id", subq).
		Where("bl.rh_account_id = ?", account)

	return query, nil
}

func buildBaselinesData(baselines []BaselinesDBLookup) []BaselineItem {
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
	return data
}
