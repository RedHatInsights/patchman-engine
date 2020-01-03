package controllers

import (
	"app/base/database"
	"app/base/models"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"net/http"
	"time"
)

type AdvisoryDetailResponse struct {
	Data  AdvisoryDetailItem     `json:"data"`
}

type AdvisoryDetailItem struct {
	Attributes   AdvisoryDetailAttributes `json:"attributes"`
	ID           string                   `json:"id"`
	Type         string                   `json:"type"`
}

type AdvisoryDetailAttributes struct {
	Description  string     `json:"description"`
	Severity     *string    `json:"severity"`
	ModifiedDate time.Time  `json:"modified_date"`
	PublicDate   time.Time  `json:"public_date"`
	Topic        string     `json:"topic"`
	Synopsis     string     `json:"synopsis"`
	Solution     string     `json:"solution"`
	Fixes        *string    `json:"fixes"`
	Cves         []string   `json:"cves"`
	References   []string   `json:"references"`
}

// @Summary Show me details an advisory by given advisory name
// @Description Show me details an advisory by given advisory name
// @ID detailAdvisory
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    advisory_id    path    string   true "Advisory ID"
// @Success 200 {object} AdvisoryDetailResponse
// @Router /api/patch/v1/advisories/{advisory_id} [get]
func AdvisoryDetailHandler(c *gin.Context) {
	advisoryName := c.Param("advisory_id")
	if advisoryName == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{"advisory_id param not found"})
		return
	}

	var advisory models.AdvisoryMetadata
	err := database.Db.Where("name = ?", advisoryName).First(&advisory).Error
	if gorm.IsRecordNotFoundError(err) {
		LogAndRespNotFound(c, err, "advisory not found")
		return
	}

	if err != nil {
		LogAndRespError(c, err, "database error")
		return
	}

	var resp = AdvisoryDetailResponse{
		Data: AdvisoryDetailItem{
			Attributes: AdvisoryDetailAttributes{
				Description: advisory.Description,
				Severity: nil,
				ModifiedDate: advisory.ModifiedDate,
				PublicDate: advisory.PublicDate,
				Topic: advisory.Summary,
				Synopsis: advisory.Synopsis,
				Solution: advisory.Solution,
				Fixes: nil,
				Cves: []string{}, // TODO joins
				References: []string{}, // TODO joins
			},
			ID: advisory.Name,
	        Type: "advisory",
		}}
	c.JSON(http.StatusOK, &resp)
	return
}
