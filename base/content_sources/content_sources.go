package content_sources

import (
	"app/base/api"
	"net/http"

	log "github.com/sirupsen/logrus"
)

type TemplateAdvisoryIDsResponse struct {
	AdvisoryIDs []string `json:"advisory_ids"`
}

func CreateContentSourcesClient() *api.Client {
	debugRequest := log.IsLevelEnabled(log.TraceLevel)
	return &api.Client{
		HTTPClient: &http.Client{},
		Debug:      debugRequest,
	}
}
