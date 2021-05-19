package mqueue

import "context"

type mockReader struct{}

func (t *mockReader) HandleMessages(_ MessageHandler) {}
func (t *mockReader) Close() error                    { return nil }

// Count how many times reader is created.
func CreateCountedMockReader(cnt *int) CreateReader {
	return func(_ string) Reader {
		reader := &mockReader{}
		*cnt++
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
