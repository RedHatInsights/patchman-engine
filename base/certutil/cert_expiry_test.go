package certutil

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDaysTillExpiration(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.NoError(t, err)

	notBefore := time.Now().UTC().Add(-time.Hour)
	notAfter := notBefore.Add(100 * 24 * time.Hour)

	tmpl := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
	}

	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	assert.NoError(t, err)

	cert := tls.Certificate{
		Certificate: [][]byte{der},
		PrivateKey:  key,
	}

	days, err := DaysTillExpiration(&cert)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, days, 99)
	assert.LessOrEqual(t, days, 100)
}

func TestDaysTillExpirationNil(t *testing.T) {
	days, err := DaysTillExpiration(nil)
	assert.NoError(t, err)
	assert.Equal(t, 0, days)
}

func TestDaysTillExpirationUnparsableChain(t *testing.T) {
	cert := tls.Certificate{
		Certificate: [][]byte{[]byte("not valid der")},
	}
	_, err := DaysTillExpiration(&cert)
	assert.Error(t, err)
	assert.NotErrorIs(t, err, ErrNoParseableCertificate)
}

func TestDaysTillExpirationEmptyChain(t *testing.T) {
	cert := tls.Certificate{Certificate: [][]byte{}}
	_, err := DaysTillExpiration(&cert)
	assert.ErrorIs(t, err, ErrNoParseableCertificate)
}
