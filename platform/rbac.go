package main

import (
	"github.com/RedHatInsights/patchman-clients/rbac"
	"github.com/gin-gonic/gin"
	"net/http"
)

func rbacHandler(c *gin.Context) {
	c.JSON(http.StatusOK, rbac.OneOfAccessPagination{
		Data: []rbac.Access{
			{Permission: "patch:*:*"},
		},
	})
}

// InitInventory routes.
func InitRbac(app *gin.Engine) {
	// Mock inventory system_profile endpoint
	app.GET("/api/rbac/v1/access", rbacHandler)
}
