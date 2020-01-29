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

func LogAndRespError(c *gin.Context, err error, respMsg string) {
	utils.Log("err", err.Error()).Error(respMsg)
	c.JSON(http.StatusInternalServerError, utils.ErrorResponse{Error: respMsg})
}

func LogAndRespBadRequest(c *gin.Context, err error, respMsg string) {
	utils.Log("err", err.Error()).Warn(respMsg)
	c.JSON(http.StatusBadRequest, utils.ErrorResponse{Error: respMsg})
}

func LogAndRespNotFound(c *gin.Context, err error, respMsg string) {
	utils.Log("err", err.Error()).Warn(respMsg)
	c.JSON(http.StatusNotFound, utils.ErrorResponse{Error: respMsg})
}

func ApplySort(c *gin.Context, tx *gorm.DB, allowedFields ...string) (*gorm.DB, []string, error) {
	query := c.DefaultQuery("sort", "id")
	fields := strings.Split(query, ",")
	var appliedFields []string
	allowedFieldSet := map[string]bool{
		"id": true,
	}

	for _, f := range allowedFields {
		allowedFieldSet[f] = true
	}
	for _, enteredField := range fields {
		if strings.HasPrefix(enteredField, "-") && allowedFieldSet[enteredField[1:]] { //nolint:gocritic
			tx = tx.Order(fmt.Sprintf("%v DESC", enteredField[1:]))
			appliedFields = append(appliedFields, enteredField[1:])
		} else if allowedFieldSet[enteredField] {
			tx = tx.Order(fmt.Sprintf("%v ASC", enteredField))
			appliedFields = append(appliedFields, enteredField)
		} else {
			// We have not found any matches in allowed fields, return an error
			return nil, nil, errors.Errorf("Invalid sort field: %v", enteredField)
		}
	}
	return tx, appliedFields, nil
}

// nolint:lll
func ListCommon(tx *gorm.DB, c *gin.Context, allowedFields []string, path string) (*gorm.DB, *ListMeta, *Links, error) {
	limit, offset, err := utils.LoadLimitOffset(c, core.DefaultLimit)
	if err != nil {
		LogAndRespBadRequest(c, err, err.Error())
		return nil, nil, nil, err
	}

	tx, sortFields, err := ApplySort(c, tx, allowedFields...)
	if err != nil {
		LogAndRespBadRequest(c, err, "Invalid sort")
		return nil, nil, nil, err
	}

	var total int
	err = tx.Count(&total).Error
	if err != nil {
		LogAndRespError(c, err, "Database connection error")
		return nil, nil, nil, err
	}

	if offset > total {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{Error: "too big offset"})
		return nil, nil, nil, err
	}
	meta := ListMeta{
		Limit:      limit,
		Offset:     offset,
		Page:       offset / limit,
		PageSize:   limit,
		Pages:      total / limit,
		Sort:       sortFields,
		TotalItems: total,
	}
	tx = tx.Limit(limit).Offset(offset)
	// TODO: Sort fields on other params
	links := CreateLinks(path, offset, limit, total, "")
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
