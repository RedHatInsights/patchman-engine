package controllers

import (
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
	"time"
)

type AdvisoryDetailResponse struct {
	Data AdvisoryDetailItem `json:"data"`
}

type AdvisoryDetailItem struct {
	Attributes AdvisoryDetailAttributes `json:"attributes"`
	ID         string                   `json:"id"`
	Type       string                   `json:"type"`
}

type AdvisoryDetailAttributes struct {
	Description  string            `json:"description"`
	ModifiedDate time.Time         `json:"modified_date"`
	PublicDate   time.Time         `json:"public_date"`
	Topic        string            `json:"topic"`
	Synopsis     string            `json:"synopsis"`
	Solution     string            `json:"solution"`
	Severity     *int              `json:"severity"`
	Fixes        *string           `json:"fixes"`
	Cves         []string          `json:"cves"`
	Packages     map[string]string `json:"packages"`
	References   []string          `json:"references"`
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
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{Error: "advisory_id param not found"})
		return
	}

	var advisory models.AdvisoryMetadata
	err := database.Db.Where("name = ?", advisoryName).First(&advisory).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		LogAndRespNotFound(c, err, "advisory not found")
		return
	}

	if err != nil {
		LogAndRespError(c, err, "database error")
		return
	}

	cves, err := parseCVEs(advisory.CveList)
	if err != nil {
		LogAndRespError(c, err, "CVEs parsing error")
		return
	}

	packages, err := parsePackages(advisory.PackageData)
	if err != nil {
		LogAndRespError(c, err, "packages parsing error")
		return
	}

	var resp = AdvisoryDetailResponse{
		Data: AdvisoryDetailItem{
			Attributes: AdvisoryDetailAttributes{
				Description:  advisory.Description,
				ModifiedDate: advisory.ModifiedDate,
				PublicDate:   advisory.PublicDate,
				Topic:        advisory.Summary,
				Synopsis:     advisory.Synopsis,
				Solution:     advisory.Solution,
				Severity:     advisory.SeverityID,
				Fixes:        nil,
				Cves:         cves,
				Packages:     packages,
				References:   []string{}, // TODO joins
			},
			ID:   advisory.Name,
			Type: "advisory",
		}}
	c.JSON(http.StatusOK, &resp)
}

func parseCVEs(jsonb []byte) ([]string, error) {
	if jsonb == nil {
		return []string{}, nil
	}

	js := json.RawMessage(string(jsonb))
	b, err := json.Marshal(js)
	if err != nil {
		return nil, err
	}

	var cves []string
	err = json.Unmarshal(b, &cves)
	if err != nil {
		return nil, err
	}
	return cves, nil
}

func parsePackages(jsonb []byte) (map[string]string, error) {
	if jsonb == nil {
		return map[string]string{}, nil
	}

	js := json.RawMessage(string(jsonb))
	b, err := json.Marshal(js)
	if err != nil {
		return nil, err
	}

	var packages map[string]string
	err = json.Unmarshal(b, &packages)
	if err != nil {
		return nil, err
	}
	return packages, nil
}
