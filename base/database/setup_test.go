package database

import (
	"app/base/utils"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDBCheck(t *testing.T) {
	utils.SkipWithoutDB(t)
	Configure()
}

func TestConnConfigSetsCredentialsOutsideDSN(t *testing.T) {
	certPath := filepath.Join(t.TempDir(), "rootcert.crt")
	writeTestCert(t, certPath)

	cfg, err := ConnConfig(&PostgreSQLConfig{
		Host:               "dbhost",
		Port:               5433,
		User:               "user name",
		Database:           "db name",
		Passwd:             "secret host=evil.example sslmode=disable",
		SSLMode:            "verify-ca",
		SSLRootCert:        certPath,
		StatementTimeoutMs: 1500})
	require.NoError(t, err)
	assert.Equal(t, "dbhost", cfg.Host)
	assert.Equal(t, uint16(5433), cfg.Port)
	assert.Equal(t, "user name", cfg.User)
	assert.Equal(t, "db name", cfg.Database)
	assert.Equal(t, "secret host=evil.example sslmode=disable", cfg.Password)
	assert.Equal(t, "1500", cfg.RuntimeParams["statement_timeout"])
	assert.NotContains(t, cfg.ConnString(), "secret")
	assert.NotContains(t, cfg.ConnString(), "user name")
	assert.NotContains(t, cfg.ConnString(), "db name")
	assert.Contains(t, cfg.ConnString(), "sslrootcert="+certPath)
	assert.NotNil(t, cfg.TLSConfig)
	assert.NotContains(t, cfg.ConnString(), "statement_timeout")
}

func writeTestCert(t *testing.T, path string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)
	file, err := os.Create(path)
	require.NoError(t, err)
	defer file.Close()
	require.NoError(t, pem.Encode(file, &pem.Block{Type: "CERTIFICATE", Bytes: der}))
}

func TestAdditionalParams(t *testing.T) {
	utils.SkipWithoutDB(t)
	Configure()

	assert.True(t, len(OtherAdvisoryTypes) == 2)
}
