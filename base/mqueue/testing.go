package mqueue

import (
	"context"
	"sync/atomic"
)

type mockReader struct{}

func (t *mockReader) HandleMessages(_ MessageHandler) {}
func (t *mockReader) Close() error                    { return nil }

// Count how many times reader is created.
func CreateCountedMockReader(cnt *int32) CreateReader {
	return func(_ string) Reader {
		reader := &mockReader{}
		atomic.AddInt32(cnt, 1)
		return reader
	}
}

type MockKafkaWriter struct {
	Messages []KafkaMessage
}

func (t *MockKafkaWriter) WriteMessages(_ context.Context, ev ...KafkaMessage) error {
	t.Messages = append(t.Messages, ev...)
	return nil
}
