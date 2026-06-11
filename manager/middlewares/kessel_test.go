package middlewares

import (
	"app/base/utils"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/gin-gonic/gin"
	kesselv2 "github.com/project-kessel/kessel-sdk-go/kessel/inventory/v1beta2"
	"github.com/redhatinsights/platform-go-middlewares/v2/identity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testXRHIdentity = "ewogICAgImVudGl0bGVtZW50cyI6IHsKICAgICAgICAiaW5zaWdodHMiOiB7CiAgICAgICAgICAgICJpc19lbnRpdGxlZCI6IHRydWUKICAgICAgICB9LAogICAgICAgICJjb3N0X21hbmFnZW1lbnQiOiB7CiAgICAgICAgICAgICJpc19lbnRpdGxlZCI6IHRydWUKICAgICAgICB9LAogICAgICAgICJhbnNpYmxlIjogewogICAgICAgICAgICAiaXNfZW50aXRsZWQiOiB0cnVlCiAgICAgICAgfSwKICAgICAgICAib3BlbnNoaWZ0IjogewogICAgICAgICAgICAiaXNfZW50aXRsZWQiOiB0cnVlCiAgICAgICAgfSwKICAgICAgICAic21hcnRfbWFuYWdlbWVudCI6IHsKICAgICAgICAgICAgImlzX2VudGl0bGVkIjogdHJ1ZQogICAgICAgIH0sCiAgICAgICAgIm1pZ3JhdGlvbnMiOiB7CiAgICAgICAgICAgICJpc19lbnRpdGxlZCI6IHRydWUKICAgICAgICB9CiAgICB9LAogICAgImlkZW50aXR5IjogewogICAgICAgICJpbnRlcm5hbCI6IHsKICAgICAgICAgICAgImF1dGhfdGltZSI6IDI5OSwKICAgICAgICAgICAgImF1dGhfdHlwZSI6ICJiYXNpYy1hdXRoIiwKICAgICAgICAgICAgIm9yZ19pZCI6ICIxMTc4OTc3MiIKICAgICAgICB9LAogICAgICAgICJhY2NvdW50X251bWJlciI6ICI2MDg5NzE5IiwKICAgICAgICAidXNlciI6IHsKICAgICAgICAgICAgImZpcnN0X25hbWUiOiAiSW5zaWdodHMiLAogICAgICAgICAgICAiaXNfYWN0aXZlIjogdHJ1ZSwKICAgICAgICAgICAgImlzX2ludGVybmFsIjogZmFsc2UsCiAgICAgICAgICAgICJsYXN0X25hbWUiOiAiUUEiLAogICAgICAgICAgICAibG9jYWxlIjogImVuX1VTIiwKICAgICAgICAgICAgImlzX29yZ19hZG1pbiI6IHRydWUsCiAgICAgICAgICAgICJ1c2VybmFtZSI6ICJpbnNpZ2h0cy1xYSIsCiAgICAgICAgICAgICJlbWFpbCI6ICJqbmVlZGxlK3FhQHJlZGhhdC5jb20iLAogICAgICAgICAgICAidXNlcl9pZCI6ICI2MDg5NzE5IgogICAgICAgIH0sCiAgICAgICAgInR5cGUiOiAiVXNlciIKICAgIH0KfQ==" //nolint:lll

type testKesselServer struct {
	kesselv2.UnimplementedKesselInventoryServiceServer
	workspaceIDs []string
}

func (s *testKesselServer) StreamedListObjects(_ *kesselv2.StreamedListObjectsRequest,
	stream kesselv2.KesselInventoryService_StreamedListObjectsServer,
) error {
	for _, id := range s.workspaceIDs {
		if err := stream.Send(&kesselv2.StreamedListObjectsResponse{
			Object: &kesselv2.ResourceReference{
				ResourceType: "workspace",
				ResourceId:   id,
			},
		}); err != nil {
			return err
		}
	}
	return nil
}

func withTestKesselServer(t *testing.T, workspaceIDs []string) func() {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	grpcServer := grpc.NewServer()
	kesselv2.RegisterKesselInventoryServiceServer(grpcServer, &testKesselServer{workspaceIDs: workspaceIDs})
	go func() {
		_ = grpcServer.Serve(listener)
	}()

	originalURL := utils.CoreCfg.KesselURL
	originalInsecure := utils.CoreCfg.KesselInsecure
	utils.CoreCfg.KesselURL = listener.Addr().String()
	utils.CoreCfg.KesselInsecure = true

	return func() {
		grpcServer.Stop()
		_ = listener.Close()
		utils.CoreCfg.KesselURL = originalURL
		utils.CoreCfg.KesselInsecure = originalInsecure
	}
}

func newKesselTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.Header.Set("x-rh-identity", testXRHIdentity)
	return c, w
}

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
	c, _ := newKesselTestContext()
	hasPermissionKessel(c)
	workspaces, found := c.Get(utils.KeyInventoryWorkspaces)
	require.True(t, found)
	workspaceIDs, ok := (workspaces).([]string)
	require.True(t, ok)
	require.Greater(t, len(workspaceIDs), 0)
	assert.Equal(t, "aaaaaaaa-0000-0000-0000-000000000001", workspaceIDs[0])
}

func TestHasPermissionKesselMultipleWorkspacesSetsContext(t *testing.T) {
	defer withTestKesselServer(t, []string{
		"aaaaaaaa-0000-0000-0000-000000000001",
		"bbbbbbbb-0000-0000-0000-000000000002",
	})()

	c, w := newKesselTestContext()
	hasPermissionKessel(c)

	assert.False(t, c.IsAborted())
	assert.Equal(t, http.StatusOK, w.Code)

	workspaces, found := c.Get(utils.KeyInventoryWorkspaces)
	require.True(t, found)
	workspaceIDs, ok := workspaces.([]string)
	require.True(t, ok)
	assert.Equal(t, []string{
		"aaaaaaaa-0000-0000-0000-000000000001",
		"bbbbbbbb-0000-0000-0000-000000000002",
	}, workspaceIDs)
}

func TestHasPermissionKesselNoWorkspacesReturnsUnauthorized(t *testing.T) {
	defer withTestKesselServer(t, nil)()

	c, w := newKesselTestContext()
	hasPermissionKessel(c)

	assert.True(t, c.IsAborted())
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	_, found := c.Get(utils.KeyInventoryWorkspaces)
	assert.False(t, found)
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
