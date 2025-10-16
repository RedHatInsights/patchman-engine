package middlewares

import (
	"app/base/utils"
	"fmt"
	"net/http"
	"strconv"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/gin-gonic/gin"
	kesselv2 "github.com/project-kessel/kessel-sdk-go/kessel/inventory/v1beta2"
	"github.com/redhatinsights/platform-go-middlewares/identity"
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

func TestProcessWorkspaces(t *testing.T) {
	expected := fmt.Sprintf("{%s,%s}", strconv.Quote(`[{"id":"test-1"}]`), strconv.Quote(`[{"id":"test-2"}]`))
	workspaces := []*kesselv2.StreamedListObjectsResponse{
		{Object: &kesselv2.ResourceReference{ResourceId: "test-1"}},
		{Object: &kesselv2.ResourceReference{ResourceId: "test-2"}},
	}
	processed, err := processWorkspaces(workspaces)
	if assert.NoError(t, err) {
		assert.Equal(t, expected, processed[utils.KeyGrouped])
	}
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
	workspaces, _, err := useStreamedListObjects(c, client, mockXRHID(), "demo_permission")
	if assert.NoError(t, err) {
		assert.Equal(t, 1, len(workspaces))
	}
}

func TestHasPermissionKessel(t *testing.T) {
	c := &gin.Context{Request: &http.Request{Header: map[string][]string{}, Method: http.MethodGet}}
	c.Request.Header.Set("x-rh-identity", "eyJlbnRpdGxlbWVudHMiOnsiaW5zaWdodHMiOnsiaXNfZW50aXRsZWQiOnRydWV9LCJjb3N0X21hbmFnZW1lbnQiOnsiaXNfZW50aXRsZWQiOnRydWV9LCJhbnNpYmxlIjp7ImlzX2VudGl0bGVkIjp0cnVlfSwib3BlbnNoaWZ0Ijp7ImlzX2VudGl0bGVkIjp0cnVlfSwic21hcnRfbWFuYWdlbWVudCI6eyJpc19lbnRpdGxlZCI6dHJ1ZX0sIm1pZ3JhdGlvbnMiOnsiaXNfZW50aXRsZWQiOnRydWV9fSwiaWRlbnRpdHkiOnsiaW50ZXJuYWwiOnsiYXV0aF90aW1lIjoyOTksImF1dGhfdHlwZSI6ImJhc2ljLWF1dGgiLCJvcmdfaWQiOiIxMTc4OTc3MiJ9LCJhY2NvdW50X251bWJlciI6IjYwODk3MTkiLCJ1c2VyIjp7ImZpcnN0X25hbWUiOiJJbnNpZ2h0cyIsImlzX2FjdGl2ZSI6dHJ1ZSwiaXNfaW50ZXJuYWwiOmZhbHNlLCJsYXN0X25hbWUiOiJRQSIsImxvY2FsZSI6ImVuX1VTIiwiaXNfb3JnX2FkbWluIjp0cnVlLCJ1c2VybmFtZSI6Imluc2lnaHRzLXFhIiwiZW1haWwiOiJqbmVlZGxlK3FhQHJlZGhhdC5jb20ifSwidHlwZSI6IlVzZXIifX0=") //nolint:lll

	hasPermissionKessel(c)
	inventoryGroups, found := c.Get(utils.KeyInventoryGroups)
	require.True(t, found)
	inventoryGroupMap, ok := (inventoryGroups).(map[string]string)
	require.True(t, ok)
	assert.Equal(t, `{"[{\"id\":\"inventory-group-1\"}]"}`, inventoryGroupMap[utils.KeyGrouped])
}

func mockXRHID() *identity.XRHID {
	return &identity.XRHID{
		Identity: identity.Identity{
			OrgID: "12345",
			User:  identity.User{UserID: "12345"},
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
