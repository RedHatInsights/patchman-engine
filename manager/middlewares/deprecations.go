package middlewares

import (
	"app/base/deprecations"

	"github.com/gin-gonic/gin"
)

func Deprecate(options ...deprecations.Deprecation) gin.HandlerFunc {
	return func(c *gin.Context) {
		for _, o := range options {
			o.Deprecate(c)
		}
	}
}
