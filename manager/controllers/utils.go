package controllers

import (
	"app/base"
	"app/base/core"
	"app/base/database"
	"app/base/utils"
	"app/manager/config"
	"app/manager/middlewares"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/gocarina/gocsv"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const InvalidOffsetMsg = "Invalid offset"
const InvalidFilter = "Invalid filter field: %v"
const InvalidTagMsg = "Invalid tag '%s'. Use 'namespace/key=val format'"
const InvalidNestedFilter = "Nested operators not yet implemented for standard filters"
const FilterNotSupportedMsg = "filtering not supported on this endpoint"

var tagRegex = regexp.MustCompile(`([^/=]+)/([^/=]+)(=([^/=]+))?`)

func LogAndRespError(c *gin.Context, err error, respMsg string) {
	utils.LogError("err", err.Error(), respMsg)
	c.AbortWithStatusJSON(http.StatusInternalServerError, utils.ErrorResponse{Error: respMsg})
}

func LogWarnAndResp(c *gin.Context, code int, respMsg string) {
	utils.LogWarn(respMsg)
	c.AbortWithStatusJSON(code, utils.ErrorResponse{Error: respMsg})
}

func LogAndRespStatusError(c *gin.Context, code int, err error, msg string) {
	utils.LogError("err", err.Error(), msg)
	c.AbortWithStatusJSON(code, utils.ErrorResponse{Error: msg})
}

func LogAndRespBadRequest(c *gin.Context, err error, respMsg string) {
	utils.LogWarn("err", err.Error(), respMsg)
	c.AbortWithStatusJSON(http.StatusBadRequest, utils.ErrorResponse{Error: respMsg})
}

func LogAndRespNotFound(c *gin.Context, err error, respMsg string) {
	utils.LogWarn("err", err.Error(), respMsg)
	c.AbortWithStatusJSON(http.StatusNotFound, utils.ErrorResponse{Error: respMsg})
}

func ApplySort(c *gin.Context, tx *gorm.DB, fieldExprs database.AttrMap,
	defaultSort, stableSort string) (*gorm.DB, []string, error) {
	query := c.DefaultQuery("sort", defaultSort)
	fields := strings.Split(query, ",")
	appliedFields := make([]string, 0, len(fields))
	allowedFieldSet := map[string]bool{
		"id": true,
	}

	for f := range fieldExprs {
		allowedFieldSet[f] = true
	}

	// We sort by a column expression and not the column name. The column expression is retrieved from fieldExprs
	for _, enteredField := range fields {
		origEnteredField := enteredField // needed for showing correct info in `meta` section
		ascDesc := "ASC"
		if strings.HasPrefix(enteredField, "-") {
			ascDesc = "DESC"
			enteredField = enteredField[1:]
		}
		if !allowedFieldSet[enteredField] {
			return nil, nil, errors.Errorf("Invalid sort field: %v", enteredField)
		}
		column := clause.OrderByColumn{
			Column: clause.Column{Name: fmt.Sprintf("%s %s NULLS LAST", fieldExprs[enteredField].OrderQuery, ascDesc),
				Raw: true},
		}

		tx = tx.Order(column)
		appliedFields = append(appliedFields, origEnteredField)
	}
	tx.Order(stableSort + " ASC")
	return tx, appliedFields, nil
}

type NestedFilterMap map[string]string

var nestedFilters = NestedFilterMap{
	"group_name":                                  "group_name",
	"group_name][in":                              "group_name", // obsoleted, backward compatible
	"system_profile][sap_system":                  "system_profile][sap_system",
	"system_profile][sap_sids":                    "system_profile][sap_sids",
	"system_profile][sap_sids][":                  "system_profile][sap_sids", // obsoleted, backward compatible
	"system_profile][sap_sids][in":                "system_profile][sap_sids", // obsoleted, backward compatible
	"system_profile][sap_sids][in][":              "system_profile][sap_sids", // obsoleted, backward compatible
	"system_profile][ansible":                     "system_profile][ansible",
	"system_profile][ansible][controller_version": "system_profile][ansible][controller_version",
	"system_profile][mssql":                       "system_profile][mssql",
	"system_profile][mssql][version":              "system_profile][mssql][version",
}

func ParseFilters(c *gin.Context, filters Filters, allowedFields database.AttrMap,
	defaultFilters map[string]FilterData) error {
	params := c.Request.URL.Query() // map[string][]string
	for name, values := range params {
		if strings.HasPrefix(name, "filter[") {
			subject := name[7 : len(name)-1] // strip key from "filter[...]"
			for _, v := range values {
				if _, ok := nestedFilters[subject]; ok {
					nested := nestedFilters[subject]
					filters.Update(InventoryFilter, nested, v)
					continue
				}
				if _, ok := allowedFields[subject]; !ok {
					return errors.Errorf(InvalidFilter, subject)
				}

				filters.Update(ColumnFilter, subject, v)
			}
		}
	}

	// Apply default filters if there isn't such filter already
	for n, v := range defaultFilters {
		if _, ok := filters[n]; !ok {
			filters[n] = v
		}
	}

	return nil
}

type ListOpts struct {
	Fields         database.AttrMap
	DefaultFilters map[string]FilterData
	DefaultSort    string
	StableSort     string
	SearchFields   []string
}

func ExportListCommon(tx *gorm.DB, c *gin.Context, opts ListOpts) (*gorm.DB, error) {
	filters := Filters{}
	err := ParseFilters(c, filters, opts.Fields, opts.DefaultFilters)
	if err != nil {
		LogAndRespBadRequest(c, err, err.Error())
		return nil, errors.Wrap(err, "filters parsing failed")
	}
	tx, _ = ApplySearch(c, tx, opts.SearchFields...)

	tx, err = filters.Apply(tx, opts.Fields)
	if err != nil {
		LogAndRespBadRequest(c, err, "Failed to apply filters")
		return nil, errors.Wrap(err, "filters applying failed")
	}
	return tx, nil
}

func ListCommonNoLimitOffset(tx *gorm.DB, c *gin.Context, filters Filters, opts ListOpts, params ...string) (
	*gorm.DB, *ListMeta, []string, error) {
	hasSystems := true
	limit, offset, err := utils.LoadLimitOffset(c, core.DefaultLimit)
	if err != nil {
		LogAndRespBadRequest(c, err, err.Error())
		return nil, nil, nil, errors.Wrap(err, "unable to parse limit, offset params")
	}
	tx, searchQ := ApplySearch(c, tx, opts.SearchFields...)

	tx, err = filters.Apply(tx, opts.Fields)
	if err != nil {
		LogAndRespBadRequest(c, err, err.Error())
		return nil, nil, nil, errors.Wrap(err, "filters applying failed")
	}

	tx, sortFields, err := ApplySort(c, tx, opts.Fields, opts.DefaultSort, opts.StableSort)
	if err != nil {
		LogAndRespBadRequest(c, err, err.Error())
		return nil, nil, nil, errors.Wrap(err, "invalid sort")
	}
	var sortQ string
	if len(sortFields) > 0 {
		sortQ = fmt.Sprintf("sort=%v", strings.Join(sortFields, ","))
	}

	meta := ListMeta{
		Limit:  limit,
		Offset: offset,
		Filter: filters,
		Sort:   sortFields,
		Search: base.RemoveInvalidChars(c.Query("search")),
		// TotalItems: will be updated later in UpdateMetaLinks
		// SubTotals:  will be updated later in UpdateMetaLinks
		HasSystems: &hasSystems,
	}

	params = append(params, filters.ToQueryParams(), sortQ, searchQ)
	return tx, &meta, params, nil
}

func ListCommon(tx *gorm.DB, c *gin.Context, filters Filters, opts ListOpts, params ...string) (
	*gorm.DB, *ListMeta, []string, error) {
	tx, meta, params, err := ListCommonNoLimitOffset(tx, c, filters, opts, params...)
	if err != nil {
		// error handled in ListCommonNoLimitOffset
		return nil, nil, nil, err
	}
	tx = ApplyLimitOffset(tx, meta)
	return tx, meta, params, nil
}

func ApplyLimitOffset(tx *gorm.DB, meta *ListMeta) *gorm.DB {
	if meta.Limit != -1 {
		tx = tx.Limit(meta.Limit)
	}
	tx = tx.Offset(meta.Offset)
	return tx
}

func UpdateMetaLinks(c *gin.Context, meta *ListMeta, total int, subTotals map[string]int, params ...string) (
	*ListMeta, *Links, error) {
	if meta.Offset > total {
		err := errors.New("Offset")
		LogAndRespBadRequest(c, err, InvalidOffsetMsg)
		return nil, nil, err
	}
	path := c.Request.URL.Path
	links := CreateLinks(path, meta.Offset, meta.Limit, total, params...)
	meta.TotalItems = total
	meta.SubTotals = subTotals
	if total == 0 {
		var hasSystems bool
		account := c.GetInt(utils.KeyAccount)
		db := middlewares.DBFromContext(c)
		db.Raw("SELECT EXISTS (SELECT 1 FROM system_platform where rh_account_id = ?)", account).Scan(&hasSystems)
		meta.HasSystems = &hasSystems
	}
	return meta, &links, nil
}

func ApplySearch(c *gin.Context, tx *gorm.DB, searchColumns ...string) (*gorm.DB, string) {
	search := base.RemoveInvalidChars(c.Query("search"))
	if search == "" {
		return tx, ""
	}

	if len(searchColumns) == 0 {
		return tx, ""
	}

	searchExtended := "%" + search + "%"
	concatValue := strings.Join(searchColumns, ",' ',")
	txWithSearch := tx.Where("LOWER(CONCAT("+concatValue+")) LIKE LOWER(?)", searchExtended)
	return txWithSearch, fmt.Sprintf("search=%s", search)
}

type Tag struct {
	Namespace *string `json:"namespace,omitempty"`
	Key       string  `json:"key"`
	Value     *string `json:"value,omitempty"`
}

func HasInventoryFilter(filters Filters) bool {
	if !config.EnableCyndiTags {
		return false
	}
	for _, data := range filters {
		switch data.Type {
		case InventoryFilter:
			return true
		case TagFilter:
			return true
		}
	}
	return false
}

func trimQuotes(s string) string {
	if len(s) >= 2 && s[0] == s[len(s)-1] && (s[0] == '"' || s[0] == '\'') {
		return s[1 : len(s)-1]
	}
	return s
}

func ParseTag(tag string) (*Tag, error) {
	// trim leading and trailing quote, otherwise we can end up with
	// e.g. namespace='"insights-client", key="key", val="val'"
	// when query is tags='insights-client/key=val' which is invalid
	trimmed := trimQuotes(tag)
	matches := tagRegex.FindStringSubmatch(trimmed)
	if len(matches) < 5 {
		// We received an invalid tag
		err := errors.Errorf(InvalidTagMsg, tag)
		return nil, err
	}
	var res Tag
	// Inventory performs similar check
	if strings.ToLower(matches[1]) == "null" {
		res.Namespace = nil
	} else {
		res.Namespace = &matches[1]
	}
	res.Key = matches[2]

	if matches[4] == "" {
		res.Value = nil
	} else {
		res.Value = &matches[4]
	}
	return &res, nil
}

func (t *Tag) ApplyTag(tx *gorm.DB) *gorm.DB {
	if t == nil {
		return tx
	}

	tagStr, _ := sonic.Marshal([]Tag{*t})
	return tx.Where("ih.tags @> ?::jsonb", tagStr)
}

func ParseAllFilters(c *gin.Context, opts ListOpts) (Filters, error) {
	filters := Filters{}

	err := parseTags(c, filters)
	if err != nil {
		return nil, err
	}

	err = ParseFilters(c, filters, opts.Fields, opts.DefaultFilters)
	if err != nil {
		err = errors.Wrap(err, "cannot parse inventory filters")
		LogAndRespBadRequest(c, err, err.Error())
		return nil, err
	}

	return filters, nil
}

func parseTags(c *gin.Context, filters Filters) error {
	tags := c.QueryArray("tags")
	for _, t := range tags {
		tag, err := ParseTag(t)
		if err != nil {
			LogAndRespBadRequest(c, err, err.Error())
			return err
		}

		key := tag.Key
		if tag.Namespace != nil {
			key = *tag.Namespace + "/" + tag.Key
		}

		var value []string
		if value = []string{}; tag.Value != nil {
			var val string
			val, err := strconv.Unquote(*tag.Value)
			if err != nil {
				val = *tag.Value
			}
			value = strings.Split(val, ",")
		}
		filters[key] = FilterData{
			Type:     TagFilter,
			Operator: "eq",
			Values:   value,
		}
	}

	return nil
}

// Filter systems by tags with subquery
func ApplyInventoryFilter(filters map[string]FilterData, tx *gorm.DB, systemIDExpr string) (*gorm.DB, bool) {
	if !config.EnableCyndiTags {
		return tx, false
	}

	subq := database.DB.
		Table("inventory.hosts ih").
		Select("ih.id")

	subq, applied := ApplyInventoryWhere(filters, subq)

	// Don't add the subquery if we don't have to
	if !applied {
		return tx, false
	}
	return tx.Where(fmt.Sprintf("%s::uuid in (?)", systemIDExpr), subq), true
}

// Apply Where clause with tags filter
func ApplyInventoryWhere(filters map[string]FilterData, tx *gorm.DB) (*gorm.DB, bool) {
	applied := false
	for key, val := range filters {
		if val.Type == TagFilter {
			tagString := key + "=" + strings.Join(val.Values, ",")
			tag, _ := ParseTag(tagString)
			tx = tag.ApplyTag(tx)
			applied = true
			continue
		}

		if val.Type == InventoryFilter {
			tx = buildInventoryQuery(tx, key, val.Values)
			applied = true
			continue
		}
	}
	return tx, applied
}

// Builds inventory sub query in generic way.
// Example:
// buildSystemProfileQuery("mssql->version", "1.0")
// returns "(ih.system_profile -> 'mssql' ->> 'version')::text = 1.0"
func buildInventoryQuery(tx *gorm.DB, key string, values []string) *gorm.DB {
	if strings.Contains(key, "group_name") {
		groups := []string{}
		for _, v := range values {
			name := v
			group, err := utils.ParseInventoryGroup(nil, &name)
			if err != nil {
				// couldn't marshal inventory group to json
				continue
			}
			groups = append(groups, group)
		}
		jsonq := fmt.Sprintf("{%s}", strings.Join(groups, ","))
		return tx.Where("ih.groups @> ANY (?::jsonb[])", jsonq)
	}

	var cmp string
	val := values[0]

	switch {
	case val == "not_nil":
		cmp = " is not null"
	case strings.Contains(key, "[sap_sids"):
		cmp = "::jsonb @> ?::jsonb"
		bval, _ := sonic.Marshal(values)
		val = string(bval)
	default:
		cmp = "::text = ?"
	}

	sbkeys := strings.Split(key, "][")
	subq := fmt.Sprintf("(ih.%s", sbkeys[0])
	nSbkeys := len(sbkeys)
	if nSbkeys > 2 {
		subq = fmt.Sprintf("%s -> '%s'", subq, strings.Join(sbkeys[1:nSbkeys-1], "' -> '"))
	}
	if nSbkeys > 1 {
		subq = fmt.Sprintf("%s ->> '%s')", subq, sbkeys[nSbkeys-1])
	}

	subq = fmt.Sprintf("%s%s", subq, cmp)
	if val == "not_nil" {
		return tx.Where(subq)
	}

	return tx.Where(subq, val)
}

func Csv(ctx *gin.Context, code int, res interface{}) {
	ctx.Status(code)
	ctx.Header("Content-Type", "text/csv")
	gocsv.SetCSVWriter(func(out io.Writer) *gocsv.SafeCSVWriter {
		writer := csv.NewWriter(out)
		writer.UseCRLF = true
		return gocsv.NewSafeCSVWriter(writer)
	})
	err := gocsv.Marshal(res, ctx.Writer)
	if err != nil {
		panic(err)
	}
}

func OutputExportData(c *gin.Context, data interface{}) {
	accept := c.GetHeader("Accept")
	switch {
	case strings.Contains(accept, "application/json"):
		c.JSON(http.StatusOK, data)
	case strings.Contains(accept, "text/csv"):
		Csv(c, http.StatusOK, data)
	default:
		LogWarnAndResp(c, http.StatusUnsupportedMediaType,
			fmt.Sprintf("Invalid content type '%s', use 'application/json' or 'text/csv'", accept))
	}
}

func systemDBLookups2SystemItems(systems []SystemDBLookup) ([]SystemItem, int, map[string]int) {
	data := make([]SystemItem, len(systems))
	var total int
	subtotals := map[string]int{
		"patched":   0,
		"unpatched": 0,
		"stale":     0,
	}
	if len(systems) > 0 {
		total = systems[0].Total
		subtotals["patched"] = systems[0].TotalPatched
		subtotals["unpatched"] = systems[0].TotalUnpatched
		subtotals["stale"] = systems[0].TotalStale
	}

	for i := range systems {
		data[i] = SystemItem{
			Attributes: systems[i].SystemItemAttributes,
			ID:         systems[i].ID,
			Type:       "system",
		}
	}
	return data, total, subtotals
}

func advisoriesIDs(advisories []AdvisoryID) IDsPlainResponse {
	var resp IDsPlainResponse
	if advisories == nil {
		return resp
	}
	resp.IDs = make([]string, 0, len(advisories))
	resp.Data = make([]IDPlain, 0, len(advisories))
	for _, x := range advisories {
		resp.IDs = append(resp.IDs, x.ID)
		resp.Data = append(resp.Data, IDPlain(x))
	}
	return resp
}

func advisoriesStatusIDs(advisories []AdvisoryStatusID) IDsStatusResponse {
	resp := IDsStatusResponse{}
	if advisories == nil {
		return resp
	}
	ids := make([]string, len(advisories))
	data := make([]IDStatus, len(advisories))
	for i, x := range advisories {
		ids[i] = x.ID
		data[i] = IDStatus{x.ID, x.Status}
	}
	resp.IDs = ids
	resp.Data = data
	return resp
}

func systemsIDs(c *gin.Context, systems []SystemsID, meta *ListMeta) (IDsPlainResponse, error) {
	var total int
	var resp IDsPlainResponse
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
	resp.IDs = make([]string, 0, len(systems))
	resp.Data = make([]IDPlain, 0, len(systems))
	for _, x := range systems {
		resp.IDs = append(resp.IDs, x.ID)
		resp.Data = append(resp.Data, IDPlain{ID: x.ID})
	}
	return resp, nil
}

func systemsSatelliteIDs(c *gin.Context, systems []SystemsSatelliteManagedID, meta *ListMeta,
) (IDsSatelliteManagedResponse, error) {
	var total int
	resp := IDsSatelliteManagedResponse{}
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
	data := make([]IDSatelliteManaged, len(systems))
	for i, x := range systems {
		ids[i] = x.ID
		data[i] = IDSatelliteManaged{x.ID, x.SatelliteManaged}
	}
	resp.IDs = ids
	resp.Data = data
	return resp, nil
}

func fillAdvisoryItemAttributeReleaseVersion(advisory AdvisoryItemAttributesCommon) AdvisoryItemAttributesCommon {
	// parse release version from json to []strings
	var err error
	advisory.ReleaseVersions, err = parseJSONList(advisory.ReleaseVersionsJSONB)
	if err != nil {
		utils.LogWarn("err", err.Error(), "json", advisory.ReleaseVersionsJSONB, "Unable to parse json list")
	}
	return advisory
}

func parseJSONList(jsonb []byte) ([]string, error) {
	if jsonb == nil {
		return []string{}, nil
	}

	var items []string
	err := sonic.Unmarshal(jsonb, &items)
	if err != nil {
		return nil, err
	}
	return items, nil
}

func isFilterInURLValid(c *gin.Context) bool {
	if strings.Contains(c.Request.URL.String(), "filter") {
		LogAndRespBadRequest(c, errors.New(FilterNotSupportedMsg), FilterNotSupportedMsg)
		return false
	}
	return true
}
