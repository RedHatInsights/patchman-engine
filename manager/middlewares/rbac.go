package middlewares

import (
	"app/base"
	"app/base/types/rbac"
	"app/base/utils"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

var rbacURL = fmt.Sprintf(
	"%s%s/access/?application=patch",
	utils.FailIfEmpty(utils.Cfg.RbacAddress, "RBAC_ADDRESS"),
	base.RBACApiPrefix,
)

var allPerms = "patch:*:*"
var readPerms = map[string]bool{allPerms: true, "patch:*:read": true}
var writePerms = map[string]bool{allPerms: true, "patch:*:write": true}

// handlerName to permissions mapping
var granularPerms = map[string]struct {
	Permission string
	Read       bool
	Write      bool
}{
	"CreateBaselineHandler":        {"patch:template:write", false, true},
	"BaselineUpdateHandler":        {"patch:template:write", false, true},
	"BaselineDeleteHandler":        {"patch:template:write", false, true},
	"BaselineSystemsRemoveHandler": {"patch:template:write", false, true},
	"SystemDeleteHandler":          {"patch:system:write", false, true},
}

func checkPermissions(access *rbac.AccessPagination, handlerName, method string) bool {
	granted := false
	for _, a := range access.Data {
		if granted {
			return true
		}
		if p, has := granularPerms[handlerName]; has {
			// API handler requires granular permissions
			if a.Permission == p.Permission {
				// the required permission is present, e.g. patch:template:write
				return true
			}
			if p.Read && !p.Write && readPerms[a.Permission] {
				// required permission is read permission
				// check whether we have either patch:*:read or patch:*:*
				return true
			}
			if p.Write && !p.Read && writePerms[a.Permission] {
				// required permission is write permission
				// check whether we have either patch:*:write or patch:*:*
				return true
			}
			// we need both read and write permissions - patch:*:*
			granted = (a.Permission == allPerms)
		} else {
			// not granular
			// require read permissions for GET and POST
			// require write permissions for PUT and DELETE
			switch method {
			case "GET", "POST":
				granted = readPerms[a.Permission]
			case "PUT", "DELETE":
				granted = writePerms[a.Permission]
			}
		}
	}
	return granted
}

func isAccessGranted(c *gin.Context) bool {
	client := makeClient(c.GetHeader("x-rh-identity"))
	access := rbac.AccessPagination{}
	err := makeRequest(client, &base.Context, rbacURL, "RBAC", &access)
	if err != nil {
		return false
	}
	nameSplitted := strings.Split(c.HandlerName(), ".")
	handlerName := nameSplitted[len(nameSplitted)-1]

	return checkPermissions(&access, handlerName, c.Request.Method)
}

func RBAC() gin.HandlerFunc {
	enableRBACCHeck := utils.GetBoolEnvOrDefault("ENABLE_RBAC", true)
	if !enableRBACCHeck {
		return func(c *gin.Context) {}
	}

	return func(c *gin.Context) {
		if isAccessGranted(c) {
			return
		}
		c.AbortWithStatusJSON(http.StatusUnauthorized,
			utils.ErrorResponse{Error: "You don't have access to this application"})
	}
}
