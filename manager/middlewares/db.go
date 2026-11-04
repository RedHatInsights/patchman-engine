package middlewares

import (
	"app/base/database"
	"app/base/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// DB returns a handle bound to the current request context.
// GET uses the read replica when one is configured.
func DB(c *gin.Context) *gorm.DB {
	if useReadReplica(c) {
		return database.DBReadReplica(c.Request.Context())
	}
	return database.DB(c.Request.Context())
}

// DBWriter returns the primary handle bound to the current request context.
func DBWriter(c *gin.Context) *gorm.DB {
	return database.DB(c.Request.Context())
}

func useReadReplica(c *gin.Context) bool {
	if utils.CoreCfg.DBReadReplicaEnabled && c.Request.Method == http.MethodGet {
		// if Host or Port is not set, don't use read replica
		return database.ReadReplicaConfigured()
	}
	return false
}
