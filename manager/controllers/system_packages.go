package controllers

import (
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"app/manager/middlewares"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SystemPackageUpdates struct {
	Updates []models.PackageUpdate `json:"updates"`
}

// nolint: lll
type SystemPackagesAttrs struct {
	Name         string `json:"name" csv:"name" query:"pn.name" gorm:"column:name"`
	EVRA         string `json:"evra" csv:"evra" query:"p.evra" gorm:"column:evra"`
	Summary      string `json:"summary" csv:"summary" query:"sum.value" gorm:"column:summary"`
	Description  string `json:"description" csv:"description" query:"descr.value" gorm:"column:description"`
	Updatable    bool   `json:"updatable" csv:"updatable" query:"(spkg.installable_id is not null)" gorm:"column:updatable"`
	UpdateStatus string `json:"update_status" csv:"update_status" query:"CASE WHEN spkg.installable_id is not null THEN 'Installable' WHEN spkg.applicable_id is not null THEN 'Applicable' ELSE 'None' END" gorm:"column:update_status"`
}

type SystemPackageData struct {
	SystemPackagesAttrs
	SystemPackageUpdates
}

type SystemPackageResponse struct {
	Data  []SystemPackageData `json:"data"`
	Meta  ListMeta            `json:"meta"`
	Links Links               `json:"links"`
}

var SystemPackagesSelect = database.MustGetSelect(&SystemPackageDBLoad{})
var SystemPackagesFields = database.MustGetQueryAttrs(&SystemPackagesAttrs{})
var SystemPackagesOpts = ListOpts{
	Fields:         SystemPackagesFields,
	DefaultFilters: nil,
	DefaultSort:    "name",
	StableSort:     "package_id",
	SearchFields:   []string{"pn.name", "sum.value"},
}

type SystemPackageDBLoad struct {
	SystemPackagesAttrs
	// helper to get Updates
	InstallableEVRA string `json:"-" csv:"-" query:"pi.evra" gorm:"column:installable_evra"`
	ApplicableEVRA  string `json:"-" csv:"-" query:"pa.evra" gorm:"column:applicable_evra"`
	// a helper to get total number of systems
	MetaTotalHelper
}

func systemPackageQuery(db *gorm.DB, account int, groups map[string]string, inventoryID string) *gorm.DB {
	query := database.SystemPackages(db, account, groups).
		Joins("LEFT JOIN package pi ON pi.id = spkg.installable_id").
		Joins("LEFT JOIN package pa ON pa.id = spkg.applicable_id").
		Joins("LEFT JOIN strings AS descr ON p.description_hash = descr.id").
		Joins("LEFT JOIN strings AS sum ON p.summary_hash = sum.id").
		Select(SystemPackagesSelect).
		Where("sp.inventory_id = ?::uuid", inventoryID)

	return query
}

// @Summary Show me details about a system packages by given inventory id
// @Description Show me details about a system packages by given inventory id
// @ID systemPackages
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    inventory_id    path    string   true "Inventory ID"
// @Param    limit          query   int     false   "Limit for paging"
// @Param    offset         query   int     false   "Offset for paging"
// @Param    search          query   string  false   "Find matching text"
// @Param    filter[name]            query   string  false "Filter"
// @Param    filter[description]     query   string  false "Filter"
// @Param    filter[evra]            query   string  false "Filter"
// @Param    filter[summary]         query   string  false "Filter"
// @Param    filter[updatable]       query   bool    false "Filter"
// @Param    filter[update_status]   query   string  false "Filter"
// @Success 200 {object} SystemPackageResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /systems/{inventory_id}/packages [get]
func SystemPackagesHandler(c *gin.Context) {
	account := c.GetInt(utils.KeyAccount)
	groups := c.GetStringMapString(utils.KeyInventoryGroups)

	inventoryID := c.Param("inventory_id")
	if inventoryID == "" {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{Error: "inventory_id param not found"})
		return
	}

	if !utils.IsValidUUID(inventoryID) {
		LogAndRespBadRequest(c, errors.New("bad request"), "incorrect inventory_id format")
		return
	}
	filters, err := ParseAllFilters(c, SystemPackagesOpts)
	if err != nil {
		// Error handled in method itself
		return
	}
	var loaded []SystemPackageDBLoad
	db := middlewares.DBFromContext(c)
	q := systemPackageQuery(db, account, groups, inventoryID)
	q, meta, params, err := ListCommon(q, c, filters, SystemPackagesOpts)
	if err != nil {
		return
	}

	err = q.Find(&loaded).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		LogAndRespNotFound(c, err, "inventory_id not found")
		return
	}

	if err != nil {
		LogAndRespError(c, err, "database error")
		return
	}

	total, data := buildSystemPackageData(loaded)
	meta, links, err := UpdateMetaLinks(c, meta, total, nil, params...)
	if err != nil {
		return // Error handled in method itself
	}
	response := SystemPackageResponse{
		Data:  data,
		Meta:  *meta,
		Links: *links,
	}

	c.JSON(http.StatusOK, response)
}

func buildSystemPackageData(loaded []SystemPackageDBLoad) (int, []SystemPackageData) {
	var total int
	if len(loaded) > 0 {
		total = loaded[0].Total
	}
	data := make([]SystemPackageData, len(loaded))
	for i, sp := range loaded {
		data[i].SystemPackagesAttrs = sp.SystemPackagesAttrs
		// keep only latest installable and applicable
		if len(sp.InstallableEVRA) > 0 && sp.InstallableEVRA != sp.EVRA {
			data[i].Updates = append(data[i].Updates, models.PackageUpdate{
				EVRA: sp.InstallableEVRA, Status: "Installable",
			})
		}
		if len(sp.ApplicableEVRA) > 0 && sp.ApplicableEVRA != sp.EVRA {
			data[i].Updates = append(data[i].Updates, models.PackageUpdate{
				EVRA: sp.ApplicableEVRA, Status: "Applicable",
			})
		}
	}
	return total, data
}
