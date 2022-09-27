package controllers

import (
	"app/base/database"
	"net/http"

	"app/manager/middlewares"

	"github.com/gin-gonic/gin"
)

type SystemTag struct {
	Key       string `json:"key"`
	Namespace string `json:"namespace"`
	Value     string `json:"value"`
}

type SystemTagItem struct {
	Count int       `json:"count"`
	Tag   SystemTag `json:"tag" gorm:"serializer:json"`
}

type SystemTagsResponse struct {
	Data  []SystemTagItem `json:"data"`
	Links Links           `json:"links"`
	Meta  ListMeta        `json:"meta"`
}

// @Summary Show me systems tags applicable to this application
// @Description Show me systems tags applicable to this application
// @ID listSystemTags
// @Security RhIdentity
// @Produce  json
// @Success 200 {object} SystemTagsResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /tags [get]
func SystemTagListHandler(c *gin.Context) {
	account := c.GetInt(middlewares.KeyAccount)

	// https://stackoverflow.com/questions/33474778/how-to-group-result-by-array-column-in-postgres
	sq := database.Systems(database.Db, account).
		Joins("JOIN inventory.hosts ih ON ih.id = sp.inventory_id").
		Select("jsonb_array_elements(ih.tags) AS tag")

	var tagsWithCount []SystemTagItem
	tx := database.Db.Table("(?) AS sq", sq).
		Select("COUNT(sq.tag) AS count, sq.tag AS tag").
		Group("sq.tag").
		Order("sq.tag").
		Scan(&tagsWithCount)

	if tx.Error != nil {
		LogAndRespError(c, tx.Error, "unable to get tags")
		return
	}

	resp := SystemTagsResponse{
		Data: tagsWithCount,
	}

	c.JSON(http.StatusOK, &resp)
}
