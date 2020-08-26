package middlewares

import (
	"app/base"
	"app/base/utils"
	"github.com/RedHatInsights/patchman-clients/rbac"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
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

func isAccessGranted(c *gin.Context) bool {
	client := makeClient(c.GetHeader("x-rh-identity"))
	// Body is closed inside api method, don't know why liter is complaining
	// nolint: bodyclose
	access, res, err := client.AccessApi.GetPrincipalAccess(base.Context, "patch", nil)
	if res != nil && res.Body != nil {
		defer res.Body.Close()
	}

	if err != nil {
		utils.Log("err", err.Error()).Error("Call to RBAC svc failed")
		status := http.StatusInternalServerError
		if res != nil {
			status = res.StatusCode
		}
		serviceErrorCnt.WithLabelValues("rbac", strconv.Itoa(status)).Inc()
		return false
	}
	// For now we either allow access or don't allow it
	for _, a := range access.Data {
		if a.Permission == "patch:*:*" {
			return true
		}

		if a.Permission == "patch:all:all" { // to support custom roles
			return true
		}
	}
	utils.Log().Trace("Access denied by RBAC")
	return false
}

func RBAC() gin.HandlerFunc {
	enableRBACCHeck := utils.GetBoolEnvOrDefault("ENABLE_RBAC", true)
	if !enableRBACCHeck {
		return func(c *gin.Context) {}
	}

	return func(c *gin.Context) {
		if !isAccessGranted(c) {
			c.AbortWithStatusJSON(http.StatusUnauthorized,
				utils.ErrorResponse{Error: "You don't have access to this application"})
		}
	}
}
