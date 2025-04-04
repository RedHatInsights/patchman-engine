package deprecations

import (
	"app/base/utils"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Deprecate maximum `limit`
func DeprecateLimit() Deprecation {
	return limitDeprecation{
		deprecationTimestamp: time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
		message:              "limit must be in [1, 100]",
		shouldDeprecate: func(c *gin.Context) bool {
			limit, err := utils.LoadParamInt(c, "limit", 20, true)
			if err == nil && (limit < 1 || limit > 100) {
				return true
			}
			return false
		},
	}
}

// Deprecate baselines api
func DeprecateBaselines() Deprecation {
	return apiDeprecation{
		deprecationTimestamp: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
		message:              "baselines are deprecated - https://access.redhat.com/articles/7097146",
		shouldDeprecate: func(c *gin.Context) bool {
			handlerName := c.HandlerName()
			return strings.Contains(strings.ToLower(handlerName), "baseline")
		},
	}
}
