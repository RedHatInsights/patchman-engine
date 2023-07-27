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

// nolint: lll
type SystemPackagesAttrsCommon struct {
	Name        string `json:"name" csv:"name" query:"pn.name" gorm:"column:name"`
	EVRA        string `json:"evra" csv:"evra" query:"p.evra" gorm:"column:evra"`
	Summary     string `json:"summary" csv:"summary" query:"sum.value" gorm:"column:summary"`
	Description string `json:"description" csv:"description" query:"descr.value" gorm:"column:description"`
	Updatable   bool   `json:"updatable" csv:"updatable" query:"(update_status(spkg.update_data) = 'Installable')" gorm:"column:updatable"`
}

type SystemPackageUpdates struct {
	Updates []models.PackageUpdate `json:"updates"`
}

type SystemPackagesAttrsV2 struct {
	SystemPackagesAttrsCommon
}

// nolint: lll
type SystemPackagesAttrsV3 struct {
	SystemPackagesAttrsCommon
	UpdateStatus string `json:"update_status" csv:"update_status" query:"update_status(spkg.update_data)" gorm:"column:update_status"`
}

type SystemPackageDataV2 struct {
	SystemPackagesAttrsV2
	SystemPackageUpdates
}
type SystemPackageDataV3 struct {
	SystemPackagesAttrsV3
	SystemPackageUpdates
}
type SystemPackageResponseV2 struct {
	Data  []SystemPackageDataV2 `json:"data"`
	Meta  ListMeta              `json:"meta"`
	Links Links                 `json:"links"`
}
type SystemPackageResponseV3 struct {
	Data  []SystemPackageDataV3 `json:"data"`
	Meta  ListMeta              `json:"meta"`
	Links Links                 `json:"links"`
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

func systemPackageQuery(db *gorm.DB, account int, groups map[string]string, inventoryID string) *gorm.DB {
	query := database.SystemPackages(db, account, groups).
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
// @Param    filter[update_status]   query   string  false "Filter"
// @Success 200 {object} SystemPackageResponseV3
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /systems/{inventory_id}/packages [get]
func SystemPackagesHandler(c *gin.Context) {
	account := c.GetInt(middlewares.KeyAccount)
	apiver := c.GetInt(middlewares.KeyApiver)
	groups := c.GetStringMapString(middlewares.KeyInventoryGroups)

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
	q := systemPackageQuery(db, account, groups, inventoryID)
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

	total, data := buildSystemPackageData(loaded)
	meta, links, err := UpdateMetaLinks(c, meta, total, nil, params...)
	if err != nil {
		return // Error handled in method itself
	}
	if apiver < 3 {
		dataV2 := systemPackageV3toV2(data)
		var resp = SystemPackageResponseV2{
			Data:  dataV2,
			Links: *links,
			Meta:  *meta,
		}
		c.JSON(http.StatusOK, &resp)
		return
	}
	response := SystemPackageResponseV3{
		Data:  data,
		Meta:  *meta,
		Links: *links,
	}

	c.JSON(http.StatusOK, response)
}

func buildSystemPackageData(loaded []SystemPackageDBLoad) (int, []SystemPackageDataV3) {
	var total int
	if len(loaded) > 0 {
		total = loaded[0].Total
	}
	data := make([]SystemPackageDataV3, len(loaded))
	for i, sp := range loaded {
		data[i].SystemPackagesAttrsV3 = sp.SystemPackagesAttrsV3
		if sp.Updates == nil {
			continue
		}
		// keep only latest installable and applicable
		installable, applicable := findLatestEVRA(sp)
		if installable.EVRA != sp.EVRA {
			data[i].Updates = append(data[i].Updates, installable)
		}
		if applicable.EVRA != sp.EVRA {
			data[i].Updates = append(data[i].Updates, applicable)
		}
	}
	return total, data
}

func systemPackageV3toV2(pkgs []SystemPackageDataV3) []SystemPackageDataV2 {
	nPkgs := len(pkgs)
	pkgsV2 := make([]SystemPackageDataV2, nPkgs)
	for i := 0; i < nPkgs; i++ {
		pkgsV2[i] = SystemPackageDataV2{
			SystemPackagesAttrsV2: SystemPackagesAttrsV2{
				SystemPackagesAttrsCommon: pkgs[i].SystemPackagesAttrsCommon,
			},
			SystemPackageUpdates: pkgs[i].SystemPackageUpdates,
		}
	}
	return pkgsV2
}
