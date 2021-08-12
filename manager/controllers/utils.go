package controllers

import (
	"app/base"
	"app/base/core"
	"app/base/database"
	"app/base/utils"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gocarina/gocsv"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const InvalidOffsetMsg = "Invalid offset"
const InvalidTagMsg = "Invalid tag '%s'. Use 'namespace/key=val format'"

var tagRegex = regexp.MustCompile(`([^/=]+)/([^/=]+)(=([^/=]+))?`)
var enableCyndiTags = utils.GetBoolEnvOrDefault("ENABLE_CYNDI_TAGS", false)
var disableCachedCounts = utils.GetBoolEnvOrDefault("DISABLE_CACHE_COUNTS", false)

func LogAndRespError(c *gin.Context, err error, respMsg string) {
	utils.Log("err", err.Error()).Error(respMsg)
	c.AbortWithStatusJSON(http.StatusInternalServerError, utils.ErrorResponse{Error: respMsg})
}

func LogWarnAndResp(c *gin.Context, code int, respMsg string) {
	utils.Log().Warn(respMsg)
	c.AbortWithStatusJSON(code, utils.ErrorResponse{Error: respMsg})
}

func LogAndRespStatusError(c *gin.Context, code int, err error, msg string) {
	utils.Log("err", err.Error()).Error(msg)
	c.AbortWithStatusJSON(code, utils.ErrorResponse{Error: msg})
}

func LogAndRespBadRequest(c *gin.Context, err error, respMsg string) {
	utils.Log("err", err.Error()).Warn(respMsg)
	c.AbortWithStatusJSON(http.StatusBadRequest, utils.ErrorResponse{Error: respMsg})
}

func LogAndRespNotFound(c *gin.Context, err error, respMsg string) {
	utils.Log("err", err.Error()).Warn(respMsg)
	c.AbortWithStatusJSON(http.StatusNotFound, utils.ErrorResponse{Error: respMsg})
}

// nolint: prealloc
func ApplySort(c *gin.Context, tx *gorm.DB, fieldExprs database.AttrMap,
	defaultSort string) (*gorm.DB, []string, error) {
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
		if strings.HasPrefix(enteredField, "-") && allowedFieldSet[enteredField[1:]] { //nolint:gocritic
			tx = tx.Order(fmt.Sprintf("%s DESC NULLS LAST", fieldExprs[enteredField[1:]].Query))
		} else if allowedFieldSet[enteredField] {
			tx = tx.Order(fmt.Sprintf("%s ASC NULLS FIRST", fieldExprs[enteredField].Query))
		} else {
			// We have not found any matches in allowed fields, return an error
			return nil, nil, errors.Errorf("Invalid sort field: %v", enteredField)
		}
		appliedFields = append(appliedFields, enteredField)
	}
	return tx, appliedFields, nil
}

func ParseFilters(q QueryMap, allowedFields database.AttrMap,
	defaultFilters map[string]FilterData) (Filters, error) {
	filters := Filters{}

	// Apply default filters
	for n, v := range defaultFilters {
		filters[n] = v
	}

	var err error
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
					panic("Nested operators not yet implemented for standard filters")
				}
				filters[f], err = ParseFilterValue(val)
			})
		}
	}

	return filters, err
}

type ListOpts struct {
	Fields         database.AttrMap
	DefaultFilters map[string]FilterData
	DefaultSort    string
	SearchFields   []string
	TotalFunc      totalFuncType
}

