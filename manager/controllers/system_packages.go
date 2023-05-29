package controllers

import (
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"app/manager/middlewares"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// nolint: lll
type SystemPackagesAttrsCommon struct {
	Name        string `json:"name" csv:"name" query:"pn.name" gorm:"column:name"`
	EVRA        string `json:"evra" csv:"evra" query:"p.evra" gorm:"column:evra"`
	Summary     string `json:"summary" csv:"summary" query:"sum.value" gorm:"column:summary"`
	Description string `json:"description" csv:"description" query:"descr.value" gorm:"column:description"`
	Updatable   bool   `json:"updatable" csv:"updatable" query:"(COALESCE(jsonb_array_length(spkg.update_data),0) > 0)" gorm:"column:updatable"`
}

type SystemPackagesAttrsV2 struct {
	SystemPackagesAttrsCommon
}

// nolint: lll
type SystemPackagesAttrsV3 struct {
	SystemPackagesAttrsCommon
	UpdateStatus string `json:"update_status" csv:"update_status" query:"update_status(spkg.update_data)" gorm:"column:update_status"`
}

type SystemPackageData struct {
	SystemPackagesAttrsV3
	Updates []models.PackageUpdate `json:"updates"`
}
type SystemPackageResponse struct {
	Data  []SystemPackageData `json:"data"`
	Meta  ListMeta            `json:"meta"`
	Links Links               `json:"links"`
}

var SystemPackagesSelect = database.MustGetSelect(&SystemPackageDBLoad{})
var SystemPackagesFields = database.MustGetQueryAttrs(&SystemPackagesAttrsV3{})
var SystemPackagesOpts = ListOpts{
	Fields:         SystemPackagesFields,
	DefaultFilters: nil,
	DefaultSort:    "name",
	StableSort:     "package_id",
	SearchFields:   []string{"pn.name", "sum.value"},
}

type SystemPackageDBLoad struct {
	SystemPackagesAttrsV3
	Updates []byte `json:"updates" query:"spkg.update_data" gorm:"column:updates"`
	// a helper to get total number of systems
	MetaTotalHelper
}

func systemPackageQuery(db *gorm.DB, account int, inventoryID string) *gorm.DB {
	query := database.SystemPackages(db, account).
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
// @Param    limit          query   int     false   "Limit for paging, set -1 to return all"
// @Param    offset         query   int     false   "Offset for paging"
// @Param    search          query   string  false   "Find matching text"
// @Param    filter[name]            query   string  false "Filter"
// @Param    filter[description]     query   string  false "Filter"
// @Param    filter[evra]            query   string  false "Filter"
// @Param    filter[summary]         query   string  false "Filter"
// @Param    filter[updatable]       query   bool    false "Filter"
// @Success 200 {object} SystemPackageResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /systems/{inventory_id}/packages [get]
func SystemPackagesHandler(c *gin.Context) {
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

	var loaded []SystemPackageDBLoad
	db := middlewares.DBFromContext(c)
	q := systemPackageQuery(db, account, inventoryID)
	q, meta, params, err := ListCommon(q, c, nil, SystemPackagesOpts)
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

	var total int
	if len(loaded) > 0 {
		total = loaded[0].Total
	}
	data := make([]SystemPackageData, len(loaded))
	for i, sp := range loaded {
		data[i].SystemPackagesAttrsV3 = sp.SystemPackagesAttrsV3
		if sp.Updates == nil {
			continue
		}
		if err := json.Unmarshal(sp.Updates, &data[i].Updates); err != nil {
			panic(err)
		}
	}
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
