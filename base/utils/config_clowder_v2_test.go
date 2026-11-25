package utils

import (
	"testing"

	clowder "github.com/redhatinsights/app-common-go/pkg/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestRbacFromClowderV2(t *testing.T) {
	origV2 := clowder.DependencyEndpointsV2
	origLoaded := clowder.LoadedConfig
	origCfg := CoreCfg
	if clowder.LoadedConfig == nil {
		clowder.LoadedConfig = &clowder.AppConfig{}
	}
	defer func() {
		clowder.DependencyEndpointsV2 = origV2
		clowder.LoadedConfig = origLoaded
		CoreCfg = origCfg
	}()

	caPath := "/tmp/ca.crt"
	clowder.DependencyEndpointsV2 = map[string]map[string]clowder.DependencyEndpointV2{
		"rbac": {"service": {Uri: "https://ewgw:443", Authenticated: true, CaCertificate: &caPath}},
	}
	CoreCfg.RbacAddress, CoreCfg.RbacAuthenticated, CoreCfg.RbacCACertificate = "", false, ""
	initServicesFromClowder()
	assert.Equal(t, "https://ewgw:443", CoreCfg.RbacAddress)
	assert.True(t, CoreCfg.RbacAuthenticated)
	assert.Equal(t, caPath, CoreCfg.RbacCACertificate)

	clowder.DependencyEndpointsV2 = map[string]map[string]clowder.DependencyEndpointV2{
		"rbac": {"service": {Uri: "http://rbac-service.svc:8000", Authenticated: false}},
	}
	CoreCfg.RbacAddress, CoreCfg.RbacAuthenticated, CoreCfg.RbacCACertificate = "", true, caPath
	initServicesFromClowder()
	assert.Equal(t, "http://rbac-service.svc:8000", CoreCfg.RbacAddress)
	assert.False(t, CoreCfg.RbacAuthenticated)
}

func TestRbacAddressEnvOverride(t *testing.T) {
	origAddr := CoreCfg.RbacAddress
	defer func() { CoreCfg.RbacAddress = origAddr }()

	t.Setenv("RBAC_ADDRESS", "http://localhost:8080")
	initServicesFromEnv()
	assert.Equal(t, "http://localhost:8080", CoreCfg.RbacAddress)
}
