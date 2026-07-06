package listener

import (
	"app/base/content_sources"
	"app/base/mqueue"
	"app/base/utils"
	"testing"
)

// InitForTemplateAdvisoryIntegrationTest configures listener for cross-package integration tests.
func InitForTemplateAdvisoryIntegrationTest(t *testing.T) (*mqueue.MockKafkaWriter, func()) {
	t.Helper()
	configure()
	address := utils.Getenv("CONTENT_SOURCES_ADDRESS", "http://platform:9001")
	utils.CoreCfg.ContentSourcesAddress = address
	contentSourcesClient = content_sources.CreateContentSourcesClient()
	contentSourcesBaseURL = address + "/api/content-sources/v1"
	t.Cleanup(func() {
		contentSourcesClient = nil
		contentSourcesBaseURL = ""
	})

	origFlag := enableTemplateAdvisoryEval
	origWriter := createdSystemsWriter
	enableTemplateAdvisoryEval = true
	mockWriter := &mqueue.MockKafkaWriter{}
	createdSystemsWriter = mockWriter
	cleanup := func() {
		enableTemplateAdvisoryEval = origFlag
		createdSystemsWriter = origWriter
	}
	return mockWriter, cleanup
}
