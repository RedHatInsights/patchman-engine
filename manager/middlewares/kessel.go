package middlewares

import (
	"app/base/utils"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/redhatinsights/platform-go-middlewares/identity"
	"google.golang.org/grpc"

	"github.com/project-kessel/kessel-sdk-go/kessel/auth"
	kesselv2 "github.com/project-kessel/kessel-sdk-go/kessel/inventory/v1beta2"
	kesselRbacV2 "github.com/project-kessel/kessel-sdk-go/kessel/rbac/v2"
)

var credentials = auth.NewOAuth2ClientCredentials(
	utils.CoreCfg.KesselAuthClientID,
	utils.CoreCfg.KesselAuthClientSecret,
	utils.CoreCfg.KesselAuthOIDCIssuer,
)

func setupClient() (kesselv2.KesselInventoryServiceClient, *grpc.ClientConn, error) {
	clientBuilder := kesselv2.NewClientBuilder(utils.CoreCfg.KesselURL)
	if utils.CoreCfg.KesselAuthEnabled {
		clientBuilder = clientBuilder.OAuth2ClientAuthenticated(&credentials, nil)
	}
	if utils.CoreCfg.KesselInsecure { // insecure TLS
		clientBuilder = clientBuilder.Insecure()
	}
	return clientBuilder.Build()
}

func processWorkspaces(workspaces []*kesselv2.StreamedListObjectsResponse) (map[string]string, error) {
	groups := make([]string, 0, len(workspaces))
	for _, workspace := range workspaces {
		group, err := utils.ParseInventoryGroup(&workspace.Object.ResourceId, nil)
		if err != nil {
			// couldn't marshal inventory group to json
			continue
		}
		groups = append(groups, group)
	}

	if len(groups) == 0 {
		return nil, errors.New("no workspaces found")
	}
	return map[string]string{utils.KeyGrouped: fmt.Sprintf("{%s}", strings.Join(groups, ","))}, nil
}

func buildPermission(c *gin.Context) string {
	permission := "patch_system_"
	nameSplit := strings.Split(c.HandlerName(), ".")
	handlerName := nameSplit[len(nameSplit)-1]
	if strings.HasPrefix(handlerName, "Template") {
		permission = "patch_template_"
	}

	switch c.Request.Method {
	case http.MethodGet, http.MethodPost:
		permission += "view"
	case http.MethodPatch, http.MethodPut, http.MethodDelete:
		permission += "edit"
	}

	return permission
}

func useStreamedListObjects(
	c *gin.Context, client kesselv2.KesselInventoryServiceClient, xrhid *identity.XRHID, permission string,
) ([]*kesselv2.StreamedListObjectsResponse, error) {
	sloReqContext, sloContextCancel := context.WithCancel(c)
	defer sloContextCancel()

	workspaces := make([]*kesselv2.StreamedListObjectsResponse, 0)
	start := time.Now()
	for res, err := range kesselRbacV2.ListWorkspaces(
		sloReqContext, client, kesselRbacV2.PrincipalSubject(xrhid.Identity.User.UserID, "redhat"), permission, "",
	) {
		if err != nil {
			utils.LogError(
				"err", err.Error(), "receivingDuration", time.Since(start), "permission", permission, "received_count",
				len(workspaces), "failed to useStreamedListObjects",
			)
			return nil, err
		}
		workspaces = append(workspaces, res)
	}

	utils.LogDebug(
		"workspaces", workspaces, "receivingDuration", time.Since(start), "permission", permission, "received_count",
		len(workspaces), "retrieved workspaces",
	)
	return workspaces, nil
}

func hasPermissionKessel(c *gin.Context) {
	client, conn, err := setupClient()
	if err != nil {
		utils.LogError("err", err.Error(), "failed to setup Kessel service client")
		c.AbortWithStatusJSON(http.StatusInternalServerError, utils.ErrorResponse{
			Error: "Unexpected server error", // missing cert or failed to make a new gRPC client
		})
		return
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			utils.LogError("err", closeErr.Error(), "failed to close gRPC client")
		}
	}()

	xrhid, err := utils.ParseXRHID(c.GetHeader("x-rh-identity"))
	if err != nil {
		utils.LogError("err", err.Error(), "failed to ParseXRHID")
		c.AbortWithStatusJSON(http.StatusUnauthorized, utils.ErrorResponse{Error: "Invalid x-rh-identity header"})
		return
	}

	workspaces, err := useStreamedListObjects(c, client, xrhid, buildPermission(c))
	if err != nil {
		// already logged in useStreamedListObjects
		c.AbortWithStatusJSON(http.StatusInternalServerError, utils.ErrorResponse{
			Error: "Communication with RBAC failed",
		})
		return
	}

	inventoryGroups, err := processWorkspaces(workspaces)
	if err != nil {
		utils.LogWarn(err.Error())
		c.AbortWithStatusJSON(http.StatusUnauthorized, utils.ErrorResponse{Error: "Missing permission"})
		return
	}
	c.Set(utils.KeyInventoryGroups, inventoryGroups)
	utils.LogDebug("Kessel check OK")
}

func Kessel() gin.HandlerFunc {
	return hasPermissionKessel
}
