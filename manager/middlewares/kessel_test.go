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

func TestUseStreamedListObjects(t *testing.T) {
	conn, _ := grpc.NewClient(utils.CoreCfg.KesselURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	defer conn.Close()

	client := kesselv2.NewKesselInventoryServiceClient(conn)
	c := &gin.Context{Request: &http.Request{Method: http.MethodGet}}
	err := useStreamedListObjects(c, client, &identity.XRHID{
		Identity: identity.Identity{User: identity.User{UserID: "12345"}},
	})
	if assert.NoError(t, err) {
		inventoryGroups, found := c.Get(utils.KeyInventoryGroups)
		assert.True(t, found)
		assert.NotEqual(t, "", inventoryGroups)
	}
}

func TestHasPermissionKessel(t *testing.T) {
	c := &gin.Context{Request: &http.Request{Header: map[string][]string{}, Method: http.MethodGet}}
	c.Request.Header.Set("x-rh-identity", "eyJlbnRpdGxlbWVudHMiOnsiaW5zaWdodHMiOnsiaXNfZW50aXRsZWQiOnRydWV9LCJjb3N0X21hbmFnZW1lbnQiOnsiaXNfZW50aXRsZWQiOnRydWV9LCJhbnNpYmxlIjp7ImlzX2VudGl0bGVkIjp0cnVlfSwib3BlbnNoaWZ0Ijp7ImlzX2VudGl0bGVkIjp0cnVlfSwic21hcnRfbWFuYWdlbWVudCI6eyJpc19lbnRpdGxlZCI6dHJ1ZX0sIm1pZ3JhdGlvbnMiOnsiaXNfZW50aXRsZWQiOnRydWV9fSwiaWRlbnRpdHkiOnsiaW50ZXJuYWwiOnsiYXV0aF90aW1lIjoyOTksImF1dGhfdHlwZSI6ImJhc2ljLWF1dGgiLCJvcmdfaWQiOiIxMTc4OTc3MiJ9LCJhY2NvdW50X251bWJlciI6IjYwODk3MTkiLCJ1c2VyIjp7ImZpcnN0X25hbWUiOiJJbnNpZ2h0cyIsImlzX2FjdGl2ZSI6dHJ1ZSwiaXNfaW50ZXJuYWwiOmZhbHNlLCJsYXN0X25hbWUiOiJRQSIsImxvY2FsZSI6ImVuX1VTIiwiaXNfb3JnX2FkbWluIjp0cnVlLCJ1c2VybmFtZSI6Imluc2lnaHRzLXFhIiwiZW1haWwiOiJqbmVlZGxlK3FhQHJlZGhhdC5jb20ifSwidHlwZSI6IlVzZXIifX0=") //nolint:lll

	hasPermissionKessel(c)
	_, exists := c.Get(utils.KeyInventoryGroups)
	assert.True(t, exists)
}
