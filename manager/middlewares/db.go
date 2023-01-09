package middlewares

import (
	"app/base/database"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const DBKey = "DB"

// Apply gin context to database so queries within context are canceled when request is aborted
func DatabaseWithContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(DBKey, database.Db.WithContext(c))
		c.Next()
	}
}

// DB handler stored in request context
func DBFromContext(c *gin.Context) *gorm.DB {
	return c.MustGet(DBKey).(*gorm.DB)
}
