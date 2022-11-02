package middlewares

import (
	"app/base"
	"app/base/api"
	"app/base/rbac"
	"app/base/utils"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
)

var (
	rbacURL      = ""
	debugRequest = os.Getenv("LOG_LEVEL") == "trace"
	httpClient   = &http.Client{}
)

const xRHIdentity = "x-rh-identity"

// Make RBAC client on demand, with specified identity
func makeClient(identity string) *api.Client {
	client := api.Client{
		HTTPClient:     httpClient,
		Debug:          debugRequest,
		DefaultHeaders: map[string]string{xRHIdentity: identity},
	}
	if rbacURL == "" {
		rbacURL = utils.FailIfEmpty(utils.Cfg.RbacAddress, "RBAC_ADDRESS") + base.RBACApiPrefix + "/access/?application=patch"
	}
	return &client
}

type rbacPerms struct {
	Read  bool
	Write bool
}

func isAccessGranted(c *gin.Context) rbacPerms {
	client := makeClient(c.GetHeader("x-rh-identity"))
	access := rbac.AccessPagination{}
	res, err := client.Request(&base.Context, http.MethodGet, rbacURL, nil, &access)
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
		case "GET", "POST":
			if grantedPerms.Read {
				return
			}
		case "DELETE", "PUT":
			if grantedPerms.Write {
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusUnauthorized,
			utils.ErrorResponse{Error: "You don't have access to this application"})
	}
}
