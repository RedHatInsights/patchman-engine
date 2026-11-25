package middlewares

import (
	"app/base"
	"app/base/utils"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/project-kessel/kessel-sdk-go/kessel/auth"
	"github.com/stretchr/testify/assert"
)

// resetRbacClient restores the package and config state touched by RBAC client tests.
func resetRbacClient(t *testing.T) {
	origCfg := utils.CoreCfg
	t.Cleanup(func() {
		utils.CoreCfg = origCfg
		credentials = auth.NewOAuth2ClientCredentials(
			origCfg.KesselAuthClientID, origCfg.KesselAuthClientSecret, origCfg.KesselAuthOIDCIssuer)
		rbacURL, httpClient, httpClientErr, httpOnce = "", &http.Client{}, nil, sync.Once{}
	})
	rbacURL, httpClient, httpClientErr, httpOnce = "", &http.Client{}, nil, sync.Once{}
}

func fakeSSO(t *testing.T, tokenCalls *int) *httptest.Server {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/.well-known/openid-configuration" {
			_ = json.NewEncoder(w).Encode(map[string]string{"issuer": srv.URL, "token_endpoint": srv.URL + "/token"})
			return
		}
		*tokenCalls++
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "tok", "token_type": "Bearer", "expires_in": 3600})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func setAuthConfig(rbacAddr, issuer string) {
	utils.CoreCfg.RbacAddress = rbacAddr
	utils.CoreCfg.RbacAuthenticated = true
	utils.CoreCfg.KesselAuthClientID = "id"
	utils.CoreCfg.KesselAuthClientSecret = "secret"
	utils.CoreCfg.KesselAuthOIDCIssuer = issuer
	credentials = auth.NewOAuth2ClientCredentials("id", "secret", issuer)
}

func TestRbacBearerAddedWhenAuthenticated(t *testing.T) {
	resetRbacClient(t)
	var got http.Header
	rbacSrv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer rbacSrv.Close()
	caPath := filepath.Join(t.TempDir(), "ca.crt")
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: rbacSrv.Certificate().Raw})
	assert.NoError(t, os.WriteFile(caPath, caPEM, 0o600))

	tokenCalls := 0
	sso := fakeSSO(t, &tokenCalls)
	setAuthConfig(rbacSrv.URL, sso.URL)
	utils.CoreCfg.RbacCACertificate = caPath

	client, err := makeClient(base.Context, "xrhid")
	assert.NoError(t, err)
	_, err = client.Request(&base.Context, http.MethodGet, rbacURL, nil, &struct{}{})
	assert.NoError(t, err)
	assert.Equal(t, "Bearer tok", got.Get("Authorization"))
	assert.Equal(t, "xrhid", got.Get(xRHIdentity))
	assert.Equal(t, 1, tokenCalls)
}

func TestRbacNoBearerWhenNotAuthenticated(t *testing.T) {
	resetRbacClient(t)
	tokenCalls := 0
	sso := fakeSSO(t, &tokenCalls)
	utils.CoreCfg.RbacAddress = "http://rbac-service.svc:8000"
	utils.CoreCfg.RbacAuthenticated = false
	utils.CoreCfg.KesselAuthOIDCIssuer = sso.URL

	client, err := makeClient(base.Context, "xrhid")
	assert.NoError(t, err)
	assert.NotContains(t, client.DefaultHeaders, "Authorization")
	assert.Equal(t, 0, tokenCalls)
}

func TestRbacAuthenticatedFailsClosed(t *testing.T) {
	resetRbacClient(t)

	// token endpoint failing
	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	defer failing.Close()
	setAuthConfig("https://ewgw:443", failing.URL)
	credentials = auth.NewOAuth2ClientCredentials("id", "secret", failing.URL, auth.WithMaxRetries(0))
	client, err := makeClient(base.Context, "xrhid")
	assert.Error(t, err)
	assert.Nil(t, client)

	// missing credentials
	setAuthConfig("https://ewgw:443", failing.URL)
	utils.CoreCfg.KesselAuthClientSecret = ""
	client, err = makeClient(base.Context, "xrhid")
	assert.Error(t, err)
	assert.Nil(t, client)

	// not https
	tokenCalls := 0
	sso := fakeSSO(t, &tokenCalls)
	setAuthConfig("http://ewgw:80", sso.URL)
	client, err = makeClient(base.Context, "xrhid")
	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Equal(t, 0, tokenCalls)
}
