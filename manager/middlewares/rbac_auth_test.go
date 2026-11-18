package middlewares

import (
	"app/base/utils"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withRbacServer points the RBAC middleware at a fake RBAC that grants full access and records request headers.
func withRbacServer(t *testing.T, authenticated bool, issuer string) (*http.Header, *int) {
	headers := &http.Header{}
	calls := new(int)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*calls++
		*headers = r.Header.Clone()
		_, _ = w.Write([]byte(`{"data":[{"permission":"patch:*:*"},{"permission":"inventory:*:*"}]}`))
	}))
	t.Cleanup(srv.Close)

	resetKesselAuth(t)
	utils.CoreCfg.RbacAddress = srv.URL
	utils.CoreCfg.RbacAuthenticated = authenticated
	utils.CoreCfg.KesselAuthOIDCIssuer = issuer
	utils.CoreCfg.KesselAuthClientID = "client-id"
	utils.CoreCfg.KesselAuthClientSecret = "client-secret"
	return headers, calls
}

func serveRBAC() *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(xRHIdentity, testXRHIdentity)
	router := gin.New()
	router.Use(RBAC())
	router.GET("/", okHandler)
	router.ServeHTTP(w, req)
	return w
}

func TestRBACAuthenticatedAddsBearer(t *testing.T) {
	oidc := newOIDCServer(t, "sso-token")
	headers, calls := withRbacServer(t, true, oidc.URL)

	w := serveRBAC()
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, *calls)
	assert.Equal(t, "Bearer sso-token", headers.Get("Authorization"))
	assert.Equal(t, testXRHIdentity, headers.Get(xRHIdentity))
}

func TestRBACUnauthenticatedNoToken(t *testing.T) {
	// empty issuer would fail any token fetch, so a 200 proves none was attempted
	headers, calls := withRbacServer(t, false, "")

	w := serveRBAC()
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, *calls)
	assert.Empty(t, headers.Get("Authorization"))
	assert.Equal(t, testXRHIdentity, headers.Get(xRHIdentity))
}

func TestRBACAuthenticatedTokenFailureFailsClosed(t *testing.T) {
	_, calls := withRbacServer(t, true, "")

	w := serveRBAC()
	assert.Equal(t, http.StatusBadGateway, w.Code)
	assert.Equal(t, 0, *calls)
}

func TestRBACAndKesselShareCredentials(t *testing.T) {
	oidc := newOIDCServer(t, "sso-token")
	withRbacServer(t, true, oidc.URL)
	utils.CoreCfg.KesselInsecure = false
	utils.CoreCfg.KesselAuthEnabled = true

	_, err := getBearerToken(t.Context())
	require.NoError(t, err)
	first := credentials
	_, conn, err := setupClient(t.Context())
	require.NoError(t, err)
	defer conn.Close()
	assert.Same(t, first, credentials)
}

func TestRbacTLSConfigTrustsCA(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	caPath := filepath.Join(t.TempDir(), "ca.crt")
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw})
	require.NoError(t, os.WriteFile(caPath, caPEM, 0o600))

	orig := utils.CoreCfg.RbacCACert
	defer func() { utils.CoreCfg.RbacCACert = orig }()
	utils.CoreCfg.RbacCACert = caPath

	client := &http.Client{Transport: &http.Transport{TLSClientConfig: rbacTLSConfig()}}
	res, err := client.Get(srv.URL)
	require.NoError(t, err)
	res.Body.Close()
	assert.Equal(t, http.StatusOK, res.StatusCode)
}
