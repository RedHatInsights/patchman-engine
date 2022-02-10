package platform

import (
	"app/base/rbac"
	"app/base/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

var rbacPermissions = utils.Getenv("RBAC_PERMISSIONS", "patch:*:read")

func rbacHandler(c *gin.Context) {
	c.JSON(http.StatusOK, rbac.AccessPagination{
		Data: []rbac.Access{
			{Permission: rbacPermissions},
		},
	})
}

// InitInventory routes.
func initRbac(app *gin.Engine) {
	// Mock inventory system_profile endpoint
	app.GET("/api/rbac/v1/access", rbacHandler)
}
