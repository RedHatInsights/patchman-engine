package utils

import (
	"os"
	"testing"

	clowder "github.com/redhatinsights/app-common-go/pkg/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestResolveRbacV2Address(t *testing.T) {
	origV2 := clowder.DependencyEndpointsV2
	defer func() { clowder.DependencyEndpointsV2 = origV2 }()

	t.Run("V2 available with URI", func(t *testing.T) {
		clowder.DependencyEndpointsV2 = map[string]map[string]clowder.DependencyEndpointV2{
			"rbac": {
				"service": {Uri: "https://rbac-service.svc:8443", Authenticated: false},
			},
		}
		assert.Equal(t, "https://rbac-service.svc:8443", resolveRbacV2Address())
	})

	t.Run("V2 nil map", func(t *testing.T) {
		clowder.DependencyEndpointsV2 = nil
		assert.Equal(t, "", resolveRbacV2Address())
	})

	t.Run("V2 empty URI", func(t *testing.T) {
		clowder.DependencyEndpointsV2 = map[string]map[string]clowder.DependencyEndpointV2{
			"rbac": {
				"service": {Uri: "", Authenticated: false},
			},
		}
		assert.Equal(t, "", resolveRbacV2Address())
	})

	t.Run("V2 wrong app key", func(t *testing.T) {
		clowder.DependencyEndpointsV2 = map[string]map[string]clowder.DependencyEndpointV2{
			"other-app": {
				"service": {Uri: "https://other.svc:8443", Authenticated: false},
			},
		}
		assert.Equal(t, "", resolveRbacV2Address())
	})

	t.Run("V2 wrong deployment key", func(t *testing.T) {
		clowder.DependencyEndpointsV2 = map[string]map[string]clowder.DependencyEndpointV2{
			"rbac": {
				"wrong-deploy": {Uri: "https://rbac.svc:8443", Authenticated: false},
			},
		}
		assert.Equal(t, "", resolveRbacV2Address())
	})

	t.Run("V2 with CA certificate", func(t *testing.T) {
		caPath := "/tmp/ca.crt"
		clowder.DependencyEndpointsV2 = map[string]map[string]clowder.DependencyEndpointV2{
			"rbac": {
				"service": {
					Uri:           "https://rbac-service.svc:8443",
					Authenticated: false,
					CaCertificate: &caPath,
				},
			},
		}
		assert.Equal(t, "https://rbac-service.svc:8443", resolveRbacV2Address())
	})
}

func TestRbacAddressEnvOverride(t *testing.T) {
	origAddr := CoreCfg.RbacAddress
	defer func() { CoreCfg.RbacAddress = origAddr }()

	// Simulate V2-resolved address
	CoreCfg.RbacAddress = "https://rbac-service.svc:8443"

	// Env var override takes precedence (initServicesFromEnv behavior)
	os.Setenv("RBAC_ADDRESS", "http://localhost:8080")
	defer os.Unsetenv("RBAC_ADDRESS")
	initServicesFromEnv()
	assert.Equal(t, "http://localhost:8080", CoreCfg.RbacAddress)
}

func TestRbacAddressEnvPreservesV2WhenUnset(t *testing.T) {
	origAddr := CoreCfg.RbacAddress
	defer func() { CoreCfg.RbacAddress = origAddr }()

	// Simulate V2-resolved address
	CoreCfg.RbacAddress = "https://rbac-service.svc:8443"

	// No env override: Getenv returns current value as default
	os.Unsetenv("RBAC_ADDRESS")
	initServicesFromEnv()
	assert.Equal(t, "https://rbac-service.svc:8443", CoreCfg.RbacAddress)
}
