package middlewares

import (
	"app/base"
	"app/base/utils"
	"context"
	"github.com/RedHatInsights/patchman-clients/rbac"
	"github.com/gin-gonic/gin"
	"net/http"
)

// Make RBAC client on demand, with specified identity
func makeClient(identity string) *rbac.APIClient {
	traceAPI := utils.GetenvOrFail("LOG_LEVEL") == "trace"

	rbacConfig := rbac.NewConfiguration()
	rbacConfig.Debug = traceAPI
	rbacConfig.BasePath = utils.GetenvOrFail("RBAC_ADDRESS") + base.RBACApiPrefix
	rbacConfig.AddDefaultHeader("x-rh-identity", identity)

	return rbac.NewAPIClient(rbacConfig)
}

func checkRbac(c *gin.Context) bool {
	client := makeClient(c.GetHeader("x-rh-identity"))
	// Body is closed inside api method, don't know why liter is complaining
	// nolint: bodyclose
	access, _, err := client.AccessApi.GetPrincipalAccess(context.Background(), "patch", nil)

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
	return func(c *gin.Context) {
		if !checkRbac(c) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, utils.ErrorResponse{Error: "RBAC check failed"})
			return
		}
	}
}
