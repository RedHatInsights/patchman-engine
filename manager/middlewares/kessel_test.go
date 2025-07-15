package middlewares

import (
	"app/base/utils"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	kesselAPIv2 "github.com/project-kessel/inventory-api/api/kessel/inventory/v1beta2"
	kesselClientCommon "github.com/project-kessel/inventory-client-go/common"
	kesselClientV2 "github.com/project-kessel/inventory-client-go/v1beta2"
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
	client, err := setupClient()
	assert.NoError(t, err)
	assert.NotNil(t, client)

	// secure TLS and no auth
	utils.CoreCfg.KesselInsecure = false
	client, err = setupClient()
	assert.NoError(t, err)
	assert.NotNil(t, client)

	// insecure TLS and auth
	utils.CoreCfg.KesselInsecure = true
	utils.CoreCfg.KesselAuthEnabled = true
	utils.CoreCfg.KesselAuthClientID = "test-client-id"
	utils.CoreCfg.KesselAuthClientSecret = "test-client-secret"
	client, err = setupClient()
	assert.NoError(t, err)
	assert.NotNil(t, client)

	// secure TLS and auth
	utils.CoreCfg.KesselInsecure = false
	client, err = setupClient()
	assert.NoError(t, err)
	assert.NotNil(t, client)

	// cleanup
	utils.CoreCfg.KesselInsecure = originalKesselInsecure
	utils.CoreCfg.KesselAuthEnabled = originalKesselAuthEnabled
	utils.CoreCfg.KesselAuthClientID = originalKesselAuthClientID
	utils.CoreCfg.KesselAuthClientSecret = originalKesselAuthClientSecret
}

func TestBuildRequest(t *testing.T) {
	c := &gin.Context{Request: &http.Request{Header: map[string][]string{}}}
	_, err := buildRequest(c)
	assert.Error(t, err)

	c = mockContext()
	req, err := buildRequest(c)
	if assert.NoError(t, err) {
		assert.Equal(t, patchReadPerm, req.Relation)
	}

	c.Request.Method = http.MethodDelete
	req, err = buildRequest(c)
	if assert.NoError(t, err) {
		assert.Equal(t, patchWritePerm, req.Relation)
	}
}

func TestReceiveAll(t *testing.T) {
	options := []func(*kesselClientCommon.Config){
		kesselClientCommon.WithgRPCUrl(utils.CoreCfg.KesselURL),
		kesselClientCommon.WithTLSInsecure(true),
	}
	client, _ := kesselClientV2.New(kesselClientCommon.NewConfig(options...))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	stream, err := client.KesselInventoryService.StreamedListObjects(ctx, nil)
	if assert.NoError(t, err) {
		workspaces, err := receiveAll(stream)
		if assert.NoError(t, err) {
			assert.Equal(t, "inventory-group-1", workspaces[0].Object.ResourceId)
		}
	}
}

func TestProcessWorkspaces(t *testing.T) {
	expected := fmt.Sprintf("{%s,%s}", strconv.Quote(`[{"id":"test-1"}]`), strconv.Quote(`[{"id":"test-2"}]`))
	workspaces := []*kesselAPIv2.StreamedListObjectsResponse{
		{Object: &kesselAPIv2.ResourceReference{ResourceId: "test-1"}},
		{Object: &kesselAPIv2.ResourceReference{ResourceId: "test-2"}},
	}
	processed, err := processWorkspaces(workspaces)
	if assert.NoError(t, err) {
		assert.Equal(t, expected, processed[utils.KeyGrouped])
	}
}

func TestHasPermissionKessel(t *testing.T) {
	c := mockContext()
	hasPermissionKessel(c)
	_, exists := c.Get(utils.KeyInventoryGroups)
	assert.True(t, exists)
}

func mockContext() *gin.Context {
	c := &gin.Context{Request: &http.Request{Header: map[string][]string{}}}
	c.Request.Header.Set("x-rh-identity", "eyJlbnRpdGxlbWVudHMiOnsiaW5zaWdodHMiOnsiaXNfZW50aXRsZWQiOnRydWV9LCJjb3N0X21hbmFnZW1lbnQiOnsiaXNfZW50aXRsZWQiOnRydWV9LCJhbnNpYmxlIjp7ImlzX2VudGl0bGVkIjp0cnVlfSwib3BlbnNoaWZ0Ijp7ImlzX2VudGl0bGVkIjp0cnVlfSwic21hcnRfbWFuYWdlbWVudCI6eyJpc19lbnRpdGxlZCI6dHJ1ZX0sIm1pZ3JhdGlvbnMiOnsiaXNfZW50aXRsZWQiOnRydWV9fSwiaWRlbnRpdHkiOnsiaW50ZXJuYWwiOnsiYXV0aF90aW1lIjoyOTksImF1dGhfdHlwZSI6ImJhc2ljLWF1dGgiLCJvcmdfaWQiOiIxMTc4OTc3MiJ9LCJhY2NvdW50X251bWJlciI6IjYwODk3MTkiLCJ1c2VyIjp7ImZpcnN0X25hbWUiOiJJbnNpZ2h0cyIsImlzX2FjdGl2ZSI6dHJ1ZSwiaXNfaW50ZXJuYWwiOmZhbHNlLCJsYXN0X25hbWUiOiJRQSIsImxvY2FsZSI6ImVuX1VTIiwiaXNfb3JnX2FkbWluIjp0cnVlLCJ1c2VybmFtZSI6Imluc2lnaHRzLXFhIiwiZW1haWwiOiJqbmVlZGxlK3FhQHJlZGhhdC5jb20ifSwidHlwZSI6IlVzZXIifX0=") //nolint:lll
	c.Request.Method = http.MethodGet
	return c
}
