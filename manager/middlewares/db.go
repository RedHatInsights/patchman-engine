package middlewares

import (
	"app/base/database"
	"app/base/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const DBKey = "DB"
const DBReadReplicaKey = "DBReadReplica"

// Apply gin context to database so queries within context are canceled when request is aborted
func DatabaseWithContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(DBKey, database.Db.WithContext(c))
		if database.DbReadReplica != nil {
			c.Set(DBReadReplicaKey, database.DbReadReplica.WithContext(c))
		}
		c.Next()
	}
}

// DB handler stored in request context
func DBFromContext(c *gin.Context) *gorm.DB {
	if useReadReplica(c) {
		return c.MustGet(DBReadReplicaKey).(*gorm.DB)
	}
	return c.MustGet(DBKey).(*gorm.DB)
}

func useReadReplica(c *gin.Context) bool {
	if utils.Cfg.DBReadReplicaEnabled && c.Request.Method == http.MethodGet {
		// if Host or Port is not set, don't use read replica
		return database.ReadReplicaConfigured()
	}
	return false
}
