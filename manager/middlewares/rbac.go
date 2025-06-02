package middlewares

import (
	"app/base"
	"app/base/api"
	"app/base/rbac"
	"app/base/utils"
	"app/manager/config"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
)

var (
	rbacURL    = ""
	httpClient = &http.Client{}
)

const xRHIdentity = "x-rh-identity"

const inventoryHostsReadPerm = "inventory:hosts:read"
const patchReadPerm = "patch:*:read"
const patchWritePerm = "patch:*:write"

// handlerName to permissions mapping
var granularPerms = map[string]string{
	"CreateBaselineHandler":        "patch:template:write",
	"BaselineUpdateHandler":        "patch:template:write",
	"BaselineDeleteHandler":        "patch:template:write",
	"BaselineSystemsRemoveHandler": "patch:template:write",
	"TemplateSystemsUpdateHandler": "content-sources:templates:write",
	"TemplateSystemsDeleteHandler": "content-sources:templates:write",
	"SystemDeleteHandler":          "patch:system:write",
}

// Make RBAC client on demand, with specified identity
func makeClient(identity string) *api.Client {
	debugRequest := log.IsLevelEnabled(log.TraceLevel)

	client := api.Client{
		HTTPClient:     httpClient,
		Debug:          debugRequest,
		DefaultHeaders: map[string]string{xRHIdentity: identity},
	}
	if rbacURL == "" {
		rbacURL = utils.FailIfEmpty(utils.CoreCfg.RbacAddress, "RBAC_ADDRESS") + base.RBACApiPrefix +
			"/access/?application=patch,inventory,content-sources"
	}
	return &client
}

// for short lists like that is slice.Contains() faster than map lookup _, ok := map[key]
func expandedPermission(perm string) []string {
	comp := strings.SplitAfterN(perm, ":", 3)
	expandedPerm := []string{perm,
		comp[0] + comp[1] + "*",
		comp[0] + "*:" + comp[2],
		"*:" + comp[1] + comp[2],
		comp[0] + "*:*",
		"*:" + comp[1] + "*",
		"*:*:" + comp[2],
		"*:*:*",
	}
	return expandedPerm
}

func checkPermissions(access *rbac.AccessPagination, handlerName, method string) bool {
	// always need inventory:hosts:read
	grantedInventory := false
	inventoryHostsReadPerms := expandedPermission(inventoryHostsReadPerm)

	// API handler specific permission
	grantedPatch := false
	patchNeededPerms := []string{}
	if p, has := granularPerms[handlerName]; has {
		patchNeededPerms = expandedPermission(p)
	} else {
		// not granular
		// require read permissions for GET and POST
		// require write permissions for PUT and DELETE
		switch method {
		case "GET", "POST":
			patchNeededPerms = expandedPermission(patchReadPerm)
		case "PUT", "DELETE":
			patchNeededPerms = expandedPermission(patchWritePerm)
		}
	}

	for _, a := range access.Data {
		if !grantedInventory {
			grantedInventory = slices.Contains(inventoryHostsReadPerms, a.Permission)
		}
		if !grantedPatch {
			grantedPatch = slices.Contains(patchNeededPerms, a.Permission)
		}
		if grantedPatch && grantedInventory {
			return true
		}
	}
	return false
}

func isAccessGranted(c *gin.Context) bool {
	client := makeClient(c.GetHeader("x-rh-identity"))
	access := rbac.AccessPagination{}
	res, err := client.Request(&base.Context, http.MethodGet, rbacURL, nil, &access)
	if res != nil && res.Body != nil {
		defer res.Body.Close()
	}

	if err != nil {
		utils.LogError("err", err.Error(), "Call to RBAC svc failed")
		status := http.StatusInternalServerError
		if res != nil {
			status = res.StatusCode
		}
		serviceErrorCnt.WithLabelValues("rbac", strconv.Itoa(status)).Inc()
		return false
	}
	nameSplitted := strings.Split(c.HandlerName(), ".")
	handlerName := nameSplitted[len(nameSplitted)-1]

	granted := checkPermissions(&access, handlerName, c.Request.Method)
	if granted {
		// collect inventory groups
		groups, err := findInventoryGroups(&access)
		if err != nil {
			utils.LogError("err", err.Error(), "RBAC")
			granted = false
		}
		c.Set(utils.KeyInventoryGroups, groups)
	}
	return granted
}

func findInventoryGroups(access *rbac.AccessPagination) (map[string]string, error) {
	res := make(map[string]string)

	if len(access.Data) == 0 {
		return res, nil
	}
	inventoryHostsReadPerms := expandedPermission(inventoryHostsReadPerm)
	groups := []string{}
	for _, a := range access.Data {
		// look for groups only on inventory:hosts:read permissions
		if !slices.Contains(inventoryHostsReadPerms, a.Permission) {
			continue
		}

		if len(a.ResourceDefinitions) == 0 {
			// access to all groups
			return nil, nil
		}
		for _, rd := range a.ResourceDefinitions {
			if rd.AttributeFilter.Key != "group.id" {
				continue
			}

			if rd.AttributeFilter.Operation != "in" && rd.AttributeFilter.Operation != "equal" {
				err := fmt.Errorf(
					"invalid value '%s' for attributeFilter.Operation",
					rd.AttributeFilter.Operation,
				)
				return nil, err
			}
			for _, v := range rd.AttributeFilter.Value {
				if v == nil {
					res[utils.KeyUngrouped] = "[]"
					continue
				}
				group, err := utils.ParseInventoryGroup(v, nil)
				if err != nil {
					// couldn't marshal inventory group to json
					continue
				}
				groups = append(groups, group)
			}
		}
	}

	if len(groups) > 0 {
		res[utils.KeyGrouped] = fmt.Sprintf("{%s}", strings.Join(groups, ","))
	}
	return res, nil
}

func RBAC() gin.HandlerFunc {
	if !config.EnableRBACCHeck {
		return func(_ *gin.Context) {}
	}

	return func(c *gin.Context) {
		if isAccessGranted(c) {
			return
		}
		c.AbortWithStatusJSON(http.StatusUnauthorized,
			utils.ErrorResponse{Error: "You don't have access to this application"})
	}
}
