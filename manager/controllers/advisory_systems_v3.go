package controllers

import (
	"app/base/database"
	"app/base/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

var AdvisorySystemsFields = database.MustGetQueryAttrs(&AdvisorySystemDBLookup{})
var AdvisorySystemsSelect = database.MustGetSelect(&AdvisorySystemDBLookup{})

type AdvisorySystemsResponseV3 struct {
	Data  []AdvisorySystemItem `json:"data"`
	Links Links                `json:"links"`
	Meta  ListMeta             `json:"meta"`
}

type AdvisorySystemItem struct {
	Attributes AdvisorySystemItemAttributes `json:"attributes"`
	ID         string                       `json:"id"`
	Type       string                       `json:"type"`
}

// nolint: lll
type AdvisorySystemDBLookup struct {
	SystemsMetaTagTotal
	AdvisorySystemItemAttributes
	SystemIDAttribute
}

// nolint: lll
type AdvisorySystemItemAttributes struct {
	SystemDisplayName
	SystemLastUpload
	SystemStale
	OSAttributes
	SystemTimestamps
	SystemTags
	BaselineNameAttr
	SystemAdvisoryStatus
}

var AdvisorySystemOptsV3 = ListOpts{
	Fields: AdvisorySystemsFields,
	// By default, we show only fresh systems. If all systems are required, you must pass in:true,false filter into the api
	DefaultFilters: map[string]FilterData{
		"stale": {
			Operator: "eq",
			Values:   []string{"false"},
		},
	},
	DefaultSort:  "-last_upload",
	StableSort:   "sp.id",
	SearchFields: []string{"sp.display_name"},
}

func advisorySystemsListHandlerV3(c *gin.Context) {
	query, meta, params, err := advisorySystemsCommon(c)
	if err != nil {
		return
	} // Error handled in method itself

	var dbItems []AdvisorySystemDBLookup

	if err = query.Scan(&dbItems).Error; err != nil {
		LogAndRespError(c, err, "database error")
		return
	}

	data, total := buildAdvisorySystemsData(dbItems)

	meta, links, err := UpdateMetaLinks(c, meta, total, nil, params...)
	if err != nil {
		return // Error handled in method itself
	}
	var resp = AdvisorySystemsResponseV3{
		Data:  data,
		Links: *links,
		Meta:  *meta,
	}
	c.JSON(http.StatusOK, &resp)
}

func buildAdvisorySystemsData(fields []AdvisorySystemDBLookup) ([]AdvisorySystemItem, int) {
	var total int
	var err error
	if len(fields) > 0 {
		total = fields[0].Total
	}
	data := make([]AdvisorySystemItem, len(fields))
	for i, as := range fields {
		as.AdvisorySystemItemAttributes.Tags, err = parseSystemTags(as.TagsStr)
		if err != nil {
			utils.LogDebug("err", err.Error(), "inventory_id", as.ID, "system tags parsing failed")
		}
		data[i] = AdvisorySystemItem{
			Attributes: as.AdvisorySystemItemAttributes,
			ID:         as.ID,
			Type:       "system",
		}
	}
	return data, total
}
