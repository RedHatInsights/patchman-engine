package utils

import (
	"errors"
	"github.com/gin-gonic/gin"
	"strconv"
)

func SetCorsResponseHeaders(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	c.Writer.Header().Set("Access-Control-Allow-Methods", "GET")
	c.Writer.Header().Set("Access-Control-Max-Age", "21600")
}

func TryLoadParamInt(c *gin.Context, param string, query bool) (bool, int, error) {
	var valueStr string
	if query {
		valueStr = c.Query(param)
	} else {
		valueStr = c.Param(param)
	}
	if valueStr == "" {
		return false, 0, nil
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return true, 0, err
	}

	return true, value, nil
}

func LoadParamInt(c *gin.Context, param string, defaultValue int, query bool) (int, error) {
	var valueStr string
	if query {
		valueStr = c.Query(param)
	} else {
		valueStr = c.Param(param)
	}
	if valueStr == "" {
		return defaultValue, nil
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return 0, err
	}

	return value, nil
}

func LoadLimitOffset(c *gin.Context, defaultLimit int) (int, int, error) {
	offset, err := LoadParamInt(c, "offset", 0, true)
	if err != nil {
		return 0, 0, err
	}

	if offset < 0 {
		return 0, 0, errors.New("offset must not be negative")
	}

	limit, err := LoadParamInt(c, "limit", defaultLimit, true)
	if err != nil {
		return 0, 0, err
	}

	if limit < 1 {
		return 0, 0, errors.New("limit must not be less than 1")
	}

	return limit, offset, nil
}
