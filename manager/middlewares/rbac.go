package middlewares

import (
	"app/base"
	"app/base/utils"
	"github.com/RedHatInsights/patchman-clients/rbac"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"strings"
)

// Make RBAC client on demand, with specified identity
func makeClient(identity string) *rbac.APIClient {
	rbacConfig := rbac.NewConfiguration()
	useTraceLevel := strings.ToLower(utils.Getenv("LOG_LEVEL", "INFO")) == "trace"
	rbacConfig.Debug = useTraceLevel
	rbacConfig.Servers[0].URL = utils.GetenvOrFail("RBAC_ADDRESS") + base.RBACApiPrefix
	rbacConfig.AddDefaultHeader("x-rh-identity", identity)
	return rbac.NewAPIClient(rbacConfig)
}

type rbacPerms struct {
	Read  bool
	Write bool
}

func isAccessGranted(c *gin.Context) rbacPerms {
	client := makeClient(c.GetHeader("x-rh-identity"))
	// Body is closed inside api method, don't know why liter is complaining
	// nolint: bodyclose
	access, res, err := client.AccessApi.GetPrincipalAccess(base.Context).Application("patch").Execute()
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
		return rbacPerms{Read: false, Write: false}
	}

	perms := rbacPerms{Read: false, Write: false}
	for _, a := range access.Data {
		switch a.Permission {
		case "patch:*:*", "patch:system:*":
			perms.Read = true
			perms.Write = true
		case "patch:*:read", "patch:system:read":
			perms.Read = true
		case "patch:*:write", "patch:system:write":
			perms.Write = true
		default:
		}
	}
	return perms
}

func RBAC() gin.HandlerFunc {
	enableRBACCHeck := utils.GetBoolEnvOrDefault("ENABLE_RBAC", true)
	if !enableRBACCHeck {
		return func(c *gin.Context) {}
	}

	return func(c *gin.Context) {
		grantedPerms := isAccessGranted(c)

		switch c.Request.Method {
		case "POST":
			if grantedPerms.Read || grantedPerms.Write {
				return
			}
		case "GET":
			if grantedPerms.Read {
				return
			}
		case "DELETE":
			if grantedPerms.Write {
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusUnauthorized,
			utils.ErrorResponse{Error: "You don't have access to this application"})
	}
}
