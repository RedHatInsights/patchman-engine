package controllers

import (
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"app/manager/middlewares"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/dialects/postgres"
	"net/http"
)

type SystemPackagesAttrs struct {
	Name        string `json:"name" query:"pn.name"`
	EVRA        string `json:"evra" query:"p.evra"`
	Summary     string `json:"summary" query:"sum.value"`
	Description string `json:"description" query:"descr.value"`
	Updatable   bool   `json:"updatable" query:"(json_array_length(spkg.update_data::json) > 0)"`
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
	SearchFields:   []string{"pn.name", "sum.value", "descr.value"},
}

type SystemPackageDBLoad struct {
	SystemPackagesAttrs
	Updates postgres.Jsonb `json:"updates" query:"spkg.update_data"`
}

func systemPackageQuery(account int, inventoryID string) *gorm.DB {
	query := database.Db.
		Table("system_package spkg").
		Joins("inner join system_platform sp on sp.id = spkg.system_id").
		Joins("inner join package p on p.id = spkg.package_id").
		Joins("inner join package_name pn on pn.id = p.name_id").
		Joins("inner join strings sum on sum.id = p.summary_hash").
		Joins("inner join strings descr on descr.id = p.description_hash").
		Where("spkg.rh_account_id = ? and sp.inventory_id = ?::uuid", account, inventoryID)

	if applyInventoryHosts {
		query = query.Joins("JOIN inventory.hosts ih ON ih.id = sp.inventory_id")
	}

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
// @Router /api/patch/v1/systems/{inventory_id}/packages [get]
func SystemPackagesHandler(c *gin.Context) {
	account := c.GetInt(middlewares.KeyAccount)

	inventoryID := c.Param("inventory_id")
	if inventoryID == "" {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{Error: "inventory_id param not found"})
		return
	}

	var loaded []SystemPackageDBLoad
	q := systemPackageQuery(account, inventoryID).Select(SystemPackagesSelect)
	q, meta, links, err := ListCommon(q, c, fmt.Sprintf("/systems/%s/packages", inventoryID), SystemPackagesOpts)
	if err != nil {
		return
	}

	err = q.Find(&loaded).Error
	if gorm.IsRecordNotFoundError(err) {
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
		if sp.Updates.RawMessage == nil {
			continue
		}
		if err := json.Unmarshal(sp.Updates.RawMessage, &response.Data[i].Updates); err != nil {
			panic(err)
		}
	}

	c.JSON(200, response)
}
