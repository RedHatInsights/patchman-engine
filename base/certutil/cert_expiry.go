package certutil

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"time"
)

// ErrNoParseableCertificate is returned when the chain is non-empty but no DER entry could be parsed.
var ErrNoParseableCertificate = errors.New("certutil: no parseable certificate in chain")

// DaysTillExpiration returns whole days until the earliest NotAfter in the TLS certificate chain.
// If every raw certificate fails to parse, it returns an error so callers do not treat a broken
// configuration as “0 days left”.
func DaysTillExpiration(certs *tls.Certificate) (int, error) {
	expires := time.Time{}.UTC()
	found := false
	if certs == nil {
		return 0, nil
	}
	var lastParseErr error
	for _, tlsCert := range certs.Certificate {
		parsed, err := x509.ParseCertificate(tlsCert)
		if err != nil {
			lastParseErr = err
			continue
		}
		if !found || parsed.NotAfter.Before(expires) {
			expires = parsed.NotAfter
			found = true
		}
	}
	if !found {
		if lastParseErr != nil {
			return 0, fmt.Errorf("certutil: parse certificate: %w", lastParseErr)
		}
		return 0, ErrNoParseableCertificate
	}
	diff := time.Until(expires)
	return int(diff.Hours() / 24), nil
}
