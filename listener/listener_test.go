package listener

import (
	"app/base/mqueue"
	"github.com/bmizerany/assert"
	"testing"
	"time"
)

var mockReaders []mqueue.Reader

type mockReader struct {
	Topic            string
	HandleEventCalls int
}

func (t *mockReader) HandleEvents(_ mqueue.EventHandler) {
	t.HandleEventCalls++
}
func (t *mockReader) Close() error { return nil }

func createMockReader(topic string) mqueue.Reader {
	reader := &mockReader{Topic: topic}
	mockReaders = append(mockReaders, reader)
	return reader
}

func TestRunReaders(t *testing.T) {
	runReaders(createMockReader)
	time.Sleep(time.Millisecond * 100)
	// it will create CONSUMER_COUNT (8) * topics (2) = readers (16)
	assert.Equal(t, 16, len(mockReaders))
}
