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
const InvalidTagMsg = "Invalid tag '%s'. Use 'namespace/key=val format'"
const InvalidNestedFilter = "Nested operators not yet implemented for standard filters"
const FilterNotSupportedMsg = "filtering not supported on this endpoint"

var tagRegex = regexp.MustCompile(`([^/=]+)/([^/=]+)(=([^/=]+))?`)
var enableCyndiTags = utils.GetBoolEnvOrDefault("ENABLE_CYNDI_TAGS", false)
var disableCachedCounts = utils.GetBoolEnvOrDefault("DISABLE_CACHE_COUNTS", false)

var validFilters = map[string]bool{
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

func validateFilters(q QueryMap, allowedFields database.AttrMap) error {
	for key := range q {
		// system_profile is hadled by tags, so it can be skipped
		if key == "system_profile" {
			continue
		}
		if _, ok := allowedFields[key]; !ok {
			return errors.Errorf("Invalid filter field: %v", key)
		}
	}
	return nil
}

func ParseFilters(q QueryMap, allowedFields database.AttrMap,
	defaultFilters map[string]FilterData, apiver int) (Filters, error) {
	filters := Filters{}
	var err error

	if err = validateFilters(q, allowedFields); err != nil {
		return filters, err
	}

	// Apply default filters
	for n, v := range defaultFilters {
		filters[n] = v
	}

	// nolint: scopelint
	for f := range allowedFields {
		if elem := q.Path(f); elem != nil {
			elem.Visit(func(path []string, val string) {
				// If we encountered error in previous element, skip processing others
				if err != nil {
					return
				}

				// the filter[a][eq]=b syntax was not yet implemented
				if len(path) > 0 {
					err = errors.New(InvalidNestedFilter)
					return
				}
				filters[f], err = ParseFilterValue(val)
			})
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

	return filters, err
}

type ListOpts struct {
	Fields         database.AttrMap
	DefaultFilters map[string]FilterData
	DefaultSort    string
	StableSort     string
	SearchFields   []string
}

func ExportListCommon(tx *gorm.DB, c *gin.Context, opts ListOpts) (*gorm.DB, error) {
	query := NestedQueryMap(c, "filter")
	apiver := c.GetInt(middlewares.KeyApiver)
	filters, err := ParseFilters(query, opts.Fields, opts.DefaultFilters, apiver)
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
func ListCommon(tx *gorm.DB, c *gin.Context, tagFilter map[string]FilterData, opts ListOpts, params ...string) (
	*gorm.DB, *ListMeta, []string, error) {
	hasSystems := true
	apiver := c.GetInt(middlewares.KeyApiver)
	limit, offset, err := utils.LoadLimitOffset(c, core.DefaultLimit)
	if err != nil {
		LogAndRespBadRequest(c, err, err.Error())
		return nil, nil, nil, errors.Wrap(err, "unable to parse limit, offset params")
	}
	tx, searchQ := ApplySearch(c, tx, opts.SearchFields...)

	query := NestedQueryMap(c, "filter")

	filters, err := ParseFilters(query, opts.Fields, opts.DefaultFilters, apiver)
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
	Namespace *string
	Key       string
	Value     *string
}

func HasTags(c *gin.Context) bool {
	if !enableCyndiTags {
		return false
	}
	hasTags := false
	if len(c.QueryArray("tags")) > 0 {
		hasTags = true
	}

	// If we have the `system_profile` filter item, then we have tags
	spQuery := NestedQueryMap(c, "filter").Path("system_profile")
	if spQuery != nil {
		spQuery.Visit(func(path []string, val string) {
			hasTags = true
		})
	}
	return hasTags
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

	ns := ""
	if t.Namespace != nil {
		ns = fmt.Sprintf(`"namespace": "%s",`, *t.Namespace)
	}

	v := ""
	if t.Value != nil {
		v = fmt.Sprintf(`, "value":"%s"`, *t.Value)
	}

	query := fmt.Sprintf(`[{%s "key": "%s" %s}]`, ns, t.Key, v)
	return tx.Where("ih.tags @> ?::jsonb", query)
}

func ParseTagsFilters(c *gin.Context) (map[string]FilterData, error) {
	filters := Filters{}

	err := parseTagsFromCtx(c, filters)
	if err != nil {
		return nil, err
	}

	parseFiltersFromCtx(c, filters)

	return filters, nil
}

func parseTagsFromCtx(c *gin.Context, filters Filters) error {
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
			Operator: "eq",
			Values:   value,
		}
	}

	return nil
}

func parseFiltersFromCtx(c *gin.Context, filters Filters) {
	filter := NestedQueryMap(c, "filter").Path("system_profile")
	if filter == nil {
		return
	}

	filter.Visit(func(path []string, val string) {
		// Specific filter keys
		if len(path) >= 1 && path[0] == "sap_sids" {
			val = strconv.Quote(val)
			var op string
			if op = "eq"; len(path) > 1 {
				op = path[1]
			}
			appendFilterData(filters, "sap_sids", op, val)

			return
		}

		// Generic filter keys
		var key string
		// Builds key in following format path[0]->path[1]->path[2]...
		for i, s := range path {
			if i == 0 {
				key = path[0]
				continue
			}
			key = fmt.Sprintf("%s->%s", key, s)
		}
		appendFilterData(filters, key, "eq", val)
	})
}

// Filter systems by tags with subquery
func ApplyTagsFilter(filters map[string]FilterData, tx *gorm.DB, systemIDExpr string) (*gorm.DB, bool) {
	if !enableCyndiTags {
		return tx, false
	}

	subq := database.Db.
		Table("inventory.hosts ih").
		Select("ih.id")

	subq, applied := ApplyTagsWhere(filters, subq)

	// Don't add the subquery if we don't have to
	if !applied {
		return tx, false
	}
	return tx.Where(fmt.Sprintf("%s::uuid in (?)", systemIDExpr), subq), true
}

// Apply Where clause with tags filter
func ApplyTagsWhere(filters map[string]FilterData, tx *gorm.DB) (*gorm.DB, bool) {
	applied := false
	for key, val := range filters {
		if strings.Contains(key, "/") {
			tagString := key + "=" + strings.Join(val.Values, ",")
			tag, _ := ParseTag(tagString)
			tx = tag.ApplyTag(tx)
			applied = true
			continue
		}

		if validFilters[key] {
			values := strings.Join(val.Values, ",")
			q := buildQuery(key, values)

			// Builds array of values
			if len(val.Values) > 1 {
				values = fmt.Sprintf("[%s]", values)
			}

			if values == "not_nil" {
				tx = tx.Where(q)
			} else {
				tx = tx.Where(q, values)
			}

			applied = true
		}
	}
	return tx, applied
}

// Builds system_profile sub query in generic way.
// Example:
// buildQuery("mssql->version", "1.0")
// returns "(ih.system_profile -> 'mssql' ->> 'version')::text = 1.0"
func buildQuery(key string, val string) string {
	var cmp string

	switch key {
	case "sap_sids":
		cmp = "::jsonb @> ?::jsonb"
	default:
		cmp = "::text = ?"
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

	return fmt.Sprintf("%s%s", subq, cmp)
}

type QueryItem interface {
	IsQuery()
	Visit(visitor func(path []string, val string), pathPrefix ...string)
}
type QueryArr []string

func (QueryArr) IsQuery() {}

func (q QueryArr) Visit(visitor func(path []string, val string), pathPrefix ...string) {
	for _, item := range q {
		visitor(pathPrefix, item)
	}
}

type QueryMap map[string]QueryItem

func (QueryMap) IsQuery() {}

func (q QueryMap) Visit(visitor func(path []string, val string), pathPrefix ...string) {
	for k, v := range q {
		newPath := make([]string, len(pathPrefix))
		copy(newPath, pathPrefix)
		newPath = append(newPath, k)
		switch val := v.(type) {
		case QueryMap:
			val.Visit(visitor, newPath...)
		case QueryArr:
			// Inlined code here to avoid calling interface method in struct receiver (low perf)
			for _, item := range val {
				visitor(newPath, item)
			}
		}
	}
}

func (q QueryMap) GetPath(keys ...string) (QueryItem, bool) {
	var item QueryItem
	item, has := q, true

	for has && item != nil && len(keys) > 0 {
		switch itemMap := item.(type) {
		case QueryMap:
			item, has = itemMap[keys[0]]
			keys = keys[1:]

		default:
			break
		}
	}
	return item, has
}

func (q QueryMap) Path(keys ...string) QueryItem {
	v, _ := q.GetPath(keys...)
	return v
}

func NestedQueryMap(c *gin.Context, key string) QueryMap {
	return nestedQueryImpl(c.Request.URL.Query(), key)
}

func (q *QueryMap) appendValue(steps []string, value []string) {
	res := *q
	for i, v := range steps {
		if i == len(steps)-1 {
			res[v] = QueryArr(value)
		} else {
			if _, has := res[v]; !has {
				res[v] = QueryMap{}
			}
			res = res[v].(QueryMap)
		}
	}
}

// nolint: gocognit
func nestedQueryImpl(values map[string][]string, key string) QueryMap {
	root := QueryMap{}

	for name, value := range values {
		var steps []string
		var i int
		var j int
		for len(name) > 0 && i >= 0 && j >= 0 {
			if i = strings.IndexByte(name, '['); i >= 0 {
				if name[0:i] == key || len(steps) > 0 {
					// if j is 0 here, that means we received []as a part of query param name, should indicate an array
					if j = strings.IndexByte(name[i+1:], ']'); j >= 0 {
						// Skip [] in param names
						if len(name[i+1:][:j]) > 0 {
							steps = append(steps, name[i+1:][:j])
						}
						name = name[i+j+2:]
					}
				} else if name[0:i] != key && steps == nil {
					// Invalid key for the context - abort.
					return root
				}
			}
		}
		root.appendValue(steps, value)
	}
	return root
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
	var err error
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
		system.Tags, err = parseSystemTags(system.TagsStr)
		if err != nil {
			utils.LogDebug("err", err.Error(), "inventory_id", system.ID, "system tags parsing failed")
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

// Parse tags from TagsStr string attribute to Tags SystemTag array attribute.
// It's used in /*systems endpoints as we can not map this attribute directly from database query result.
func (s *SystemDBLookupSlice) ParseAndFillTags() {
	var err error
	for i, system := range *s {
		(*s)[i].Tags, err = parseSystemTags(system.TagsStr)
		if err != nil {
			utils.LogDebug("err", err.Error(), "inventory_id", system.ID, "system tags to export parsing failed")
		}
	}
}

// Parse tags from TagsStr string attribute to Tags SystemTag array attribute.
// It's used in /*systems endpoints as we can not map this attribute directly from database query result.
func (s *AdvisorySystemDBLookupSlice) ParseAndFillTags() {
	var err error
	for i, system := range *s {
		(*s)[i].Tags, err = parseSystemTags(system.TagsStr)
		if err != nil {
			utils.LogDebug("err", err.Error(), "inventory_id", system.ID, "system tags to export parsing failed")
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

func appendFilterData(filters Filters, key string, op, val string) {
	if fd, ok := filters[key]; ok {
		fd.Values = append(fd.Values, val)
		filters[key] = fd
	} else {
		filters[key] = FilterData{
			Operator: op,
			Values:   strings.Split(val, ","),
		}
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