func ExportListCommon(tx *gorm.DB, c *gin.Context, opts ListOpts) (*gorm.DB, error) {
	query := NestedQueryMap(c, "filter")
	filters, err := ParseFilters(query, opts.Fields, opts.DefaultFilters)
	if err != nil {
		LogAndRespBadRequest(c, err, "Failed to parse filters")
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

// function type to return subtotals
type totalFuncType func(tx *gorm.DB) (int, map[string]int, error)

// generic function to return only total number of rows
func CountRows(tx *gorm.DB) (total int, subTotals map[string]int, err error) {
	var total64 int64
	err = tx.Count(&total64).Error
	return int(total64), subTotals, err
}

//nolint: funlen
func ListCommon(tx *gorm.DB, c *gin.Context, path string, opts ListOpts, params ...string) (
	*gorm.DB, *ListMeta, *Links, error) {
	limit, offset, err := utils.LoadLimitOffset(c, core.DefaultLimit)
	if err != nil {
		LogAndRespBadRequest(c, err, err.Error())
		return nil, nil, nil, errors.Wrap(err, "unable to parse limit, offset params")
	}
	tx, searchQ := ApplySearch(c, tx, opts.SearchFields...)

	tx, sortFields, err := ApplySort(c, tx, opts.Fields, opts.DefaultSort)
	if err != nil {
		LogAndRespBadRequest(c, err, err.Error())
		return nil, nil, nil, errors.Wrap(err, "invalid sort")
	}
	var sortQ string
	if len(sortFields) > 0 {
		sortQ = fmt.Sprintf("sort=%v", strings.Join(sortFields, ","))
	}

	query := NestedQueryMap(c, "filter")

	filters, err := ParseFilters(query, opts.Fields, opts.DefaultFilters)
	if err != nil {
		LogAndRespBadRequest(c, err, err.Error())
		return nil, nil, nil, errors.Wrap(err, "filters parsing failed")
	}

	tx, err = filters.Apply(tx, opts.Fields)
	if err != nil {
		LogAndRespBadRequest(c, err, err.Error())
		return nil, nil, nil, errors.Wrap(err, "filters applying failed")
	}

	// err = tx.Count(&total).Error
	total, subTotals, err := opts.TotalFunc(tx)
	if err != nil {
		LogAndRespError(c, err, "Database connection error")
		return nil, nil, nil, err
	}

	if offset > total {
		err = errors.New("Offset")
		LogAndRespBadRequest(c, err, InvalidOffsetMsg)
		return nil, nil, nil, err
	}

	meta := ListMeta{
		Limit:      limit,
		Offset:     offset,
		Filter:     filters,
		Sort:       sortFields,
		Search:     base.RemoveInvalidChars(c.Query("search")),
		TotalItems: total,
		SubTotals:  subTotals,
	}

	tagQ := extractTagsQueryString(c)

	params = append(params, filters.ToQueryParams(), sortQ, tagQ, searchQ)
	links := CreateLinks(path, offset, limit, total, params...)

	if limit != -1 {
		tx = tx.Limit(limit)
	}
	tx = tx.Offset(offset)
	return tx, &meta, &links, nil
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

func ParseTag(tag string) (*Tag, error) {
	matches := tagRegex.FindStringSubmatch(tag)
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
	ns := ""
	if t.Namespace != nil {
		ns = fmt.Sprintf(`"namespace": "%s",`, *t.Namespace)
	}

	v := ""
	if t.Value != nil {
		v = fmt.Sprintf(`, "value":"%s"`, *t.Value)
	}

	query := fmt.Sprintf(`[{%s "key": "%s" %s}]`, ns, t.Key, v)
	return tx.Where("h.tags @> ?::jsonb", query)
}

// Filter systems by tags,
func ApplyTagsFilter(c *gin.Context, tx *gorm.DB, systemIDExpr string) (*gorm.DB, bool, error) {
	if !enableCyndiTags {
		return tx, false, nil
	}
	var applied bool

	subq := database.Db.
		Table("inventory.hosts h").
		Select("h.id")

	tags := c.QueryArray("tags")
	for _, t := range tags {
		tag, err := ParseTag(t)
		if err != nil {
			LogAndRespBadRequest(c, err, err.Error())
			return tx, false, err
		}
		subq = tag.ApplyTag(subq)
		applied = true
	}

	// Additional filters
	filter := NestedQueryMap(c, "filter").Path("system_profile")
	if filter != nil {
		filter.Visit(func(path []string, val string) {
			if len(path) == 1 && path[0] == "sap_system" {
				subq = subq.Where("(h.system_profile ->> 'sap_system')::text = ?", val)
				applied = true
			}
			if len(path) >= 1 && path[0] == "sap_sids" {
				val = fmt.Sprintf(`"%s"`, val)
				subq = subq.Where("(h.system_profile ->> 'sap_sids')::jsonb @> ?::jsonb", val)
				applied = true
			}
		})
	}

	// Don't add the subquery if we don't have to
	if !applied {
		return tx, false, nil
	}
	return tx.Where(fmt.Sprintf("%s::uuid in (?)", systemIDExpr), subq), true, nil
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

func nestedQueryImpl(values map[string][]string, key string) QueryMap {
	root := QueryMap{}

	for name, value := range values {
		var steps []string
		var i int
		var j int
		for len(name) > 0 && i >= 0 && j >= 0 {
			if i = strings.IndexByte(name, '['); i >= 0 && (name[0:i] == key || len(steps) > 0) {
				// if j is 0 here, that means we received []as a part of query param name, should indicate an array
				if j = strings.IndexByte(name[i+1:], ']'); j >= 0 {
					// Skip [] in param names
					if len(name[i+1:][:j]) > 0 {
						steps = append(steps, name[i+1:][:j])
					}
					name = name[i+j+2:]
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
