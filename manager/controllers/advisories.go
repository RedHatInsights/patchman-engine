package controllers

import (
	"app/base/database"
	"app/manager/middlewares"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"net/http"
	"time"
)

var AdvisoriesSortFields = []string{"type", "synopsis", "public_date"}

type AdvisoriesResponse struct {
	Data  []AdvisoryItem `json:"data"`
	Links Links          `json:"links"`
	Meta  ListMeta       `json:"meta"`
}

type AdvisoryWithApplicableSystems struct {
	Name              string
	Description       string
	Synopsis          string
	PublicDate        time.Time
	AdvisoryTypeID    int
	Severity          *int
	ApplicableSystems int
}

// nolint:lll
// @Summary Show me all applicable advisories for all my systems
// @Description Show me all applicable advisories for all my systems
// @ID listAdvisories
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    limit          query   int     false   "Limit for paging"
// @Param    offset         query   int     false   "Offset for paging"
// @Param    sort           query   string  false   "Sort field"    Enums(id,type,synopsis,public_date)
// @Param    search         query   string  false   "Find matching text"
// @Success 200 {object} AdvisoriesResponse
// @Router /api/patch/v1/advisories [get]
func AdvisoriesListHandler(c *gin.Context) {
	account := c.GetString(middlewares.KeyAccount)

	query := buildQueryAdvisories(account)

	query = ApplySearch(c, query, "am.name", "synopsis", "description")
	query, meta, links, err := ListCommon(query, c, AdvisoriesSortFields, "/api/patch/v1/advisories")
	if err != nil {
		// Error handling and setting of result code & content is done in ListCommon
		return
	}

	var advisories []AdvisoryWithApplicableSystems

	err = query.Find(&advisories).Error
	if err != nil {
		LogAndRespError(c, err, "db error")
		return
	}

	data := buildAdvisoriesData(&advisories)
	var resp = AdvisoriesResponse{
		Data:  *data,
		Links: *links,
		Meta:  *meta,
	}
	c.JSON(http.StatusOK, &resp)
}

func buildQueryAdvisories(account string) *gorm.DB {
	query := database.Db.Table("advisory_metadata am").
		Select("am.id AS id, am.name AS name, COALESCE(systems_affected, 0) AS applicable_systems,"+
			"synopsis,description, public_date, advisory_type_id, advisory_type_id as type").
		Joins("JOIN advisory_account_data aad ON am.id = aad.advisory_id").
		Joins("JOIN rh_account ra ON aad.rh_account_id = ra.id").
		Where("ra.name = ?", account)
	return query
}

func buildAdvisoriesData(advisories *[]AdvisoryWithApplicableSystems) *[]AdvisoryItem {
	data := make([]AdvisoryItem, len(*advisories))
	for i := 0; i < len(*advisories); i++ {
		advisory := (*advisories)[i]
		data[i] = AdvisoryItem{
			Attributes: AdvisoryItemAttributes{
				SystemAdvisoryItemAttributes: SystemAdvisoryItemAttributes{
					Description:  advisory.Description,
					PublicDate:   advisory.PublicDate,
					Synopsis:     advisory.Synopsis,
					AdvisoryType: advisory.AdvisoryTypeID,
					Severity:     advisory.Severity},
				ApplicableSystems: advisory.ApplicableSystems},
			ID:   advisory.Name,
			Type: "advisory",
		}
	}
	return &data
}
