package platform

import (
	"app/base/rbac"
	"app/base/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

var rbacPermissions = utils.PodConfig.GetString("rbac_permissions", "patch:*:read")

var inventoryGroup = "inventory-group-1"

func rbacHandler(c *gin.Context) {
	c.JSON(http.StatusOK, rbac.AccessPagination{
		Data: []rbac.Access{
			{Permission: rbacPermissions},
			{
				Permission: "inventory:hosts:read",
				ResourceDefinitions: []rbac.ResourceDefinition{{
					AttributeFilter: rbac.AttributeFilter{
						Key:       "group.id",
						Operation: "in",
						Value:     []*string{&inventoryGroup, nil},
					},
				}},
			},
		},
	})
}

func workspacesHandler(c *gin.Context) {
	workspaceType := c.Query("type")
	switch workspaceType {
	case "default":
		c.Data(http.StatusOK, "application/json", []byte(`{"meta":{"count":1,"limit":10,"offset":0},"links":{"first":`+
			`"/api/rbac/v2/workspaces/?limit=10&offset=0&type=default","next":null,"previous":null,"last":`+
			`"/api/rbac/v2/workspaces/?limit=10&offset=0&type=default"},"data":[{"name":"Default Workspace","id":`+
			`"00000000-0000-0000-0000-000000000001","parent_id":"00000000-0000-0000-0000-000000000002","description":null,`+
			`"created":"2025-09-01T08:00:42.141526Z","modified":"2025-09-02T08:00:42.400009Z","type":"default"}]}`,
		))
	case "root":
		c.Data(http.StatusOK, "application/json", []byte(`{"meta":{"count":1,"limit":10,"offset":0},"links":{"first":`+
			`"/api/rbac/v2/workspaces/?limit=10&offset=0&type=root","next":null,"previous":null,"last":`+
			`"/api/rbac/v2/workspaces/?limit=10&offset=0&type=root"},"data":[{"name":"Root Workspace","id":`+
			`"00000000-0000-0000-0000-000000000002","parent_id":null,"description":null,`+
			`"created":"2025-09-01T08:00:42.141526Z","modified":"2025-09-02T08:00:42.400009Z","type":"root"}]}`,
		))
	}
}

// InitInventory routes.
func initRbac(app *gin.Engine) {
	// Mock inventory system_profile endpoint
	app.GET("/api/rbac/v1/access", rbacHandler)
	app.GET("/api/rbac/v2/workspaces/", workspacesHandler)
}
