package controllers

import (
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"app/manager/middlewares"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

var BaselineSystemFields = database.MustGetQueryAttrs(&BaselineSystemsDBLookup{})
var BaselineSystemSelect = database.MustGetSelect(&BaselineSystemsDBLookup{})
var BaselineSystemSelectV2 = database.MustGetSelect(&BaselineSystemsDBLookupV2{})
var BaselineSystemOpts = ListOpts{
	Fields:         BaselineSystemFields,
	DefaultFilters: map[string]FilterData{},
	DefaultSort:    "-display_name",
	StableSort:     "id",
	SearchFields:   []string{"sp.display_name"},
}

type BaselineSystemsDBLookupV2 struct {
	SystemIDAttribute
	// a helper to get total number of systems
	MetaTotalHelper
	BaselineSystemAttributesV2
}

type BaselineSystemsDBLookup struct {
	SystemIDAttribute
	// a helper to get total number of systems
	SystemsMetaTagTotal
	BaselineSystemAttributes
}

type BaselineSystemAttributesV2 struct {
	// Baseline system display name
	DisplayName string `json:"display_name" csv:"display_name" query:"sp.display_name" gorm:"column:display_name" example:"my-baselined-system"` // nolint: lll
}

// nolint: lll
type BaselineSystemAttributes struct {
	BaselineSystemAttributesV2
	OSAttributes
	InstallableAdvisories
	ApplicableAdvisories
	SystemTags
	SystemLastUpload
}

type BaselineSystemItemCommon struct {
	// Baseline system inventory ID (uuid format)
	InventoryID string `json:"inventory_id" example:"00000000-0000-0000-0000-000000000001"`
	// Document type name
	Type string `json:"type" example:"baseline_system"`
}

type BaselineSystemItem struct {
	// Additional baseline system attributes
	Attributes BaselineSystemAttributes `json:"attributes"`
	BaselineSystemItemCommon
}

type BaselineSystemItemV2 struct {
	// Additional baseline system attributes
	Attributes BaselineSystemAttributesV2 `json:"attributes"`
	BaselineSystemItemCommon
}

type BaselineSystemsResponseV2 struct {
	Data  []BaselineSystemItemV2 `json:"data"`
	Links Links                  `json:"links"`
	Meta  ListMeta               `json:"meta"`
}

type BaselineSystemsResponse struct {
	Data  []BaselineSystemItem `json:"data"`
	Links Links                `json:"links"`
	Meta  ListMeta             `json:"meta"`
}

func queryBaselineSystems(c *gin.Context, account, apiver int, groups map[string]string) (*gorm.DB, error) {
	baselineID := c.Param("baseline_id")
	id, err := strconv.ParseInt(baselineID, 10, 64)
	if err != nil {
		LogAndRespBadRequest(c, err, fmt.Sprintf("Invalid baseline_id: %s", baselineID))
		return nil, err
	}

	db := middlewares.DBFromContext(c)
	var exists int64
	err = db.Model(&models.Baseline{}).
		Where("id = ? ", id).Count(&exists).Error
	if err != nil {
		LogAndRespError(c, err, "database error")
		return nil, err
	}
	if exists == 0 {
		LogAndRespNotFound(c, errors.New("Baseline not found"), "Baseline not found")
		return nil, err
	}

	query := buildQueryBaselineSystems(db, account, groups, id, apiver)
	filters, err := ParseTagsFilters(c)
	if err != nil {
		return nil, err
	} // Error handled in method itself
	query, _ = ApplyTagsFilter(filters, query, "sp.inventory_id")
	return query, nil
}

func baselineSystemsCommon(c *gin.Context, account, apiver int, groups map[string]string,
) (*gorm.DB, *ListMeta, []string, error) {
	query, err := queryBaselineSystems(c, account, apiver, groups)
	if err != nil {
		return nil, nil, nil, err
	} // Error handled in method itself

	query, meta, params, err := ListCommon(query, c, nil, BaselineSystemOpts)
	if err != nil {
		// Error handling and setting of result code & content is done in ListCommon
		return nil, nil, nil, err
	}

	return query, meta, params, err
}

// nolint: lll
// @Summary Show me all systems belonging to a baseline
// @Description  Show me all systems applicable to a baseline
// @ID listBaselineSystems
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    baseline_id    path    int     true    "Baseline ID"
// @Param    limit          query   int     false   "Limit for paging, set -1 to return all"
// @Param    offset         query   int     false   "Offset for paging"
// @Param    sort           query   string  false   "Sort field"    Enums(id,display_name,os,installable_rhsa_count,installable_rhba_count,installable_rhea_count,installable_other_count,applicable_rhsa_count,applicable_rhba_count,applicable_rhea_count,applicable_other_count,last_upload)
// @Param    search         query   string  false   "Find matching text"
// @Param    filter[display_name]           query   string  false "Filter"
// @Param    filter[os]           			query   string  false "Filter"
// @Param    tags           query   []string  false "Tag filter"
// @Param    filter[system_profile][sap_system]						query string  	false "Filter only SAP systems"
// @Param    filter[system_profile][sap_sids][in]					query []string  false "Filter systems by their SAP SIDs"
// @Param    filter[system_profile][ansible]						query string 	false "Filter systems by ansible"
// @Param    filter[system_profile][ansible][controller_version]	query string 	false "Filter systems by ansible version"
// @Param    filter[system_profile][mssql]							query string 	false "Filter systems by mssql version"
// @Param    filter[system_profile][mssql][version]					query string 	false "Filter systems by mssql version"
// @Success 200 {object} BaselineSystemsResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /baselines/{baseline_id}/systems [get]
func BaselineSystemsListHandler(c *gin.Context) {
	account := c.GetInt(middlewares.KeyAccount)
	apiver := c.GetInt(middlewares.KeyApiver)
	groups := c.GetStringMapString(middlewares.KeyInventoryGroups)

	query, meta, params, err := baselineSystemsCommon(c, account, apiver, groups)
	if err != nil {
		return
	} // Error handled in method itself

	var baselineSystems []BaselineSystemsDBLookup
	err = query.Find(&baselineSystems).Error
	if err != nil {
		LogAndRespError(c, err, err.Error())
		return
	}

	data, total := buildBaselineSystemData(baselineSystems)
	meta, links, err := UpdateMetaLinks(c, meta, total, nil, params...)
	if err != nil {
		return // Error handled in method itself
	}
	var resp = BaselineSystemsResponse{
		Data:  data,
		Links: *links,
		Meta:  *meta,
	}
	if apiver < 3 {
		respV2 := baselineSystemResponse2V2(&resp)
		c.JSON(http.StatusOK, respV2)
		return
	}
	c.JSON(http.StatusOK, &resp)
}

// nolint: lll
// @Summary Show me all systems belonging to a baseline
// @Description  Show me all systems applicable to a baseline
// @ID listBaselineSystemsIds
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    baseline_id    path    int     true    "Baseline ID"
// @Param    limit          query   int     false   "Limit for paging, set -1 to return all"
// @Param    offset         query   int     false   "Offset for paging"
// @Param    sort           query   string  false   "Sort field"    Enums(id,display_name,os,installable_rhsa_count,installable_rhba_count,installable_rhea_count,installable_other_count,applicable_rhsa_count,applicable_rhba_count,applicable_rhea_count,applicable_other_count,last_upload)
// @Param    search         query   string  false   "Find matching text"
// @Param    filter[display_name]           query   string  false "Filter"
// @Param    filter[os]           			query   string  false "Filter"
// @Param    tags           query   []string  false "Tag filter"
// @Success 200 {object} IDsResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /ids/baselines/{baseline_id}/systems [get]
func BaselineSystemsListIDsHandler(c *gin.Context) {
	account := c.GetInt(middlewares.KeyAccount)
	apiver := c.GetInt(middlewares.KeyApiver)
	groups := c.GetStringMapString(middlewares.KeyInventoryGroups)
	if apiver < 3 {
		c.AbortWithStatus(404)
		return
	}

	query, meta, _, err := baselineSystemsCommon(c, account, apiver, groups)
	if err != nil {
		return
	} // Error handled in method itself

	var sids []SystemsID

	if err = query.Scan(&sids).Error; err != nil {
		LogAndRespError(c, err, "db error")
		return
	}

	ids, err := systemsIDs(c, sids, meta)
	if err != nil {
		return // Error handled in method itself
	}
	var resp = IDsResponse{IDs: ids}
	c.JSON(http.StatusOK, &resp)
}

func buildQueryBaselineSystems(db *gorm.DB, account int, groups map[string]string, baselineID int64, apiver int,
) *gorm.DB {
	query := database.Systems(db, account, groups).
		Where("sp.baseline_id = ?", baselineID)
	if apiver < 3 {
		query.Select(BaselineSystemSelectV2)
	} else {
		query.Select(BaselineSystemSelect)
	}
	return query
}

func buildBaselineSystemData(baselineSystems []BaselineSystemsDBLookup) ([]BaselineSystemItem, int) {
	var total int
	var err error
	if len(baselineSystems) > 0 {
		total = baselineSystems[0].Total
	}
	data := make([]BaselineSystemItem, len(baselineSystems))
	for i := 0; i < len(baselineSystems); i++ {
		baselineSystems[i].Tags, err = parseSystemTags(baselineSystems[i].TagsStr)
		if err != nil {
			utils.LogDebug("err", err.Error(), "inventory_id", baselineSystems[i].ID, "system tags parsing failed")
		}
		data[i] = BaselineSystemItem{
			Attributes: baselineSystems[i].BaselineSystemAttributes,
			BaselineSystemItemCommon: BaselineSystemItemCommon{
				InventoryID: baselineSystems[i].ID,
				Type:        "baseline_system",
			},
		}
	}
	return data, total
}

func baselineSystemResponse2V2(resp *BaselineSystemsResponse) *BaselineSystemsResponseV2 {
	v2Items := make([]BaselineSystemItemV2, 0, len(resp.Data))
	for _, v := range resp.Data {
		v2Items = append(v2Items, BaselineSystemItemV2{
			Attributes:               v.Attributes.BaselineSystemAttributesV2,
			BaselineSystemItemCommon: v.BaselineSystemItemCommon,
		})
	}
	respV2 := BaselineSystemsResponseV2{
		Data:  v2Items,
		Links: resp.Links,
		Meta:  resp.Meta,
	}
	return &respV2
}
