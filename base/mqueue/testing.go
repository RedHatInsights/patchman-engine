package mqueue

type mockReader struct{}

func (t *mockReader) HandleEvents(_ EventHandler) {}
func (t *mockReader) Close() error                { return nil }

// Count how many times reader is created.
func CreateCountedMockReader(cnt *int) CreateReader {
	return func(_ string) Reader {
		reader := &mockReader{}
		*cnt++
		return reader
	}
}
