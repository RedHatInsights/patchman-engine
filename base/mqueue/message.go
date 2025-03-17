package mqueue

import "github.com/bytedance/sonic"

func MessageFromJSON(k string, v interface{}) (KafkaMessage, error) {
	var m KafkaMessage
	var err error

	m.Key = []byte(k)
	m.Value, err = sonic.Marshal(v)
	return m, err
}
