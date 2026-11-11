package utils

import (
	"os"
	"testing"

	clowder "github.com/redhatinsights/app-common-go/pkg/api/v1"
	"github.com/stretchr/testify/assert"
)

const testRbacURI = "https://rbac-service.svc:8443"
const testRbacApp = "rbac"

func TestInitRbacFromClowder(t *testing.T) {
	origV2 := clowder.DependencyEndpointsV2
	origCfg := CoreCfg
	defer func() {
		clowder.DependencyEndpointsV2 = origV2
		CoreCfg = origCfg
	}()

	run := func(v2 map[string]map[string]clowder.DependencyEndpointV2) {
		CoreCfg.RbacAddress, CoreCfg.RbacAuthenticated, CoreCfg.RbacCACert = "", false, ""
		clowder.DependencyEndpointsV2 = v2
		initRbacFromClowder()
	}

	t.Run("V2 unauthenticated", func(t *testing.T) {
		run(map[string]map[string]clowder.DependencyEndpointV2{
			testRbacApp: {"service": {Uri: testRbacURI}},
		})
		assert.Equal(t, testRbacURI, CoreCfg.RbacAddress)
		assert.False(t, CoreCfg.RbacAuthenticated)
		assert.Equal(t, "", CoreCfg.RbacCACert)
	})

	t.Run("V2 authenticated with CA certificate", func(t *testing.T) {
		caPath := "/tmp/ca.crt"
		run(map[string]map[string]clowder.DependencyEndpointV2{
			testRbacApp: {"service": {Uri: "https://ewgw.example:443", Authenticated: true, CaCertificate: &caPath}},
		})
		assert.Equal(t, "https://ewgw.example:443", CoreCfg.RbacAddress)
		assert.True(t, CoreCfg.RbacAuthenticated)
		assert.Equal(t, caPath, CoreCfg.RbacCACert)
	})

	missing := map[string]map[string]map[string]clowder.DependencyEndpointV2{
		"nil map":        nil,
		"empty URI":      {testRbacApp: {"service": {Uri: ""}}},
		"wrong app":      {"other-app": {"service": {Uri: "https://other.svc:8443"}}},
		"wrong endpoint": {testRbacApp: {"wrong-deploy": {Uri: "https://rbac.svc:8443"}}},
	}
	for name, v2 := range missing {
		t.Run("V2 missing: "+name, func(t *testing.T) {
			run(v2)
			assert.Equal(t, "", CoreCfg.RbacAddress)
			assert.False(t, CoreCfg.RbacAuthenticated)
		})
	}
}

func TestRbacAddressEnvOverride(t *testing.T) {
	origAddr := CoreCfg.RbacAddress
	defer func() { CoreCfg.RbacAddress = origAddr }()

	// Simulate V2-resolved address
	CoreCfg.RbacAddress = testRbacURI

	// Env var override takes precedence (initServicesFromEnv behavior)
	t.Setenv("RBAC_ADDRESS", "http://localhost:8080")
	initServicesFromEnv()
	assert.Equal(t, "http://localhost:8080", CoreCfg.RbacAddress)
}

func TestRbacAddressEnvPreservesV2WhenUnset(t *testing.T) {
	origAddr := CoreCfg.RbacAddress
	defer func() { CoreCfg.RbacAddress = origAddr }()

	// Simulate V2-resolved address
	CoreCfg.RbacAddress = testRbacURI

	// No env override: Getenv returns current value as default.
	// t.Setenv registers cleanup to restore original value after test;
	// os.Unsetenv then clears it so LookupEnv returns ok=false.
	t.Setenv("RBAC_ADDRESS", "")
	os.Unsetenv("RBAC_ADDRESS")
	initServicesFromEnv()
	assert.Equal(t, testRbacURI, CoreCfg.RbacAddress)
}
