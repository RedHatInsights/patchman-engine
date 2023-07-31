package controllers

import (
	"app/base"
	"app/base/core"
	"app/base/database"
	"app/base/utils"
	"app/manager/middlewares"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gocarina/gocsv"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const InvalidOffsetMsg = "Invalid offset"
const InvalidFilter = "Invalid filter field: %v"
const InvalidTagMsg = "Invalid tag '%s'. Use 'namespace/key=val format'"
const InvalidNestedFilter = "Nested operators not yet implemented for standard filters"
const FilterNotSupportedMsg = "filtering not supported on this endpoint"

var tagRegex = regexp.MustCompile(`([^/=]+)/([^/=]+)(=([^/=]+))?`)
var enableCyndiTags = utils.GetBoolEnvOrDefault("ENABLE_CYNDI_TAGS", false)
var disableCachedCounts = utils.GetBoolEnvOrDefault("DISABLE_CACHE_COUNTS", false)

var validSystemProfileFilters = map[string]bool{
	"sap_sids":                    true,
	"sap_system":                  true,
	"mssql":                       true,
	"mssql->version":              true,
	"ansible":                     true,
	"ansible->controller_version": true,
}

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

// nolint: prealloc
func ApplySort(c *gin.Context, tx *gorm.DB, fieldExprs database.AttrMap,
	defaultSort, stableSort string) (*gorm.DB, []string, error) {
	apiver := c.GetInt(middlewares.KeyApiver)
	query := c.DefaultQuery("sort", defaultSort)
	fields := strings.Split(query, ",")
	var appliedFields []string
	allowedFieldSet := map[string]bool{
		"id": true,
	}

	for f := range fieldExprs {
		allowedFieldSet[f] = true
	}

	// We sort by a column expression and not the column name. The column expression is retrieved from fieldExprs
	for _, enteredField := range fields {
		origEnteredField := enteredField // needed for showing correct info in `meta` section
		if apiver < 3 && strings.Contains(enteredField, "applicable_systems") {
			// `applicable_systems` is used in v1 and v2 API for consistency, in fact it is installable_systems
			// therefore, if user wants to sort by `applicable_systems`, we need to sort `installable_systems` in DB
			enteredField = strings.Replace(enteredField, "applicable", "installable", 1)
		}
		if strings.HasPrefix(enteredField, "-") && allowedFieldSet[enteredField[1:]] { //nolint:gocritic
			tx = tx.Order(fmt.Sprintf("%s DESC NULLS LAST", fieldExprs[enteredField[1:]].OrderQuery))
		} else if allowedFieldSet[enteredField] {
			tx = tx.Order(fmt.Sprintf("%s ASC NULLS LAST", fieldExprs[enteredField].OrderQuery))
		} else {
			// We have not found any matches in allowed fields, return an error
			return nil, nil, errors.Errorf("Invalid sort field: %v", enteredField)
		}
		appliedFields = append(appliedFields, origEnteredField)
	}
	tx.Order(stableSort + " ASC")
	return tx, appliedFields, nil
}

type NestedFilterMap map[string]string

var nestedFilters = NestedFilterMap{
	"group_name":                                  "group_name",
	"group_name][in":                              "group_name", // obsoleted, backward compatible
	"system_profile][sap_system":                  "sap_system",
	"system_profile][sap_sids":                    "sap_sids",
	"system_profile][sap_sids][":                  "sap_sids", // obsoleted, backward compatible
	"system_profile][sap_sids][in":                "sap_sids", // obsoleted, backward compatible
	"system_profile][sap_sids][in][":              "sap_sids", // obsoleted, backward compatible
	"system_profile][ansible":                     "ansible",
	"system_profile][ansible][controller_version": "ansible->controller_version",
	"system_profile][mssql":                       "mssql",
	"system_profile][mssql][version":              "mssql->version",
}

func ParseFilters(c *gin.Context, filters Filters, allowedFields database.AttrMap,
	defaultFilters map[string]FilterData, apiver int) error {
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

	// backward compatibility for v2 api and applicable_systems filter
	if apiver < 3 {
		if _, ok := filters["applicable_systems"]; ok {
			// replace with `installable_systems`
			filters["installable_systems"] = filters["applicable_systems"]
			delete(filters, "applicable_systems")
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
	apiver := c.GetInt(middlewares.KeyApiver)
	filters := Filters{}
	err := ParseFilters(c, filters, opts.Fields, opts.DefaultFilters, apiver)
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

func extractTagsQueryString(c *gin.Context) string {
	var tagQ string
	var tags = c.QueryArray("tags")
	for _, t := range tags {
		tagQ = fmt.Sprintf("tags=%s&%s", t, tagQ)
	}
	return strings.TrimSuffix(tagQ, "&")
}

// nolint: funlen, lll
func ListCommon(tx *gorm.DB, c *gin.Context, filters Filters, tagFilter Filters, opts ListOpts, params ...string) (
	*gorm.DB, *ListMeta, []string, error) {
	hasSystems := true
	limit, offset, err := utils.LoadLimitOffset(c, core.DefaultLimit)
	if err != nil {
		LogAndRespBadRequest(c, err, err.Error())
		return nil, nil, nil, errors.Wrap(err, "unable to parse limit, offset params")
	}
	tx, searchQ := ApplySearch(c, tx, opts.SearchFields...)

	if err != nil {
		LogAndRespBadRequest(c, err, err.Error())
		return nil, nil, nil, errors.Wrap(err, "filters parsing failed")
	}

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

	tagQ := extractTagsQueryString(c)

	params = append(params, filters.ToQueryParams(), sortQ, tagQ, searchQ)
	mergeMaps(meta.Filter, tagFilter)

	if limit != -1 {
		tx = tx.Limit(limit)
	}
	tx = tx.Offset(offset)
	return tx, &meta, params, nil
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
		account := c.GetInt(middlewares.KeyAccount)
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
	if !enableCyndiTags {
		return false
	}
	for _, data := range filters {
		if data.Type == InventoryFilter {
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

	tagStr, _ := json.Marshal([]Tag{*t})
	return tx.Where("ih.tags @> ?::jsonb", tagStr)
}

func ParseAllFilters(c *gin.Context, opts ListOpts) (Filters, error) {
	filters := Filters{}

	err := parseTags(c, filters)
	if err != nil {
		return nil, err
	}

	apiver := c.GetInt(middlewares.KeyApiver)
	err = ParseFilters(c, filters, opts.Fields, opts.DefaultFilters, apiver)
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
	if !enableCyndiTags {
		return tx, false
	}

	subq := database.Db.
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
		if strings.Contains(key, "/") {
			tagString := key + "=" + strings.Join(val.Values, ",")
			tag, _ := ParseTag(tagString)
			tx = tag.ApplyTag(tx)
			applied = true
			continue
		}

		if validSystemProfileFilters[key] {
			tx = buildSystemProfileQuery(tx, key, val.Values)

			applied = true
			continue
		}

		if strings.Contains(key, "group_name") {
			groups := []string{}
			for _, v := range val.Values {
				name := v
				group, err := utils.ParseInventoryGroup(nil, &name)
				if err != nil {
					// couldn't marshal inventory group to json
					continue
				}
				groups = append(groups, group)
			}
			jsonq := fmt.Sprintf("{%s}", strings.Join(groups, ","))
			tx = tx.Where("ih.groups @> ANY (?::jsonb[])", jsonq)
			applied = true
		}
	}
	return tx, applied
}

// Builds system_profile sub query in generic way.
// Example:
// buildSystemProfileQuery("mssql->version", "1.0")
// returns "(ih.system_profile -> 'mssql' ->> 'version')::text = 1.0"
func buildSystemProfileQuery(tx *gorm.DB, key string, values []string) *gorm.DB {
	var cmp string
	var val string

	switch key {
	case "sap_sids":
		cmp = "::jsonb @> ?::jsonb"
		bval, _ := json.Marshal(values)
		val = string(bval)
	default:
		cmp = "::text = ?"
		val = values[0]
	}

	if val == "not_nil" {
		cmp = " is not null"
	}

	subq := "(ih.system_profile"
	sbkeys := strings.Split(key, "->")
	for i, sbkey := range sbkeys {
		sbkey = fmt.Sprintf("'%s'", sbkey)
		if i == len(sbkeys)-1 {
			subq = fmt.Sprintf("%s ->> %s)", subq, sbkey)
		} else {
			subq = fmt.Sprintf("%s -> %s", subq, sbkey)
		}
	}

	subq = fmt.Sprintf("%s%s", subq, cmp)
	if val == "not_nil" {
		return tx.Where(subq)
	}

	return tx.Where(subq, val)
}

func Csv(ctx *gin.Context, code int, res interface{}) {
	ctx.Status(http.StatusOK)
	ctx.Header("Content-Type", "text/csv")
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

	for i, system := range systems {
		if err := parseSystemItems(system.TagsStr, &system.Tags); err != nil {
			utils.LogDebug("err", err.Error(), "inventory_id", system.ID, "system tags parsing failed")
		}
		if err := parseSystemItems(system.GroupsStr, &system.Groups); err != nil {
			utils.LogDebug("err", err.Error(), "inventory_id", system.ID, "system groups parsing failed")
		}
		data[i] = SystemItem{
			Attributes: system.SystemItemAttributesAll,
			ID:         system.ID,
			Type:       "system",
		}
	}
	return data, total, subtotals
}

func systemItems2SystemItemsV2(items []SystemItem) []SystemItemV2 {
	res := make([]SystemItemV2, 0, len(items))
	for _, x := range items {
		res = append(res, SystemItemV2{
			Attributes: SystemItemAttributesV2{
				x.Attributes.SystemItemAttributesCommon, x.Attributes.SystemItemAttributesV2Only,
			},
			ID:   x.ID,
			Type: x.Type,
		})
	}
	return res
}

func systemItems2SystemItemsV3(items []SystemItem) []SystemItemV3 {
	res := make([]SystemItemV3, 0, len(items))
	for _, x := range items {
		res = append(res, SystemItemV3{
			Attributes: SystemItemAttributesV3{
				x.Attributes.SystemItemAttributesCommon, x.Attributes.SystemItemAttributesV3Only,
			},
			ID:   x.ID,
			Type: x.Type,
		})
	}
	return res
}

func advisoriesIDs(advisories []AdvisoryID) []string {
	if advisories == nil {
		return []string{}
	}
	ids := make([]string, len(advisories))
	for i, x := range advisories {
		ids[i] = x.ID
	}
	return ids
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

func systemsIDs(c *gin.Context, systems []SystemsID, meta *ListMeta) ([]string, error) {
	var total int
	if len(systems) > 0 {
		total = systems[0].Total
	}
	if meta.Offset > total {
		err := errors.New("Offset")
		LogAndRespBadRequest(c, err, InvalidOffsetMsg)
		return []string{}, err
	}
	if systems == nil {
		return []string{}, nil
	}
	ids := make([]string, len(systems))
	for i, x := range systems {
		ids[i] = x.ID
	}
	return ids, nil
}

type SystemDBLookupSlice []SystemDBLookup
type AdvisorySystemDBLookupSlice []AdvisorySystemDBLookup
type BaselineSystemsDBLookupSlice []BaselineSystemsDBLookup

// Parse tags from TagsStr string attribute to Tags SystemTag array attribute.
// It's used in /*systems endpoints as we can not map this attribute directly from database query result.
func (s *SystemDBLookupSlice) ParseAndFillTags() {
	for i, system := range *s {
		if err := parseSystemItems(system.TagsStr, &(*s)[i].Tags); err != nil {
			utils.LogDebug("err", err.Error(), "inventory_id", system.ID, "system tags to export parsing failed")
		}
		if err := parseSystemItems(system.GroupsStr, &(*s)[i].Groups); err != nil {
			utils.LogDebug("err", err.Error(), "inventory_id", system.ID, "system groups to export parsing failed")
		}
	}
}

// Parse tags from TagsStr string attribute to Tags SystemTag array attribute.
// It's used in /*systems endpoints as we can not map this attribute directly from database query result.
func (s *AdvisorySystemDBLookupSlice) ParseAndFillTags() {
	for i, system := range *s {
		if err := parseSystemItems(system.TagsStr, &(*s)[i].Tags); err != nil {
			utils.LogDebug("err", err.Error(), "inventory_id", system.ID, "system tags to export parsing failed")
		}
		if err := parseSystemItems(system.GroupsStr, &(*s)[i].Groups); err != nil {
			utils.LogDebug("err", err.Error(), "inventory_id", system.ID, "system groups to export parsing failed")
		}
	}
}

// Parse tags from TagsStr string attribute to Tags SystemTag array attribute.
// It's used in /*systems endpoints as we can not map this attribute directly from database query result.
func (s *BaselineSystemsDBLookupSlice) ParseAndFillTags() {
	for i, system := range *s {
		if err := parseSystemItems(system.TagsStr, &(*s)[i].Tags); err != nil {
			utils.LogDebug("err", err.Error(), "inventory_id", system.ID, "system tags to export parsing failed")
		}
		if err := parseSystemItems(system.GroupsStr, &(*s)[i].Groups); err != nil {
			utils.LogDebug("err", err.Error(), "inventory_id", system.ID, "system groups to export parsing failed")
		}
	}
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
	err := json.Unmarshal(jsonb, &items)
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

func mergeMaps(first map[string]FilterData, second map[string]FilterData) {
	for key, val := range second {
		first[key] = val
	}
}

// Pagination query for handlers where ListCommon is not used
func Paginate(tx *gorm.DB, limit *int, offset *int) (int, int, error) {
	var total int64
	if limit == nil {
		limit = &core.DefaultLimit
	}
	if offset == nil {
		offset = &core.DefaultOffset
	}
	if err := utils.CheckLimitOffset(*limit, *offset); err != nil {
		return *limit, *offset, err
	}

	tx.Count(&total)
	if total < int64(*offset) {
		return *limit, *offset, errors.New(InvalidOffsetMsg)
	}
	if *limit != -1 {
		// consistency with ListCommon
		tx.Limit(*limit)
	}
	tx.Offset(*offset)
	return *limit, *offset, nil
}

// Return value for v3 api or return nil for previous versions
func APIV3Compat[T any](x *T, apiver int) *T {
	if apiver < 3 {
		return nil
	}
	return x
}
