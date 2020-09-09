package mqueue

type Counter interface {
	Inc()
}

// nolint: unused
var (
	kafkaErrorReadCnt  Counter = &emptyCnt{}
	kafkaErrorWriteCnt Counter = &emptyCnt{}
)

func SetKafkaErrorReadCnt(cnt Counter) {
	kafkaErrorReadCnt = cnt
}

func SetKafkaErrorWriteCnt(cnt Counter) {
	kafkaErrorWriteCnt = cnt
}

type emptyCnt struct{}

func (t *emptyCnt) Inc() {}
