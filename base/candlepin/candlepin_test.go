package candlepin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"app/base/utils"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCandlepinOverrideHeader(t *testing.T) {
	originalConfig := utils.CoreCfg
	originalLevel := log.GetLevel()
	t.Cleanup(func() {
		utils.CoreCfg = originalConfig
		log.SetLevel(originalLevel)
	})
	utils.CoreCfg.CandlepinCert = ""
	utils.CoreCfg.CandlepinKey = ""
	log.SetLevel(log.TraceLevel)

	tests := []struct {
		name          string
		method        string
		overrideValue string
		wantDebug     bool
	}{
		{name: "GET with override", method: http.MethodGet, overrideValue: "test-override-secret"},
		{name: "PUT with override", method: http.MethodPut, overrideValue: "test-override-secret"},
		{name: "GET without override", method: http.MethodGet, wantDebug: true},
		{name: "PUT without override", method: http.MethodPut, wantDebug: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			utils.CoreCfg.CandlepinXRhOverrideExposed = tc.overrideValue

			client := CreateCandlepinClient()
			t.Cleanup(client.HTTPClient.CloseIdleConnections)
			assert.Equal(t, tc.wantDebug, client.Debug, "HTTP dumps must not expose the override secret")

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, tc.method, r.Method)
				if tc.overrideValue == "" {
					assert.NotContains(t, r.Header, http.CanonicalHeaderKey("x-rh-override-exposed"))
				} else {
					assert.Equal(t, []string{tc.overrideValue}, r.Header.Values("x-rh-override-exposed"))
				}
				w.WriteHeader(http.StatusNoContent)
			}))
			t.Cleanup(server.Close)

			ctx := context.Background()
			resp, err := client.Request(&ctx, tc.method, server.URL, nil, &ConsumersDetailResponse{})
			require.NoError(t, err)
			require.NoError(t, resp.Body.Close())
			assert.Equal(t, http.StatusNoContent, resp.StatusCode)
		})
	}
}
