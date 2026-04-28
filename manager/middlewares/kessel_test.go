package middlewares

import (
	"app/base/utils"
	"net/http"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/gin-gonic/gin"
	kesselv2 "github.com/project-kessel/kessel-sdk-go/kessel/inventory/v1beta2"
	"github.com/redhatinsights/platform-go-middlewares/v2/identity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupClient(t *testing.T) {
	originalKesselInsecure := utils.CoreCfg.KesselInsecure
	originalKesselAuthEnabled := utils.CoreCfg.KesselAuthEnabled
	originalKesselAuthClientID := utils.CoreCfg.KesselAuthClientID
	originalKesselAuthClientSecret := utils.CoreCfg.KesselAuthClientSecret

	// insecure TLS and no auth
	utils.CoreCfg.KesselInsecure = true
	utils.CoreCfg.KesselAuthEnabled = false
	client, conn, err := setupClient()
	defer conn.Close()
	if assert.NoError(t, err) {
		assert.NotNil(t, client)
	}

	// secure TLS and no auth
	utils.CoreCfg.KesselInsecure = false
	client, conn, err = setupClient()
	defer conn.Close()
	if assert.NoError(t, err) {
		assert.NotNil(t, client)
	}

	// insecure TLS and auth
	utils.CoreCfg.KesselInsecure = true
	utils.CoreCfg.KesselAuthEnabled = true
	utils.CoreCfg.KesselAuthClientID = "test-client-id"
	utils.CoreCfg.KesselAuthClientSecret = "test-client-secret"
	client, conn, err = setupClient()
	defer conn.Close()
	if assert.NoError(t, err) {
		assert.NotNil(t, client)
	}

	// secure TLS and auth
	utils.CoreCfg.KesselInsecure = false
	client, conn, err = setupClient()
	defer conn.Close()
	if assert.NoError(t, err) {
		assert.NotNil(t, client)
	}

	// cleanup
	utils.CoreCfg.KesselInsecure = originalKesselInsecure
	utils.CoreCfg.KesselAuthEnabled = originalKesselAuthEnabled
	utils.CoreCfg.KesselAuthClientID = originalKesselAuthClientID
	utils.CoreCfg.KesselAuthClientSecret = originalKesselAuthClientSecret
}

func TestBuildPermission(t *testing.T) {
	c := &gin.Context{Request: &http.Request{Method: http.MethodGet}}
	permission := buildPermission(c)
	assert.Equal(t, "patch_system_view", permission)

	c = &gin.Context{Request: &http.Request{Method: http.MethodPut}}
	permission = buildPermission(c)
	assert.Equal(t, "patch_system_edit", permission)
}

func TestUseStreamedListObjects(t *testing.T) {
	client, conn := mockClient(t)
	defer conn.Close()

	c := &gin.Context{Request: &http.Request{Method: http.MethodGet}}
	workspaces, err := useStreamedListObjects(c, client, mockXRHID("user"), "demo_permission")
	if assert.NoError(t, err) {
		assert.Equal(t, 1, len(workspaces))
	}

	workspaces, err = useStreamedListObjects(c, client, mockXRHID("service_account"), "demo_permission")
	if assert.NoError(t, err) {
		assert.Equal(t, 1, len(workspaces))
	}
}

func TestHasPermissionKessel(t *testing.T) {
	c := &gin.Context{Request: &http.Request{Header: map[string][]string{}, Method: http.MethodGet}}
	c.Request.Header.Set("x-rh-identity", "ewogICAgImVudGl0bGVtZW50cyI6IHsKICAgICAgICAiaW5zaWdodHMiOiB7CiAgICAgICAgICAgICJpc19lbnRpdGxlZCI6IHRydWUKICAgICAgICB9LAogICAgICAgICJjb3N0X21hbmFnZW1lbnQiOiB7CiAgICAgICAgICAgICJpc19lbnRpdGxlZCI6IHRydWUKICAgICAgICB9LAogICAgICAgICJhbnNpYmxlIjogewogICAgICAgICAgICAiaXNfZW50aXRsZWQiOiB0cnVlCiAgICAgICAgfSwKICAgICAgICAib3BlbnNoaWZ0IjogewogICAgICAgICAgICAiaXNfZW50aXRsZWQiOiB0cnVlCiAgICAgICAgfSwKICAgICAgICAic21hcnRfbWFuYWdlbWVudCI6IHsKICAgICAgICAgICAgImlzX2VudGl0bGVkIjogdHJ1ZQogICAgICAgIH0sCiAgICAgICAgIm1pZ3JhdGlvbnMiOiB7CiAgICAgICAgICAgICJpc19lbnRpdGxlZCI6IHRydWUKICAgICAgICB9CiAgICB9LAogICAgImlkZW50aXR5IjogewogICAgICAgICJpbnRlcm5hbCI6IHsKICAgICAgICAgICAgImF1dGhfdGltZSI6IDI5OSwKICAgICAgICAgICAgImF1dGhfdHlwZSI6ICJiYXNpYy1hdXRoIiwKICAgICAgICAgICAgIm9yZ19pZCI6ICIxMTc4OTc3MiIKICAgICAgICB9LAogICAgICAgICJhY2NvdW50X251bWJlciI6ICI2MDg5NzE5IiwKICAgICAgICAidXNlciI6IHsKICAgICAgICAgICAgImZpcnN0X25hbWUiOiAiSW5zaWdodHMiLAogICAgICAgICAgICAiaXNfYWN0aXZlIjogdHJ1ZSwKICAgICAgICAgICAgImlzX2ludGVybmFsIjogZmFsc2UsCiAgICAgICAgICAgICJsYXN0X25hbWUiOiAiUUEiLAogICAgICAgICAgICAibG9jYWxlIjogImVuX1VTIiwKICAgICAgICAgICAgImlzX29yZ19hZG1pbiI6IHRydWUsCiAgICAgICAgICAgICJ1c2VybmFtZSI6ICJpbnNpZ2h0cy1xYSIsCiAgICAgICAgICAgICJlbWFpbCI6ICJqbmVlZGxlK3FhQHJlZGhhdC5jb20iLAogICAgICAgICAgICAidXNlcl9pZCI6ICI2MDg5NzE5IgogICAgICAgIH0sCiAgICAgICAgInR5cGUiOiAiVXNlciIKICAgIH0KfQ==") //nolint:lll

	hasPermissionKessel(c)
	workspaces, found := c.Get(utils.KeyInventoryWorkspaces)
	require.True(t, found)
	workspaceIDs, ok := (workspaces).([]string)
	require.True(t, ok)
	require.Greater(t, len(workspaceIDs), 0)
	assert.Equal(t, "inventory-group-1", workspaceIDs[0])
}

func mockXRHID(userType string) *identity.XRHID {
	if userType == "service_account" {
		return &identity.XRHID{
			Identity: identity.Identity{
				OrgID:          "12345",
				ServiceAccount: &identity.ServiceAccount{UserId: "12345"},
			},
		}
	}
	return &identity.XRHID{
		Identity: identity.Identity{
			OrgID: "12345",
			User:  &identity.User{UserID: "12345"},
		},
	}
}

func mockClient(t *testing.T) (kesselv2.KesselInventoryServiceClient, *grpc.ClientConn) {
	conn, err := grpc.NewClient(utils.CoreCfg.KesselURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fail()
	}
	return kesselv2.NewKesselInventoryServiceClient(conn), conn
}
