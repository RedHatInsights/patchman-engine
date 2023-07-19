package controllers

import (
	"app/base/database"
	"app/base/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
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
	SystemGroups
	BaselineIDAttr
	BaselineNameAttr
	SystemAdvisoryStatus
}

type SystemsStatusID struct {
	SystemsID
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
		if err = parseSystemItems(as.TagsStr, &as.AdvisorySystemItemAttributes.Tags); err != nil {
			utils.LogDebug("err", err.Error(), "inventory_id", as.ID, "system tags parsing failed")
		}
		if err = parseSystemItems(as.GroupsStr, &as.AdvisorySystemItemAttributes.Groups); err != nil {
			utils.LogDebug("err", err.Error(), "inventory_id", as.ID, "system groups parsing failed")
		}
		data[i] = AdvisorySystemItem{
			Attributes: as.AdvisorySystemItemAttributes,
			ID:         as.ID,
			Type:       "system",
		}
	}
	return data, total
}

func systemsIDsStatus(c *gin.Context, systems []SystemsStatusID, meta *ListMeta) (IDsStatusResponse, error) {
	var total int
	resp := IDsStatusResponse{}
	if len(systems) > 0 {
		total = systems[0].Total
	}
	if meta.Offset > total {
		err := errors.New("Offset")
		LogAndRespBadRequest(c, err, InvalidOffsetMsg)
		return resp, err
	}
	if systems == nil {
		return resp, nil
	}
	ids := make([]string, len(systems))
	data := make([]IDStatus, len(systems))
	for i, x := range systems {
		ids[i] = x.ID
		data[i] = IDStatus{x.ID, x.Status}
	}
	resp.IDs = ids
	resp.Data = data
	return resp, nil
}
