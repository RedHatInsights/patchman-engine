package controllers

import (
	"github.com/gin-gonic/gin"
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
// @Accept   json
// @Produce  json
// @Param    advisory_id    path    string   true "Advisory ID"
// @Success 200 {object} AdvisoryDetailResponse
// @Router /api/patch/v1/advisories/{advisory_id} [get]
func AdvisoryDetailHandler(c *gin.Context) {
	var resp = AdvisoryDetailResponse{
		Data: AdvisoryDetailItem{
			Attributes: AdvisoryDetailAttributes{
				Description: "A padding oracle flaw was found in the Secure Sockets Layer version 2.0 (SSLv2) protocol...",
				Severity: nil,
				ModifiedDate: time.Now(),
				PublicDate: time.Now(),
				Topic: "A new kpatch-patch-4_18_0-147_0_3 package is now available for Red Hat Enterprise Linux 8.",
				Synopsis: "new package: kpatch-patch-4_18_0-147_0_3",
				Solution: "Before applying this update, make sure all previously released errata relevant to your system " +
					"have been applied.",
				Fixes: nil,
				Cves: []string{},
				References: []string{},
			},
			ID: "RHEA-2019:3902",
	        Type: "advisory",
		}}
	c.JSON(http.StatusOK, &resp)
	return
}
