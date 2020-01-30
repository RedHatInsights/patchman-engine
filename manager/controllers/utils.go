package controllers

import (
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

func ApplySort(c *gin.Context, tx *gorm.DB, allowedFields ...string) (*gorm.DB, error) {
	query := c.DefaultQuery("sort", "id")
	fields := strings.Split(query, ",")

	allowedFieldSet := map[string]bool{
		"id": true,
	}

	for _, f := range allowedFields {
		allowedFieldSet[f] = true
	}
	for _, enteredField := range fields {
		if strings.HasPrefix(enteredField, "-") && allowedFieldSet[enteredField[1:]] { //nolint:gocritic
			tx = tx.Order(fmt.Sprintf("%v DESC", enteredField[1:]))
		} else if allowedFieldSet[enteredField] {
			tx = tx.Order(fmt.Sprintf("%v ASC", enteredField))
		} else {
			// We have not found any matches in allowed fields, return an error
			return nil, errors.Errorf("Invalid sort field: %v", enteredField)
		}
	}
	return tx, nil
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
