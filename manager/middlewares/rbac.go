package middlewares

import (
	"app/base"
	"app/base/utils"
	"context"
	"github.com/RedHatInsights/patchman-clients/inventory"
	"github.com/RedHatInsights/patchman-clients/rbac"
	"github.com/gin-gonic/gin"
	"net/http"
)

var rbacClient *rbac.APIClient

// lazily configure rbac client only when middleware is created
func configure() {
	traceAPI := utils.GetenvOrFail("LOG_LEVEL") == "trace"

	rbacConfig := rbac.NewConfiguration()
	rbacConfig.Debug = traceAPI

	rbacConfig.BasePath = utils.GetenvOrFail("RBAC_ADDRESS") + base.RBACApiPrefix

	rbacClient = rbac.NewAPIClient(rbacConfig)
}

func checkRbac(c *gin.Context) bool {
	apiKey := rbac.APIKey{Prefix: "", Key: c.GetHeader("x-rh-identity")}
	// Create new context, which has the apikey value set. This is then used as a value for `x-rh-identity`
	ctx := context.WithValue(context.Background(), inventory.ContextAPIKey, apiKey)

	access, _, err := rbacClient.AccessApi.GetPrincipalAccess(ctx, "patch", nil)
	if err != nil {
		return false
	}

	for _, a := range access.Data {
		if a.Permission == "patch:*:*" {
			return true
		}
	}
	return false
}


func RBAC() gin.HandlerFunc {
	if rbacClient == nil {
		configure()
	}
	return func(c *gin.Context) {
		if !checkRbac(c) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, utils.ErrorResponse{Error: "RBAC check failed"})
			return
		}
	}
}