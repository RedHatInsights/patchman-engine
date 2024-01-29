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

var TemplateFields = database.MustGetQueryAttrs(&TemplatesDBLookup{})
var TemplateSelect = database.MustGetSelect(&TemplatesDBLookup{})
var TemplateOpts = ListOpts{
	Fields:         TemplateFields,
	DefaultFilters: nil,
	DefaultSort:    "name",
	StableSort:     "id",
	SearchFields:   []string{"tp.name"},
}

type TemplatesDBLookup struct {
	ID string `json:"id" csv:"id" query:"tp.uuid"`
	// a helper to get total number of systems
	MetaTotalHelper

	TemplateItemAttributes
}

type TemplateItemAttributes struct {
	// Template name
	Name string `json:"name" csv:"name" query:"tp.name"`
	// Count of the systems associated with the template
	Systems int `json:"systems" csv:"systems" query:"systems"`
	// Created and updated dates
	Published  *time.Time `json:"published,omitempty" csv:"published" query:"published"`
	LastEdited *time.Time `json:"last_edited,omitempty" csv:"last_edited" query:"last_edited"`
	Creator    *string    `json:"creator,omitempty" csv:"creator" query:"creator"`
}

type TemplateItem struct {
	Attributes TemplateItemAttributes `json:"attributes"` // Additional template attributes
	ID         string                 `json:"id"`         // Unique template id
	Type       string                 `json:"type"`       // Document type name
}

type TemplatesMeta struct {
	ListMeta
	Creators []*string `json:"creators,omitempty"`
}

type TemplatesResponse struct {
	Data  []TemplateItem `json:"data"`  // Template items
	Links Links          `json:"links"` // Pagination links
	Meta  TemplatesMeta  `json:"meta"`  // Generic response fields (pagination params, filters etc.)
}

// @Summary Show all templates for an account
// @Description  Show all templates for an account
// @ID listTemplate
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    limit          query   int     false   "Limit for paging"
// @Param    offset         query   int     false   "Offset for paging"
// @Param    sort           query   string  false   "Sort field"    Enums(id,name,systems,published,last_edited)
// @Param    search         query   string  false   "Find matching text"
// @Param    filter[id]           query   string  false "Filter "
// @Param    filter[name]         query   string  false "Filter"
// @Param    filter[systems]      query   string  false "Filter"
// @Param    tags           query   []string  false "Tag filter"
// @Success 200 {object} TemplatesResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /templates [get]
func TemplatesListHandler(c *gin.Context) {
	account := c.GetInt(utils.KeyAccount)
	groups := c.GetStringMapString(utils.KeyInventoryGroups)
	filters, err := ParseAllFilters(c, TemplateOpts)
	if err != nil {
		return
	}

	db := middlewares.DBFromContext(c)
	query := templatesQuery(db, filters, account, groups)

	query, meta, params, err := ListCommon(query, c, filters, TemplateOpts)
	if err != nil {
		// Error handling and setting of result code & content is done in ListCommon
		return
	}

	var templates []TemplatesDBLookup
	err = query.Find(&templates).Error
	if err != nil {
		LogAndRespError(c, err, err.Error())
		return
	}

	data, total := templatesData(templates)
	meta, links, err := UpdateMetaLinks(c, meta, total, nil, params...)
	if err != nil {
		return // Error handled in method itself
	}

	creators, err := templatesCreators(db, account)
	if err != nil {
		LogAndRespError(c, err, err.Error())
		return
	}

	templatesMeta := TemplatesMeta{
		ListMeta: *meta,
		Creators: creators,
	}
	resp := TemplatesResponse{
		Data:  data,
		Links: *links,
		Meta:  templatesMeta,
	}
	c.JSON(http.StatusOK, &resp)
}

func templatesQuery(db *gorm.DB, filters map[string]FilterData, account int, groups map[string]string) *gorm.DB {
	subq := database.Systems(db, account, groups).
		Select("sp.template_id, count(*) as systems").
		Group("sp.template_id")

	subq, _ = ApplyInventoryFilter(filters, subq, "sp.inventory_id")

	query := db.Table("template as tp").
		Select(TemplateSelect).
		Joins("LEFT JOIN (?) sp ON sp.template_id = tp.id", subq).
		Where("tp.rh_account_id = ?", account)

	return query
}

func templatesData(templates []TemplatesDBLookup) ([]TemplateItem, int) {
	var total int
	if len(templates) > 0 {
		total = templates[0].Total
	}
	data := make([]TemplateItem, len(templates))
	for i := 0; i < len(templates); i++ {
		template := templates[i]
		data[i] = TemplateItem{
			Attributes: template.TemplateItemAttributes,
			ID:         template.ID,
			Type:       "template",
		}
	}
	return data, total
}

func templatesCreators(db *gorm.DB, account int) ([]*string, error) {
	// list of creators for account
	creators := []*string{}
	err := db.Table("template bl").
		Distinct("creator").
		Joins("JOIN rh_account acc ON bl.rh_account_id = acc.id").
		Where("bl.rh_account_id = ?", account).
		Scan(&creators).Error
	if err != nil {
		return nil, err
	}

	if len(creators) == 1 && creators[0] == nil {
		creators = []*string{}
	}
	return creators, nil
}
