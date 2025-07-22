package middlewares

import (
	"app/base/utils"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	kesselAPIv2 "github.com/project-kessel/inventory-api/api/kessel/inventory/v1beta2"
	kesselClientCommon "github.com/project-kessel/inventory-client-go/common"
	kesselClientV2 "github.com/project-kessel/inventory-client-go/v1beta2"
)

type ListObjectStreamingClient = grpc.ServerStreamingClient[kesselAPIv2.StreamedListObjectsResponse]

func setupClient() (*kesselClientV2.InventoryClient, error) {
	// TODO: use secure credentials
	options := []func(*kesselClientCommon.Config){
		kesselClientCommon.WithgRPCUrl(utils.CoreCfg.KesselURL),
		kesselClientCommon.WithTLSInsecure(utils.CoreCfg.KesselInsecure),
	}
	return kesselClientV2.New(kesselClientCommon.NewConfig(options...))
}

func buildRequest(c *gin.Context) (*kesselAPIv2.StreamedListObjectsRequest, error) {
	xrhid, err := utils.ParseXRHID(c.GetHeader("x-rh-identity"))
	if err != nil {
		return nil, err
	}

	reporterType := "rbac"
	req := &kesselAPIv2.StreamedListObjectsRequest{
		ObjectType: &kesselAPIv2.RepresentationType{
			ResourceType: "workspace",
			ReporterType: &reporterType,
		},
		Subject: &kesselAPIv2.SubjectReference{
			Resource: &kesselAPIv2.ResourceReference{
				ResourceType: "principal",
				ResourceId:   fmt.Sprintf("redhat/%s", xrhid.Identity.User.UserID),
				Reporter: &kesselAPIv2.ReporterReference{
					Type: "rbac",
				},
			},
		},
	}

	switch c.Request.Method {
	case http.MethodGet, http.MethodPost:
		req.Relation = patchReadPerm
	case http.MethodPut, http.MethodDelete:
		req.Relation = patchWritePerm
	}

	return req, nil
}

func receiveAll(stream ListObjectStreamingClient) ([]*kesselAPIv2.StreamedListObjectsResponse, error) {
	responses := make([]*kesselAPIv2.StreamedListObjectsResponse, 0)
	for res, err := stream.Recv(); err != io.EOF; res, err = stream.Recv() {
		if err != nil {
			return nil, err
		}
		responses = append(responses, res)
	}
	return responses, nil
}

func processWorkspaces(workspaces []*kesselAPIv2.StreamedListObjectsResponse) (map[string]string, error) {
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

func hasPermissionKessel(c *gin.Context) {
	client, err := setupClient()
	if err != nil {
		utils.LogError("err", err.Error(), "failed to setup a Kessel service client")
		c.AbortWithStatusJSON(http.StatusInternalServerError, utils.ErrorResponse{
			Error: "Unexpecred server error", // missing cert or failed to make a new gRPC client
		})
		return
	}

	req, err := buildRequest(c)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, utils.ErrorResponse{Error: "Invalid x-rh-identity header"})
		return
	}

	// TODO: how long should the timeout be
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	stream, err := client.KesselInventoryService.StreamedListObjects(ctx, req)
	if err != nil {
		utils.LogError("err", err.Error(), "failed to establish a gRPC stream with Kessel")
		c.AbortWithStatusJSON(http.StatusInternalServerError, utils.ErrorResponse{
			Error: "Failed to communicate with Kessel RBAC service",
		})
		return
	}

	workspaces, err := receiveAll(stream)
	if err != nil {
		utils.LogError("err", err.Error(), "failed to receive from Kessel")
		c.AbortWithStatusJSON(http.StatusInternalServerError, utils.ErrorResponse{
			Error: "Failed to communicate with Kessel RBAC service",
		})
		return
	}

	inventoryGroups, err := processWorkspaces(workspaces)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, utils.ErrorResponse{
			Error: "You don't have access to this application",
		})
		return
	}

	c.Set(utils.KeyInventoryGroups, inventoryGroups)
	utils.LogDebug("Kessel check successful")
}

func Kessel() gin.HandlerFunc {
	return hasPermissionKessel
}
