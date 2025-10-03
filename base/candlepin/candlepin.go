package candlepin

import (
	"app/base/api"
	"app/base/utils"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"
)

type ConsumersEnvironmentsRequest struct {
	ConsumerUuids  []string `json:"consumerUuids"`
	EnvironmentIDs []string `json:"environmentIds"`
}

type ConsumersUpdateResponse struct {
	Message string `json:"displayMessage"`
}

type ConsumersDetailResponse struct {
	Environments []ConsumersEnvironment `json:"environments"`
}

type ConsumersEnvironment struct {
	ID string `json:"id"`
}

var ErrCandlepin = errors.New("candlepin error")

var (
	// Toggle compression when calling Candlepi API
	CandlepinCallCmp = utils.PodConfig.GetBool("candlepin_call_compression", true)
	// Number of retries on Candlepin API
	CandlepinRetries = utils.PodConfig.GetInt("candlepin_retries", 5)
	// Toggle exponential retries on Candlepin API
	CandlepinExpRetries = utils.PodConfig.GetBool("candlepin_exp_retries", true)
)

func CreateCandlepinClient() api.Client {
	getTLSConfig := func() (*tls.Config, error) {
		var tlsConfig *tls.Config
		if utils.CoreCfg.CandlepinCert != "" && utils.CoreCfg.CandlepinKey != "" {
			clientCert, err := tls.X509KeyPair([]byte(utils.CoreCfg.CandlepinCert), []byte(utils.CoreCfg.CandlepinKey))
			if err != nil {
				return nil, err
			}
			certPool, err := x509.SystemCertPool()
			if err != nil {
				return nil, err
			}
			if utils.CoreCfg.CandlepinCA != "" {
				ok := certPool.AppendCertsFromPEM([]byte(utils.CoreCfg.CandlepinCA))
				if !ok {
					return nil, fmt.Errorf("could not parse candlepin ca cert")
				}
			}
			tlsConfig = &tls.Config{
				Certificates: []tls.Certificate{clientCert},
				RootCAs:      certPool,
				MinVersion:   tls.VersionTLS12,
			}
			utils.LogInfo("using cert to access candlepin")
		}
		return tlsConfig, nil
	}

	tlsConfig, err := getTLSConfig()
	if err != nil {
		utils.LogError("err", err, "parsing candlepin cert")
	}

	debugRequest := log.IsLevelEnabled(log.TraceLevel)

	return api.Client{
		HTTPClient: &http.Client{Transport: &http.Transport{
			DisableCompression: !CandlepinCallCmp,
			TLSClientConfig:    tlsConfig,
		}},
		Debug: debugRequest,
	}
}
