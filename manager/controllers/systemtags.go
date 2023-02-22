package controllers

import (
	"app/base/database"
	"errors"
	"net/http"

	"app/manager/middlewares"

	"github.com/gin-gonic/gin"
)

type SystemTag struct {
	Key       string `json:"key"`
	Namespace string `json:"namespace"`
	Value     string `json:"value"`
}

var SystemTagSelect = database.MustGetSelect(&SystemTagDBLookup{})

type SystemTagDBLookup struct {
	// a helper to get total number of systems
	MetaTotalHelper

	SystemTagItem
}

type SystemTagItem struct {
	Count int       `json:"count" query:"count(sq.tag)" gorm:"column:cnt"`
	Tag   SystemTag `json:"tag" query:"sq.tag" gorm:"serializer:json;column:tag"`
}

type SystemTagsResponse struct {
	Data  []SystemTagItem `json:"data"`
	Links Links           `json:"links"`
	Meta  ListMeta        `json:"meta"`
}

var SystemTagsOpts = ListOpts{
	Fields: database.AttrMap{
		"tag": {
			OrderQuery: "sq.tag",
		},
		"count": {
			OrderQuery: "COUNT(sq.tag)",
		},
	},
	StableSort:  "tag",
	DefaultSort: "tag",
}

// @Summary Show me systems tags applicable to this application
// @Description Show me systems tags applicable to this application
// @ID listSystemTags
// @Security RhIdentity
// @Produce  json
// @Param	sort	query	string	false	"Sort field" Enums(tag, count)
// @Param	limit	query	int		fals	"Limit for paging, set -1 to return all"
// @Param 	offset	query	int		false	"Offset for paging"
// @Success 200 {object} SystemTagsResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /tags [get]
func SystemTagListHandler(c *gin.Context) {
	var err error
	account := c.GetInt(middlewares.KeyAccount)

	db := middlewares.DBFromContext(c)
	// https://stackoverflow.com/questions/33474778/how-to-group-result-by-array-column-in-postgres
	sq := database.Systems(db, account).
		Joins("JOIN inventory.hosts ih ON ih.id = sp.inventory_id").
		Select("jsonb_array_elements(ih.tags) AS tag")

	query := db.Table("(?) AS sq", sq).
		Select(SystemTagSelect).
		Group("sq.tag")

	tx, meta, params, err := ListCommon(query, c, nil, SystemTagsOpts)
	if !checkSortMeta(meta.Sort) {
		LogAndRespBadRequest(c, errors.New("invalid sort field(s)"), "invalid sort")
		return
	}
	if err != nil {
		// error handling is done within ListCommon
		return
	}

	var tagsWithCount []SystemTagDBLookup
	tx = tx.Scan(&tagsWithCount)
	if tx.Error != nil {
		LogAndRespError(c, tx.Error, "unable to get tags")
		return
	}
	var total int
	if len(tagsWithCount) > 0 {
		total = tagsWithCount[0].Total
	}
	data := make([]SystemTagItem, len(tagsWithCount))
	for i, sp := range tagsWithCount {
		data[i] = sp.SystemTagItem
	}
	meta, links, err := UpdateMetaLinks(c, meta, total, nil, params...)
	if err != nil {
		return // Error handled in method itself
	}
	resp := SystemTagsResponse{
		Data:  data,
		Meta:  *meta,
		Links: *links,
	}

	c.JSON(http.StatusOK, &resp)
}

// check for sort fields and disallow special case of hardcoded sort by "id",
// as it is unavailable for aggregated SQLs
func checkSortMeta(sort []string) bool {
	for _, sortField := range sort {
		if sortField == "id" {
			return false
		}
	}
	return true
}
