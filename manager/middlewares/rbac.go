package middlewares

import (
	"app/base"
	"app/base/api"
	"app/base/rbac"
	"app/base/telemetry"
	"app/base/utils"
	"app/manager/config"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/project-kessel/kessel-sdk-go/kessel/auth"
	log "github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
)

const rootWorkspaceID = "00000000-0000-0000-0000-999999999999"

const tokenTimeout = 10 * time.Second

var (
	httpClient    = &http.Client{}
	httpClientErr error
	httpOnce      sync.Once
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

func newRbacHTTPClient() (*http.Client, error) {
	client := httpClient
	if caPath := utils.CoreCfg.RbacCACertificate; caPath != "" {
		pem, err := os.ReadFile(caPath) // #nosec G304
		if err != nil {
			return nil, errors.Wrap(err, "reading RBAC CA certificate")
		}
		pool, err := x509.SystemCertPool()
		if err != nil {
			return nil, errors.Wrap(err, "loading system cert pool")
		}
		if !pool.AppendCertsFromPEM(pem) {
			return nil, errors.New("could not parse RBAC CA certificate")
		}
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.TLSClientConfig = &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}
		client = &http.Client{Transport: transport}
	}
	return telemetry.InstrumentHTTPClient(client), nil
}

// bearerHeader returns the service account token, never an empty one: an authenticated RBAC endpoint must not be
// called without it.
func bearerHeader(ctx context.Context) (string, error) {
	if err := requireAuthConfig(); err != nil {
		return "", err
	}
	token, err := credentials.GetToken(ctx, auth.GetTokenOptions{HttpClient: &http.Client{Timeout: tokenTimeout}})
	if err != nil {
		return "", errors.Wrap(err, "fetching RBAC bearer token")
	}
	return "Bearer " + token.AccessToken, nil
}

// Make RBAC client on demand, with specified identity
func makeClient(ctx context.Context, identity string) (*api.Client, error) {
	httpOnce.Do(func() {
		httpClient, httpClientErr = newRbacHTTPClient()
	})
	if httpClientErr != nil {
		return nil, httpClientErr
	}
	debugRequest := log.IsLevelEnabled(log.TraceLevel)

	headers := map[string]string{xRHIdentity: identity}
	if utils.CoreCfg.RbacAuthenticated {
		if !strings.HasPrefix(utils.CoreCfg.RbacAddress, "https://") {
			return nil, errors.New("authenticated RBAC endpoint must use https")
		}
		bearer, err := bearerHeader(ctx)
		if err != nil {
			return nil, err
		}
		headers["Authorization"] = bearer
	}

	client := api.Client{
		HTTPClient:     httpClient,
		Debug:          debugRequest,
		DefaultHeaders: headers,
	}
	return &client, nil
}

func rbacAccessURL() string {
	return utils.FailIfEmpty(utils.CoreCfg.RbacAddress, "RBAC_ADDRESS") + base.RBACApiPrefix +
		"/access/?application=patch,inventory,content-sources"
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
	access := rbac.AccessPagination{}
	client, err := makeClient(base.Context, c.GetHeader("x-rh-identity"))
	var res *http.Response
	if err == nil {
		res, err = client.Request(&base.Context, http.MethodGet, rbacAccessURL(), nil, &access)
	}
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

	return func(c *gin.Context) {
		if isAccessGranted(c) {
			return
		}
		c.AbortWithStatusJSON(http.StatusUnauthorized,
			utils.ErrorResponse{Error: "You don't have access to this application"})
	}
}
