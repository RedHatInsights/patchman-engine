package controllers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

type SystemDetailResponse struct {
	Data  SystemItem     `json:"data"`
}

// @Summary Show me details about a system by given inventory id
// @Description Show me details about a system by given inventory id
// @Accept   json
// @Produce  json
// @Param    inventory_id    path    string   true "Inventory ID"
// @Success 200 {object} SystemDetailResponse
// @Router /api/patch/v1/systems/{inventory_id} [get]
func SystemDetailHandler(c *gin.Context) {
	le := time.Now()
	var resp = SystemDetailResponse{
		Data: SystemItem{
			Attributes: SystemItemAttributes{
				LastEvaluation: &le,
				LastUpload:     nil,
				RhsaCount:      2,
				RhbaCount:      5,
				RheaCount:      1,
				Enabled:        true },
			Id: "b89e2f25-8b28-4e1c-9879-947143c2cee9",
			Type: "system",
		},
	}
	c.JSON(http.StatusOK, &resp)
	return
}
