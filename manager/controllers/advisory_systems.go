package controllers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

type AdvisorySystemsResponse struct {
	Data  []SystemItem        `json:"data"`
	Links Links               `json:"links"`
	Meta  AdvisorySystemsMeta `json:"meta"`
}

type AdvisorySystemsMeta struct {
	DataFormat string  `json:"data_format"`
	Filter     *string `json:"filter"`
	Limit      int     `json:"limit"`
	Offset     int     `json:"offset"`
	Advisory   string  `json:"advisory"`
	Page       int     `json:"page"`
	PageSize   int     `json:"page_size"`
	Pages      int     `json:"pages"`
	Enabled    bool    `json:"enabled"`
	TotalItems int     `json:"total_items"`
}

// @Summary Show me systems on which the given advisory is applicable
// @Description Show me systems on which the given advisory is applicable
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    advisory_id    path    string   true "Advisory ID"
// @Success 200 {object} AdvisorySystemsResponse
// @Router /api/patch/v1/advisories/{advisory_id}/systems [get]
func AdvisorySystemsListHandler(c *gin.Context) {
	le := time.Now()
	var resp = AdvisorySystemsResponse{
		Data: []SystemItem{{
			Attributes: SystemItemAttributes{
				LastEvaluation: &le,
				LastUpload:     nil,
				RhsaCount:      2,
				RhbaCount:      5,
				RheaCount:      1,
				Enabled:        true,
		},
			Id: "b89e2f25-8b28-4e1c-9879-947143c2cee9",
			Type: "system" },
		},
		Links: Links{
			First: "/api/patch/v1/advisories/$ADVISORY_ID/systems?offset=0&limit=25&data_format=json&show_all=True",
			Last: "/api/patch/v1/advisories/$ADVISORY_ID/systems?offset=21475&limit=25&data_format=json&show_all=True",
			Next: nil,
			Previous: nil,
		},
		Meta:  AdvisorySystemsMeta{
			DataFormat: "json",
			Filter: nil,
			Limit: 25,
			Offset: 0,
			Advisory: "RHEA-2019:3902",
			Page: 1,
			PageSize: 25,
			Pages: 10,
			Enabled: true,
			TotalItems: 250,
		},
	}
	c.JSON(http.StatusOK, &resp)
	return
}
