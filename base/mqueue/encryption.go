package mqueue

import (
	"app/base/utils"
	"crypto/tls"
	"crypto/x509"
	"github.com/segmentio/kafka-go"
	"io/ioutil"
	"time"
)

// Init encrypting dialer if env var configured or return nil
func tryCreateSecuredDialerFromEnv() *kafka.Dialer {
	enableKafkaSsl := utils.GetBoolEnvOrDefault("ENABLE_KAFKA_SSL", false)
	if !enableKafkaSsl {
		return nil
	}

	caCertPath := utils.GetenvOrFail("KAFKA_SSL_CERT")
	caCert, err := ioutil.ReadFile(caCertPath)
	if err != nil {
		panic(err)
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	tlsConfig := tls.Config{RootCAs: caCertPool} // nolint:gosec
	dialer := &kafka.Dialer{
		Timeout:   10 * time.Second,
		DualStack: true,
		TLS:       &tlsConfig,
	}
	return dialer
}
