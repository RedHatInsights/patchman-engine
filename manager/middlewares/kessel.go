package middlewares

import (
	"app/base/rbac"
	"app/base/utils"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/redhatinsights/platform-go-middlewares/identity"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/project-kessel/kessel-sdk-go/kessel/auth"
	kesselv2 "github.com/project-kessel/kessel-sdk-go/kessel/inventory/v1beta2"
)

var granularPermissions = map[string]string{
	"TemplateSystemsUpdateHandler": "patch_template_edit",
	"TemplateSystemsDeleteHandler": "patch_template_edit",
	"SystemDeleteHandler":          "patch_system_edit",
}

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
		return nil, errors.New("no workspaces were found")
	}
	return map[string]string{utils.KeyGrouped: fmt.Sprintf("{%s}", strings.Join(groups, ","))}, nil
}

func getToken(ctx context.Context) (string, error) {
	tokenReqCtx, tokenCtxCancel := context.WithCancel(ctx)
	defer tokenCtxCancel()
	res, err := credentials.GetToken(tokenReqCtx, auth.GetTokenOptions{})
	if err != nil {
		return "", err
	}
	return res.AccessToken, nil
}

func getDefaultWorkspaceID(ctx context.Context, xrhid *identity.XRHID) (string, error) {
	workspaceReqCtx, workspaceCtxCancel := context.WithCancel(ctx)
	defer workspaceCtxCancel()

	req, err := http.NewRequestWithContext(
		workspaceReqCtx, http.MethodGet, utils.CoreCfg.RbacURL+"/v2/workspaces/?type=default", nil)
	if err != nil {
		return "", errors.Wrap(err, "Failed to create a request for default workspaceID")
	}
	req.Header.Add("x-rh-rbac-org-id", xrhid.Identity.OrgID)

	if utils.CoreCfg.KesselAuthEnabled {
		token, err := getToken(workspaceReqCtx)
		if err != nil {
			return "", errors.Wrap(err, "Request for RBAC token failed")
		}
		req.Header.Add("authorization", fmt.Sprintf("Bearer %s", token))
	}

	httpRes, err := utils.CallAPI(&http.Client{}, req, log.IsLevelEnabled(log.TraceLevel))
	if err != nil {
		if httpRes != nil && httpRes.Body != nil {
			httpRes.Body.Close()
		}
		return "", errors.Wrap(err, "Request failed")
	}

	var res rbac.DefaultWorkspaceResponse
	err = sonic.ConfigDefault.NewDecoder(httpRes.Body).Decode(&res)
	if err != nil && err != io.EOF {
		return "", errors.Wrap(err, "Response body reading failed")
	}

	if len(res.Data) != 1 {
		return "", errors.New("RBAC returned an unexpected number of default workspaces")
	}

	return res.Data[0].ID, nil
}

func useCheckForUpdate(
	c *gin.Context, client kesselv2.KesselInventoryServiceClient, xrhid *identity.XRHID, permission string,
) error {
	checkReqCtx, checkCtxCancel := context.WithCancel(c)
	defer checkCtxCancel()

	workspaceID, err := getDefaultWorkspaceID(checkReqCtx, xrhid)
	if err != nil {
		return errors.Wrap(err, "could not get default workspaceID")
	}

	res, err := client.CheckForUpdate(checkReqCtx, &kesselv2.CheckForUpdateRequest{
		Object: &kesselv2.ResourceReference{
			ResourceType: "workspace",
			ResourceId:   workspaceID,
			Reporter: &kesselv2.ReporterReference{
				Type: "rbac",
			},
		},
		Relation: permission,
		Subject:  buildSubject(xrhid),
	})
	if err != nil {
		return errors.Wrap(err, "failed to communicate with Kessel")
	}

	if res.Allowed != kesselv2.Allowed_ALLOWED_TRUE {
		c.AbortWithStatusJSON(http.StatusUnauthorized, utils.ErrorResponse{
			Error: "Missing permission", // does not have granular permission
		})
	}
	return nil
}

func useStreamedListObjects(c *gin.Context, client kesselv2.KesselInventoryServiceClient, xrhid *identity.XRHID) error {
	sloReqContext, sloContextCancel := context.WithCancel(c)
	defer sloContextCancel()

	var permission string
	switch c.Request.Method {
	case http.MethodGet, http.MethodPost:
		permission = "patch_all_view"
	case http.MethodPut, http.MethodDelete:
		permission = "patch_all_edit"
	}

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
		return errors.Wrap(err, "failed to establish a gRPC stream with Kessel")
	}

	workspaces := make([]*kesselv2.StreamedListObjectsResponse, 0)
	for res, err := stream.Recv(); err != io.EOF; res, err = stream.Recv() {
		if err != nil {
			return errors.Wrap(err, "failed to receive all from Kessel")
		}
		workspaces = append(workspaces, res)
	}

	inventoryGroups, err := processWorkspaces(workspaces)
	if err != nil {
		utils.LogError("err", err.Error(), "processWorkspaces")
		c.AbortWithStatusJSON(http.StatusUnauthorized, utils.ErrorResponse{
			Error: "You don't have access to this application",
		})
	}
	c.Set(utils.KeyInventoryGroups, inventoryGroups)
	return nil
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
		c.AbortWithStatusJSON(http.StatusUnauthorized, utils.ErrorResponse{Error: "Invalid x-rh-identity header"})
	}

	// Require granular permission if set for handler
	nameSplit := strings.Split(c.HandlerName(), ".")
	handlerName := nameSplit[len(nameSplit)-1]
	if permission, has := granularPermissions[handlerName]; has {
		err = useCheckForUpdate(c, client, xrhid, permission)
		if err != nil {
			utils.LogError("err", err.Error(), "useCheckForUpdate failed")
			c.AbortWithStatus(http.StatusInternalServerError)
		}
		return
	}

	// Require method-derived permission otherwise
	err = useStreamedListObjects(c, client, xrhid)
	if err != nil {
		utils.LogError("err", err.Error(), "useStreamedListObjects failed")
		c.AbortWithStatus(http.StatusInternalServerError)
	}
}

func Kessel() gin.HandlerFunc {
	return hasPermissionKessel
}
