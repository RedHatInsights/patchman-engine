package controllers

import (
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"app/manager/middlewares"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

//nolint: lll
type SystemPackagesAttrs struct {
	Name        string `json:"name" csv:"name" query:"pn.name" gorm:"column:name"`
	EVRA        string `json:"evra" csv:"evra" query:"p.evra" gorm:"column:evra"`
	Summary     string `json:"summary" csv:"summary" query:"sum.value" gorm:"column:summary"`
	Description string `json:"description" csv:"description" query:"descr.value" gorm:"column:description"`
	Updatable   bool   `json:"updatable" csv:"updatable" query:"(COALESCE(json_array_length(spkg.update_data::json),0) > 0)" gorm:"column:updatable"`
}

type SystemPackageData struct {
	SystemPackagesAttrs
	Updates []models.PackageUpdate `json:"updates"`
}
type SystemPackageResponse struct {
	Data  []SystemPackageData `json:"data"`
	Meta  ListMeta            `json:"meta"`
	Links Links               `json:"links"`
}

var SystemPackagesSelect = fmt.Sprintf("%s,spkg.update_data as updates", database.MustGetSelect(&SystemPackagesAttrs{}))
var SystemPackagesFields = database.MustGetQueryAttrs(&SystemPackagesAttrs{})
var SystemPackagesOpts = ListOpts{
	Fields:         SystemPackagesFields,
	DefaultFilters: nil,
	DefaultSort:    "name",
	SearchFields:   []string{"pn.name", "sum.value"},
	TotalFunc:      CountRows,
}

type SystemPackageDBLoad struct {
	SystemPackagesAttrs
	Updates []byte `json:"updates" query:"spkg.update_data" gorm:"column:updates"`
}

func systemPackageQuery(account int, inventoryID string) *gorm.DB {
	query := database.SystemPackages(database.Db, account).
		Joins("JOIN strings AS descr ON p.description_hash = descr.id").
		Joins("JOIN strings AS sum ON p.summary_hash = sum.id").
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
	q := systemPackageQuery(account, inventoryID)
	q, meta, links, err := ListCommon(q, c, nil, fmt.Sprintf("/systems/%s/packages", inventoryID), SystemPackagesOpts)
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

	response := SystemPackageResponse{
		Data:  make([]SystemPackageData, len(loaded)),
		Meta:  *meta,
		Links: *links,
	}
	for i, sp := range loaded {
		response.Data[i].SystemPackagesAttrs = sp.SystemPackagesAttrs
		if sp.Updates == nil {
			continue
		}
		if err := json.Unmarshal(sp.Updates, &response.Data[i].Updates); err != nil {
			panic(err)
		}
	}

	c.JSON(200, response)
}
