package mqueue

import (
	"github.com/bytedance/sonic"
	"github.com/segmentio/kafka-go"
)

func MessageFromJSON(k string, v interface{}, h []kafka.Header) (KafkaMessage, error) {
	var m KafkaMessage
	var err error

	m.Key = []byte(k)
	m.Headers = h
	m.Value, err = sonic.Marshal(v)
	return m, err
}
