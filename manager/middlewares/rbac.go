package middlewares

import (
	"app/base"
	"app/base/api"
	"app/base/rbac"
	"app/base/telemetry"
	"app/base/utils"
	"app/manager/config"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
)

const rootWorkspaceID = "00000000-0000-0000-0000-999999999999"

var (
	rbacURL    = ""
	httpClient = &http.Client{}
	httpOnce   sync.Once
)

const xRHIdentity = "x-rh-identity"

const inventoryHostsReadPerm = "inventory:hosts:read"
const patchReadPerm = "patch:*:read"
const patchWritePerm = "patch:*:write"

// handlerName to permissions mapping
var granularPerms = map[string]string{
	"TemplateSystemsUpdateHandler": "content-sources:templates:write",
	"TemplateSystemsDeleteHandler": "content-sources:templates:write",
	"SystemDeleteHandler":          "patch:system:write",
}

// Make RBAC client on demand, with specified identity and optional gateway bearer token
func makeClient(identity, bearer string) *api.Client {
	debugRequest := log.IsLevelEnabled(log.TraceLevel)

	headers := map[string]string{xRHIdentity: identity}
	if bearer != "" {
		headers["Authorization"] = "Bearer " + bearer
	}
	return &api.Client{
		HTTPClient:     httpClient,
		Debug:          debugRequest,
		DefaultHeaders: headers,
	}
}

func rbacTLSConfig() *tls.Config {
	pem, err := os.ReadFile(utils.CoreCfg.RbacCACert)
	if err != nil {
		panic(fmt.Sprintf("failed to read RBAC CA certificate %s: %v", utils.CoreCfg.RbacCACert, err))
	}
	pool, err := x509.SystemCertPool()
	if err != nil {
		pool = x509.NewCertPool()
	}
	if !pool.AppendCertsFromPEM(pem) {
		panic(fmt.Sprintf("no certificates found in RBAC CA certificate %s", utils.CoreCfg.RbacCACert))
	}
	return &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}
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

func isAccessGranted(c *gin.Context, bearer string) bool {
	client := makeClient(c.GetHeader("x-rh-identity"), bearer)
	access := rbac.AccessPagination{}
	res, err := client.Request(&base.Context, http.MethodGet, rbacURL, nil, &access)
	if res != nil && res.Body != nil {
		defer res.Body.Close()
	}

	if err != nil {
		utils.LogError("err", err, "Call to RBAC svc failed")
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
		workspaces, err := findInventoryWorkspaces(&access)
		if err != nil {
			utils.LogError("err", err, "RBAC")
			granted = false
		}
		c.Set(utils.KeyInventoryWorkspaces, workspaces)
	}
	return granted
}

func findInventoryWorkspaces(access *rbac.AccessPagination) ([]string, error) {
	if len(access.Data) == 0 {
		utils.LogDebug("rbac zero access data")
		return nil, nil
	}
	inventoryHostsReadPerms := expandedPermission(inventoryHostsReadPerm)
	workspaceIDs := make([]string, 0)
	for _, a := range access.Data {
		if !slices.Contains(inventoryHostsReadPerms, a.Permission) {
			continue
		}

		if len(a.ResourceDefinitions) == 0 {
			utils.LogDebug("rbac access to all workspaces")
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
					workspaceIDs = append(workspaceIDs, rootWorkspaceID)
					continue
				}
				id, err := uuid.Parse(*v)
				if err != nil {
					continue
				}
				workspaceIDs = append(workspaceIDs, id.String())
			}
		}
	}

	utils.LogDebug("workspace_count", len(workspaceIDs), "processed workspaces")
	return workspaceIDs, nil
}

func RBAC() gin.HandlerFunc {
	if !config.EnableRBACCHeck {
		return func(_ *gin.Context) {}
	}

	rbacURL = utils.FailIfEmpty(utils.CoreCfg.RbacAddress, "RBAC_ADDRESS (or Clowder v2 dependency rbac/service)") +
		base.RBACApiPrefix + "/access/?application=patch,inventory,content-sources"
	httpOnce.Do(func() {
		if utils.CoreCfg.RbacCACert != "" {
			httpClient = &http.Client{Transport: &http.Transport{TLSClientConfig: rbacTLSConfig()}}
		}
		httpClient = telemetry.InstrumentHTTPClient(httpClient)
	})

	return func(c *gin.Context) {
		var bearer string
		if utils.CoreCfg.RbacAuthenticated {
			var err error
			// fail closed: never call an authenticated RBAC endpoint without the bearer
			if bearer, err = getBearerToken(c); err != nil {
				utils.LogError("err", err, "Failed to get token for authenticated RBAC endpoint")
				serviceErrorCnt.WithLabelValues("rbac", strconv.Itoa(http.StatusBadGateway)).Inc()
				c.AbortWithStatusJSON(http.StatusBadGateway, utils.ErrorResponse{Error: "Communication with RBAC failed"})
				return
			}
		}
		if isAccessGranted(c, bearer) {
			return
		}
		c.AbortWithStatusJSON(http.StatusUnauthorized,
			utils.ErrorResponse{Error: "You don't have access to this application"})
	}
}
