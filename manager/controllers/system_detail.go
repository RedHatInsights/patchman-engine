package controllers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

type SystemDetailResponse struct {
	Data  []AdvisoryItem `json:"data"`  // advisories items
	Links Links          `json:"links"`
	Meta  AdvisoryMeta   `json:"meta"`
}

// @Summary Show me details about a system by given inventory id
// @Description Show me details about a system by given inventory id
// @Accept   json
// @Produce  json
// @Param    inventory_id    path    string   true "Inventory ID"
// @Success 200 {object} SystemDetailResponse
// @Router /api/patch/v1/systems/{inventory_id} [get]
func SystemDetailHandler(c *gin.Context) {
	var resp = SystemDetailResponse{
		Data: []AdvisoryItem{{
			Attributes: AdvisoryItemAttributes{
				Description: "The kernel-rt packages provide the Real Time Linux Kernel, ...",
				Severity: "Important",
				PublicDate: time.Now(),
				Synopsis: "Important: kernel-rt security update",
				AdvisoryType: 2,
				ApplicableSystems: 6 },
			Id: "RHSA-2019:3908",
			Type: "advisory" },
		},
		Links: Links{
			First: "/api/patch/v1/advisories?offset=0&limit=25&data_format=json&show_all=True",
			Last: "/api/patch/v1/advisories?offset=21475&limit=25&data_format=json&show_all=True",
			Next: "/api/patch/v1/advisories?offset=25&limit=25&data_format=json&show_all=True",
			Previous: nil,
		},
		Meta:  AdvisoryMeta{
			DataFormat: "json",
			Filter: nil,
			Severity: nil,
			Limit: 25,
			Offset: 0,
			Page: 1,
			PageSize: 25,
			Pages: 10,
			PublicFrom: nil,
			PublicTo: nil,
			ShowAll: true,
			Sort: nil,
			TotalItems: 250,
		},
	}
	c.JSON(http.StatusOK, &resp)
	return
}
