package metrics

import (
	"app/base/certutil"
	"app/base/utils"
	"crypto/tls"

	"github.com/prometheus/client_golang/prometheus"
)

const candlepinCertLabel = "candlepin"

// CertificateExpiryDays mirrors content-sources-backend certificate_expiry_days (GaugeVec by label).
// It is registered only where metrics are pushed (e.g. vmaas_sync), not on every pod's default registry.
var CertificateExpiryDays = prometheus.NewGaugeVec(prometheus.GaugeOpts{
	Help:      "Number of days until the certificate expires by the certificate label",
	Namespace: "patchman_engine",
	Subsystem: "core",
	Name:      "certificate_expiry_days",
}, []string{"certificate_label"})

// UpdateCandlepinCertificateExpiry refreshes the candlepin series from CoreCfg (single shot; no background loop).
func UpdateCandlepinCertificateExpiry() {
	applyCandlepinCertExpiry(CertificateExpiryDays, utils.CoreCfg.CandlepinCert, utils.CoreCfg.CandlepinKey)
}

// applyCandlepinCertExpiry refreshes or removes the candlepin expiry series. On parse/calculation
// errors it deletes the label so Prometheus does not keep a stale last-good value.
func applyCandlepinCertExpiry(gauge *prometheus.GaugeVec, certPEM, keyPEM string) {
	if certPEM == "" || keyPEM == "" {
		gauge.DeleteLabelValues(candlepinCertLabel)
		return
	}
	cert, err := tls.X509KeyPair([]byte(certPEM), []byte(keyPEM))
	if err != nil {
		utils.LogError("err", err, "certificate_expiry: candlepin X509KeyPair")
		gauge.DeleteLabelValues(candlepinCertLabel)
		return
	}
	days, err := certutil.DaysTillExpiration(&cert)
	if err != nil {
		utils.LogError("err", err, "certificate_expiry: candlepin DaysTillExpiration")
		gauge.DeleteLabelValues(candlepinCertLabel)
		return
	}
	gauge.WithLabelValues(candlepinCertLabel).Set(float64(days))
}
