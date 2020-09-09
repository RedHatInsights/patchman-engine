package mqueue

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

type infiniteReader struct{}

func (t *infiniteReader) HandleMessages(_ MessageHandler) {
	select {}
}
func (t *infiniteReader) Close() error { return nil }

func CreateBlockingReader(_ string) Reader {
	return &infiniteReader{}
}
