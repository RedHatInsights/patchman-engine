package metrics

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testCandlepinPEMs(t *testing.T) (certPEM, keyPEM string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	notBefore := time.Now().UTC().Add(-time.Hour)
	notAfter := notBefore.Add(100 * 24 * time.Hour)
	tmpl := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test-candlepin"},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
	}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	require.NoError(t, err)

	var certBuf, keyBuf bytes.Buffer
	require.NoError(t, pem.Encode(&certBuf, &pem.Block{Type: "CERTIFICATE", Bytes: der}))
	require.NoError(t, pem.Encode(&keyBuf, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}))
	return certBuf.String(), keyBuf.String()
}

func TestApplyCandlepinCertExpiry_setsGauge(t *testing.T) {
	gv := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "patchman_engine",
		Subsystem: "core",
		Name:      "certificate_expiry_days_test_helper",
		Help:      "test",
	}, []string{"certificate_label"})

	certPEM, keyPEM := testCandlepinPEMs(t)
	applyCandlepinCertExpiry(gv, certPEM, keyPEM)

	v := testutil.ToFloat64(gv.WithLabelValues(candlepinCertLabel))
	assert.GreaterOrEqual(t, v, 99.0)
	assert.LessOrEqual(t, v, 100.0)
}

func TestApplyCandlepinCertExpiry_badPEMDeletesSeries(t *testing.T) {
	gv := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "patchman_engine",
		Subsystem: "core",
		Name:      "certificate_expiry_days_test_helper_bad",
		Help:      "test",
	}, []string{"certificate_label"})

	certPEM, keyPEM := testCandlepinPEMs(t)
	reg := prometheus.NewPedanticRegistry()
	reg.MustRegister(gv)

	applyCandlepinCertExpiry(gv, certPEM, keyPEM)
	require.NotZero(t, testutil.ToFloat64(gv.WithLabelValues(candlepinCertLabel)))

	applyCandlepinCertExpiry(gv, "not-valid-pem", "not-valid-pem")
	n, err := testutil.GatherAndCount(reg)
	require.NoError(t, err)
	assert.Zero(t, n)
}

func TestApplyCandlepinCertExpiry_missingConfigDeletesSeries(t *testing.T) {
	gv := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "patchman_engine",
		Subsystem: "core",
		Name:      "certificate_expiry_days_test_helper_clear",
		Help:      "test",
	}, []string{"certificate_label"})

	reg := prometheus.NewPedanticRegistry()
	reg.MustRegister(gv)

	certPEM, keyPEM := testCandlepinPEMs(t)
	applyCandlepinCertExpiry(gv, certPEM, keyPEM)
	applyCandlepinCertExpiry(gv, "", "")
	n, err := testutil.GatherAndCount(reg)
	require.NoError(t, err)
	assert.Zero(t, n)
}
