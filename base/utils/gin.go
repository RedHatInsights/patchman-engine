package utils

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

const (
	KeyApiver          = "apiver"
	KeyAccount         = "account"
	KeyOrgID           = "org_id"
	KeyUser            = "user"
	KeySystem          = "system_cn"
	KeyInventoryGroups = "inventoryGroups"
	KeyGrouped         = "grouped"
	KeyUngrouped       = "ungrouped"
	// ReadHeaderTimeout same as nginx default
	ReadHeaderTimeout = 60 * time.Second
)

func LoadParamInt(c *gin.Context, param string, defaultValue int, query bool) (int, error) {
	var valueStr string
	if query {
		// don't use c.Query(), it caches query parameters in context
		// and data changed directly in c.Request.URL won't be returned
		// as they are not present in the cache
		valueStr = c.Request.URL.Query().Get(param)
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

	limit, err := LoadParamInt(c, "limit", defaultLimit, true)
	if err != nil {
		return 0, 0, err
	}

	if err := CheckLimitOffset(limit, offset); err != nil {
		return 0, 0, err
	}

	return limit, offset, nil
}

func CheckLimitOffset(limit int, offset int) error {
	if offset < 0 {
		return errors.New("offset must not be negative")
	}
	if CoreCfg.LimitPageSize {
		if limit < 1 || limit > 100 {
			return errors.New("limit must be in [1, 100]")
		}
	}
	if limit < 1 && limit != -1 {
		return errors.New("limit must not be less than 1, or should be -1 to return all items")
	}
	return nil
}

func IsParamValid(param *string, nullable, emptyAllowed bool) bool {
	if param == nil && !nullable {
		return false
	}

	if param != nil {
		if *param == "" && !emptyAllowed {
			return false
		}
		// string containing only whitespaces is not allowed by empty check in DB
		match, err := regexp.MatchString(`^\s+$`, *param)
		if err != nil || match {
			return false
		}
	}

	return true
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func RunServer(ctx context.Context, handler http.Handler, port int) error {
	addr := fmt.Sprintf(":%d", port)
	srv := http.Server{Addr: addr, Handler: handler, ReadHeaderTimeout: ReadHeaderTimeout, MaxHeaderBytes: 65535}
	go func() {
		<-ctx.Done()
		err := srv.Shutdown(context.Background())
		if err != nil {
			LogError("err", err.Error(), "server shutting down failed")
			return
		}
		LogInfo("server closed successfully")
	}()

	err := srv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return errors.Wrap(err, "server listening failed")
	}
	return nil
}
