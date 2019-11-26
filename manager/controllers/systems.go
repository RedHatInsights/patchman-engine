package controllers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

type SystemsResponse struct {
	Data  []SystemItem `json:"data"`
	Links Links  `json:"links"`
	Meta  SystemsMeta  `json:"meta"`
}

type SystemsMeta struct {
	DataFormat string  `json:"data_format"`
	Filter     *string `json:"filter"`
	Limit      int     `json:"limit"`
	Offset     int     `json:"offset"`
	Page       int     `json:"page"`
	PageSize   int     `json:"page_size"`
	Pages      int     `json:"pages"`
	Enabled    bool    `json:"enabled"`
	TotalItems int     `json:"total_items"`
}

// @Summary Show me all my systems
// @Description Show me all my systems
// @Accept   json
// @Produce  json
// @Success 200 {object} SystemsResponse
// @Router /api/patch/v1/systems [get]
func SystemsListHandler(c *gin.Context) {
	var resp = SystemsResponse{
		Data: []SystemItem{{
			Attributes: SystemItemAttributes{
				LastEvaluation: time.Now(),
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
			First: "/api/patch/v1/systems?offset=0&limit=25&data_format=json&show_all=True",
			Last: "/api/patch/v1/systems?offset=21475&limit=25&data_format=json&show_all=True",
			Next: "/api/patch/v1/systems?offset=25&limit=25&data_format=json&show_all=True",
			Previous: nil,
		},
		Meta:  SystemsMeta{
			DataFormat: "json",
			Filter: nil,
			Limit: 25,
			Offset: 0,
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
