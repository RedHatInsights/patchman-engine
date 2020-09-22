package controllers

import (
	"app/base"
	"app/base/core"
	"app/base/database"
	"app/base/utils"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gocarina/gocsv"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"net/http"
	"regexp"
	"strings"
)

const InvalidOffsetMsg = "Invalid offset"

var tagRegex = regexp.MustCompile(`([^/=]+)/([^/=]+)=([^/=]+)`)
var enableCyndiTags = utils.GetBoolEnvOrDefault("ENABLE_CYNDI_TAGS", false)

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
			tx = tx.Order(fmt.Sprintf("%s DESC", fieldExprs[enteredField[1:]].Query))
		} else if allowedFieldSet[enteredField] {
			tx = tx.Order(fmt.Sprintf("%s ASC", fieldExprs[enteredField].Query))
		} else {
			// We have not found any matches in allowed fields, return an error
			return nil, nil, errors.Errorf("Invalid sort field: %v", enteredField)
		}
		appliedFields = append(appliedFields, enteredField)
	}
	return tx, appliedFields, nil
}

func ParseFilters(c *gin.Context, allowedFields database.AttrMap,
	defaultFilters map[string]FilterData) (Filters, error) {
	filters := Filters{}

	// Apply default filters
	for n, v := range defaultFilters {
		filters[n] = v
	}

	// Apply query filters, if there are any
	queryFilters, has := c.GetQueryMap("filter")
	if !has {
		return filters, nil
	}
	for k, v := range queryFilters {
		if _, has := allowedFields[k]; !has {
			return nil, errors.New(fmt.Sprintf("Invalid filter field: %v", k))
		}
		filter, err := ParseFilterValue(v)
		if err != nil {
			return nil, err
		}
		filters[k] = filter
	}
	return filters, nil
}

type ListOpts struct {
	Fields         database.AttrMap
	DefaultFilters map[string]FilterData
	DefaultSort    string
}

func checkBadRequest(tx *gorm.DB, c *gin.Context, opts ListOpts) (txx *gorm.DB, limit int, offset int,
	sortFields []string, filters Filters, err error) {
	limit, offset, err = utils.LoadLimitOffset(c, core.DefaultLimit)
	if err != nil {
		return nil, 0, 0, nil, nil, errors.
			Wrap(err, "unable to parse limit, offset params")
	}

	tx, sortFields, err = ApplySort(c, tx, opts.Fields, opts.DefaultSort)
	if err != nil {
		return nil, 0, 0, nil, nil, errors.Wrap(err, "invalid sort")
	}

	filters, err = ParseFilters(c, opts.Fields, opts.DefaultFilters)
	if err != nil {
		return nil, 0, 0, nil, nil, errors.Wrap(err, "filters parsing failed")
	}

	tx, err = filters.Apply(tx, opts.Fields)
	if err != nil {
		return nil, 0, 0, nil, nil, errors.Wrap(err, "filters applying failed")
	}

	return tx, limit, offset, sortFields, filters, nil
}

func ExportListCommon(tx *gorm.DB, c *gin.Context, opts ListOpts) (*gorm.DB, error) {
	filters, err := ParseFilters(c, opts.Fields, opts.DefaultFilters)
	if err != nil {
		LogAndRespBadRequest(c, err, "Failed to parse filters")
		return nil, errors.Wrap(err, "filters parsing failed")
	}

	tx, err = filters.Apply(tx, opts.Fields)
	if err != nil {
		LogAndRespBadRequest(c, err, "Failed to apply filters")
		return nil, errors.Wrap(err, "filters applying failed")
	}
	return tx, nil
}

func ListCommon(tx *gorm.DB, c *gin.Context, path string, opts ListOpts) (*gorm.DB, *ListMeta, *Links, error) {
	tx, limit, offset, sortFields, filters, err := checkBadRequest(tx, c, opts)
	if err != nil {
		LogAndRespBadRequest(c, err, err.Error())
		return nil, nil, nil, err
	}

	var total int
	err = tx.Count(&total).Error
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
		TotalItems: total,
	}

	var sortQ string
	if len(sortFields) > 0 {
		sortQ = fmt.Sprintf("sort=%v", strings.Join(sortFields, ","))
	}

	links := CreateLinks(path, offset, limit, total, filters.ToQueryParams(), sortQ)

	if limit != -1 {
		tx = tx.Limit(limit)
	}
	tx = tx.Offset(offset)
	return tx, &meta, &links, nil
}

func ApplySearch(c *gin.Context, tx *gorm.DB, searchColumns ...string) *gorm.DB {
	search := base.RemoveInvalidChars(c.Query("search"))
	if search == "" {
		return tx
	}

	if len(searchColumns) == 0 {
		return tx
	}

	searchExtended := "%" + search + "%"
	concatValue := strings.Join(searchColumns, ",' ',")
	txWithSearch := tx.Where("LOWER(CONCAT("+concatValue+")) LIKE LOWER(?)", searchExtended)
	return txWithSearch
}

func HasTags(c *gin.Context) bool {
	if !enableCyndiTags {
		return false
	}
	if len(c.QueryArray("tags")) == 0 {
		return false
	}
	return true
}

// Filter systems by tags,
func ApplyTagsFilter(c *gin.Context, tx *gorm.DB, systemIDExpr string) (*gorm.DB, bool) {
	if !enableCyndiTags {
		return tx, false
	}
	tags := c.QueryArray("tags")
	if len(tags) == 0 {
		return tx, false
	}

	subq := database.Db.
		Table("inventory.hosts h").
		Select("h.id::text")

	for _, t := range tags {
		matches := tagRegex.FindStringSubmatch(t)
		tagJSON := fmt.Sprintf(`[{"namespace":"%s", "key": "%s", "value": "%s"}]`, matches[1], matches[2], matches[3])
		subq = subq.Where(" h.tags @> ?::jsonb", tagJSON)
	}

	return tx.Where(fmt.Sprintf("%s in (?)", systemIDExpr), subq.SubQuery()), true
}

func Csv(ctx *gin.Context, code int, res interface{}) {
	ctx.Status(http.StatusOK)
	ctx.Header("Content-Type", "text/csv")
	err := gocsv.Marshal(res, ctx.Writer)
	if err != nil {
		panic(err)
	}
}
