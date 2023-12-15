package deprecations

import (
	"app/base/utils"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Deprecate V1 and V2 APIs
func DeprecateV1V2APIs() Deprecation {
	redirectTS := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	return apiDeprecation{
		deprecationTimestamp: time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
		redirectTimestamp:    &redirectTS,
		// currentLocation is set by Deprecate receiver
		locationReplacer: strings.NewReplacer("v1", "v3", "v2", "v3"),
		message:          "APIs /v1 and /v2 are deprecated, use /v3 instead",
		shouldDeprecate: func(c *gin.Context) bool {
			apiver := c.GetInt(utils.KeyApiver)
			return apiver < 3
		},
	}
}

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
