package platform

import (
	"app/base/rbac"
	"app/base/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

var rbacPermissions = utils.Getenv("RBAC_PERMISSIONS", "patch:*:read")

var inventoryGroup = "inventory-group-1"

func rbacHandler(c *gin.Context) {
	c.JSON(http.StatusOK, rbac.AccessPagination{
		Data: []rbac.Access{
			{Permission: rbacPermissions},
			{
				Permission: "inventory:hosts:read",
				ResourceDefinitions: []rbac.ResourceDefinition{{
					AttributeFilter: rbac.AttributeFilter{
						Key:   "group.id",
						Value: []*string{&inventoryGroup, nil},
					},
				}},
			},
		},
	})
}

// InitInventory routes.
func initRbac(app *gin.Engine) {
	// Mock inventory system_profile endpoint
	app.GET("/api/rbac/v1/access", rbacHandler)
}
