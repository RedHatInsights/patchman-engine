package listener

import (
	"app/base/mqueue"
	"testing"
)

// InitForTemplateAdvisoryIntegrationTest configures listener for cross-package integration tests.
func InitForTemplateAdvisoryIntegrationTest(t *testing.T) (*mqueue.MockKafkaWriter, func()) {
	t.Helper()
	configure()

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
