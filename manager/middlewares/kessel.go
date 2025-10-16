package middlewares

import (
	"app/base/utils"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/redhatinsights/platform-go-middlewares/identity"
	"google.golang.org/grpc"

	"github.com/project-kessel/kessel-sdk-go/kessel/auth"
	kesselv2 "github.com/project-kessel/kessel-sdk-go/kessel/inventory/v1beta2"
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

func buildSubject(xrhid *identity.XRHID) *kesselv2.SubjectReference {
	return &kesselv2.SubjectReference{
		Resource: &kesselv2.ResourceReference{
			ResourceType: "principal",
			ResourceId:   fmt.Sprintf("redhat/%s", xrhid.Identity.User.UserID),
			Reporter: &kesselv2.ReporterReference{
				Type: "rbac",
			},
		},
	}
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
) ([]*kesselv2.StreamedListObjectsResponse, time.Duration, error) {
	sloReqContext, sloContextCancel := context.WithCancel(c)
	defer sloContextCancel()

	resourceType := "rbac"
	stream, err := client.StreamedListObjects(sloReqContext, &kesselv2.StreamedListObjectsRequest{
		ObjectType: &kesselv2.RepresentationType{
			ResourceType: "workspace",
			ReporterType: &resourceType,
		},
		Relation: permission,
		Subject:  buildSubject(xrhid),
	})
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to establish a gRPC stream with Kessel")
	}

	workspaces := make([]*kesselv2.StreamedListObjectsResponse, 0)
	start := time.Now()
	for res, err := stream.Recv(); err != io.EOF; res, err = stream.Recv() {
		if err != nil {
			return nil, time.Since(start), errors.Wrap(err, "failed to receive all from Kessel")
		}
		workspaces = append(workspaces, res)
	}

	return workspaces, time.Since(start), nil
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
	defer conn.Close()

	xrhid, err := utils.ParseXRHID(c.GetHeader("x-rh-identity"))
	if err != nil {
		utils.LogError("err", err.Error(), "failed to ParseXRHID")
		c.AbortWithStatusJSON(http.StatusUnauthorized, utils.ErrorResponse{Error: "Invalid x-rh-identity header"})
		return
	}

	permission := buildPermission(c)
	workspaces, receivingDuration, err := useStreamedListObjects(c, client, xrhid, permission)
	if err != nil {
		utils.LogError(
			"err", err.Error(), "receivingDuration", receivingDuration, "permission", permission,
			"failed to useStreamedListObjects",
		)
		c.AbortWithStatusJSON(http.StatusInternalServerError, utils.ErrorResponse{
			Error: "Communication with RBAC failed",
		})
		return
	}
	utils.LogDebug("workspaces", workspaces, "receivingDuration", receivingDuration, "retrieved workspaces")

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
