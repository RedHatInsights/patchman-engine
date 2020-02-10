package controllers

import (
	"app/base/core"
	"app/base/utils"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"net/http"
	"strings"
)

const InvalidOffsetMsg = "Invalid offset"

type AttrName = string
type AttrQuery = string

// Used to store field name => sql query mapping
type AttrMap = map[AttrName]AttrQuery

func MakeSelect(attrs AttrMap) string {
	fields := make([]string, 0, len(attrs))
	for n, q := range attrs {
		fields = append(fields, fmt.Sprintf("%v as %v", q, n))
	}
	return strings.Join(fields, ",")
}

func LogAndRespError(c *gin.Context, err error, respMsg string) {
	utils.Log("err", err.Error()).Error(respMsg)
	c.AbortWithStatusJSON(http.StatusInternalServerError, utils.ErrorResponse{Error: respMsg})
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
func ApplySort(c *gin.Context, tx *gorm.DB, fieldExprs AttrMap) (*gorm.DB, []string, error) {
	query := c.DefaultQuery("sort", "id")
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
			tx = tx.Order(fmt.Sprintf("%v DESC", fieldExprs[enteredField[1:]]))
		} else if allowedFieldSet[enteredField] {
			tx = tx.Order(fmt.Sprintf("%v ASC", fieldExprs[enteredField]))
		} else {
			// We have not found any matches in allowed fields, return an error
			return nil, nil, errors.Errorf("Invalid sort field: %v", enteredField)
		}
		appliedFields = append(appliedFields, enteredField)
	}
	return tx, appliedFields, nil
}

func ParseFilters(c *gin.Context, allowedFields AttrMap) (Filters, error) {
	queryFilters, has := c.GetQueryMap("filter")
	filters := Filters{}
	if !has {
		return []Filter{}, nil
	}
	for k, v := range queryFilters {
		filter, err := ParseFilterValue(k, v)
		if err != nil {
			c.AbortWithStatusJSON(500, err)
		}
		utils.Log("filter", filter).Debug("Successfully parsed filter")
		filters = append(filters, filter)
	}
	return filters.FilterFilters(allowedFields)
}

// nolint:lll, funlen
func ListCommon(tx *gorm.DB, c *gin.Context, fields AttrMap, path string) (*gorm.DB, *ListMeta, *Links, error) {
	limit, offset, err := utils.LoadLimitOffset(c, core.DefaultLimit)
	if err != nil {
		LogAndRespBadRequest(c, err, err.Error())
		return nil, nil, nil, err
	}

	tx, sortFields, err := ApplySort(c, tx, fields)
	if err != nil {
		LogAndRespBadRequest(c, err, "Invalid sort")
		return nil, nil, nil, err
	}

	filters, err := ParseFilters(c, fields)
	if err != nil {
		LogAndRespBadRequest(c, err, "Invalid filter")
		return nil, nil, nil, err
	}

	for _, f := range filters {
		query, args, err := f.ToWhere(fields)
		if err != nil {
			LogAndRespBadRequest(c, err, "Invalid filter")
			return nil, nil, nil, err
		}
		tx = tx.Where(query, args...)
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
		Page:       offset / limit,
		PageSize:   limit,
		Pages:      total / limit,
		Filter:     filters.ToMetaMap(),
		Sort:       sortFields,
		TotalItems: total,
	}

	var sortQ string
	if len(sortFields) > 0 {
		sortQ = fmt.Sprintf("sort=%v", strings.Join(sortFields, ","))
	}

	links := CreateLinks(path, offset, limit, total, filters.ToQueryParams(), sortQ)

	tx = tx.Limit(limit).Offset(offset)
	return tx, &meta, &links, nil
}

func ApplySearch(c *gin.Context, tx *gorm.DB, searchColumns ...string) *gorm.DB {
	search := c.Query("search")
	if search == "" {
		return tx
	}

	if len(searchColumns) == 0 {
		return tx
	}

	searchExtended := "%" + search + "%"
	txWithSearch := tx.Where("LOWER("+searchColumns[0]+") LIKE LOWER(?)", searchExtended)
	if len(searchColumns) > 1 {
		for _, searchColumn := range searchColumns[1:] {
			txWithSearch = txWithSearch.Or("LOWER("+searchColumn+") LIKE LOWER(?)", searchExtended)
		}
	}
	return txWithSearch
}
