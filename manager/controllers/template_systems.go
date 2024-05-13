package controllers

import (
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"app/manager/middlewares"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

var templateSystemFields = database.MustGetQueryAttrs(&TemplateSystemsDBLookup{})
var templateSystemSelect = database.MustGetSelect(&TemplateSystemsDBLookup{})
var TemplateSystemOpts = ListOpts{
	Fields:         templateSystemFields,
	DefaultFilters: map[string]FilterData{},
	DefaultSort:    "-display_name",
	StableSort:     "id",
	SearchFields:   []string{"sp.display_name"},
}

type TemplateSystemsDBLookup struct {
	SystemIDAttribute
	// a helper to get total number of systems
	MetaTotalHelper
	TemplateSystemAttributes
}

type TemplateSystemAttributes struct {
	SystemDisplayName
	OSAttributes
	InstallableAdvisories
	ApplicableAdvisories
	SystemTags
	SystemGroups
	SystemLastUpload
}

type TemplateSystemItem struct {
	Attributes TemplateSystemAttributes `json:"attributes"`
	// Template system inventory ID (uuid format)
	InventoryID string `json:"inventory_id"`
	// Document type name
	Type string `json:"type"`
}

type TemplateSystemsResponse struct {
	Data  []TemplateSystemItem `json:"data"`
	Links Links                `json:"links"`
	Meta  ListMeta             `json:"meta"`
}

func getTemplateID(c *gin.Context, tx *gorm.DB, account int, uuid string) (int64, error) {
	var id int64
	if !utils.IsValidUUID(uuid) {
		err := errors.Errorf("Invalid template uuid: %s", uuid)
		LogAndRespNotFound(c, err, err.Error())
		return 0, err
	}
	err := tx.Model(&models.Template{}).
		Select("id").
		Where("rh_account_id = ? AND uuid = ?::uuid ", account, uuid).
		// use Find() not First() otherwise it returns error "no rows found" if uuid is not present
		Find(&id).Error
	if err != nil {
		LogAndRespError(c, err, "database error")
		return 0, err
	}
	if id == 0 {
		err := errors.New("Template not found")
		LogAndRespNotFound(c, err, err.Error())
		return 0, err
	}
	return id, nil
}

func templateSystemsQuery(c *gin.Context, account int, groups map[string]string) (*gorm.DB, Filters, error) {
	templateUUID := c.Param("template_id")
	db := middlewares.DBFromContext(c)

	templateID, err := getTemplateID(c, db, account, templateUUID)
	if err != nil {
		// respose set in getTemplateID()
		return nil, nil, err
	}

	query := database.Systems(db, account, groups).
		Where("sp.template_id = ?", templateID).
		Select(templateSystemSelect)

	filters, err := ParseAllFilters(c, TemplateSystemOpts)
	if err != nil {
		return nil, nil, err
	} // Error handled in method itself
	query, _ = ApplyInventoryFilter(filters, query, "sp.inventory_id")
	return query, filters, nil
}

func templateSystemsCommon(c *gin.Context, account int, groups map[string]string,
) (*gorm.DB, *ListMeta, []string, error) {
	query, filters, err := templateSystemsQuery(c, account, groups)
	if err != nil {
		return nil, nil, nil, err
	} // Error handled in method itself

	query, meta, params, err := ListCommon(query, c, filters, TemplateSystemOpts)
	if err != nil {
		// Error handling and setting of result code & content is done in ListCommon
		return nil, nil, nil, err
	}

	return query, meta, params, err
}

func templateSystemData(templateSystems []TemplateSystemsDBLookup) ([]TemplateSystemItem, int) {
	var total int
	nSystems := len(templateSystems)
	if nSystems > 0 {
		total = templateSystems[0].Total
	}
	data := make([]TemplateSystemItem, nSystems)
	for i := 0; i < nSystems; i++ {
		data[i] = TemplateSystemItem{
			Attributes:  templateSystems[i].TemplateSystemAttributes,
			InventoryID: templateSystems[i].ID,
			Type:        "template_system",
		}
	}
	return data, total
}

// nolint: lll
// @Summary Show me all systems belonging to a template
// @Description  Show me all systems applicable to a template
// @ID listTemplateSystems
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    template_id    path    string  true    "Template ID"
// @Param    limit          query   int     false   "Limit for paging"
// @Param    offset         query   int     false   "Offset for paging"
// @Param    sort           query   string  false   "Sort field"    Enums(id,display_name,os,installable_rhsa_count,installable_rhba_count,installable_rhea_count,installable_other_count,applicable_rhsa_count,applicable_rhba_count,applicable_rhea_count,applicable_other_count,last_upload,groups)
// @Param    search         query   string  false   "Find matching text"
// @Param    filter[display_name]           query   string  false "Filter"
// @Param    filter[os]           			query   string  false "Filter"
// @Param    tags           query   []string  false "Tag filter"
// @Param    filter[group_name] 									query []string 	false "Filter systems by inventory groups"
// @Param    filter[system_profile][sap_system]						query string  	false "Filter only SAP systems"
// @Param    filter[system_profile][sap_sids]						query []string  false "Filter systems by their SAP SIDs"
// @Param    filter[system_profile][ansible]						query string 	false "Filter systems by ansible"
// @Param    filter[system_profile][ansible][controller_version]	query string 	false "Filter systems by ansible version"
// @Param    filter[system_profile][mssql]							query string 	false "Filter systems by mssql version"
// @Param    filter[system_profile][mssql][version]					query string 	false "Filter systems by mssql version"
// @Success 200 {object} TemplateSystemsResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /templates/{template_id}/systems [get]
func TemplateSystemsListHandler(c *gin.Context) {
	account := c.GetInt(utils.KeyAccount)
	groups := c.GetStringMapString(utils.KeyInventoryGroups)

	query, meta, params, err := templateSystemsCommon(c, account, groups)
	if err != nil {
		return
	} // Error handled in method itself

	var templateSystems []TemplateSystemsDBLookup
	err = query.Find(&templateSystems).Error
	if err != nil {
		LogAndRespError(c, err, err.Error())
		return
	}

	data, total := templateSystemData(templateSystems)
	meta, links, err := UpdateMetaLinks(c, meta, total, nil, params...)
	if err != nil {
		return // Error handled in method itself
	}
	var resp = TemplateSystemsResponse{
		Data:  data,
		Links: *links,
		Meta:  *meta,
	}
	c.JSON(http.StatusOK, &resp)
}

// nolint: lll
// @Summary Show me all systems belonging to a template
// @Description  Show me all systems applicable to a template
// @ID listTemplateSystemsIds
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    template_id    path    string  true    "Template ID"
// @Param    limit          query   int     false   "Limit for paging, set -1 to return all"
// @Param    offset         query   int     false   "Offset for paging"
// @Param    sort           query   string  false   "Sort field"    Enums(id,display_name,os,installable_rhsa_count,installable_rhba_count,installable_rhea_count,installable_other_count,applicable_rhsa_count,applicable_rhba_count,applicable_rhea_count,applicable_other_count,last_upload)
// @Param    search         query   string  false   "Find matching text"
// @Param    filter[display_name]           query   string  false "Filter"
// @Param    filter[os]           			query   string  false "Filter"
// @Param    tags           query   []string  false "Tag filter"
// @Success 200 {object} IDsPlainResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /ids/templates/{template_id}/systems [get]
func TemplateSystemsListIDsHandler(c *gin.Context) {
	account := c.GetInt(utils.KeyAccount)
	groups := c.GetStringMapString(utils.KeyInventoryGroups)

	query, meta, _, err := templateSystemsCommon(c, account, groups)
	if err != nil {
		return
	} // Error handled in method itself

	var sids []SystemsID

	if err = query.Scan(&sids).Error; err != nil {
		LogAndRespError(c, err, "db error")
		return
	}

	resp, err := systemsIDs(c, sids, meta)
	if err != nil {
		return // Error handled in method itself
	}
	c.JSON(http.StatusOK, &resp)
}
