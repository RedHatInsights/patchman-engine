package middlewares

import (
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

func Gzip(options ...gzip.Option) gin.HandlerFunc {
	gzipFn := gzip.Gzip(gzip.DefaultCompression, options...)
	return func(c *gin.Context) {
		tempLogDebugGinContextRequestHeader(c, "Gzip before")
		defer tempLogDebugGinContextRequestHeader(c, "Gzip after")
		gzipFn(c)
	}
}
