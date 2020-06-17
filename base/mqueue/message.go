package mqueue

import (
	"encoding/json"
	"github.com/segmentio/kafka-go"
)

func MessageFromJSON(k string, v interface{}) (kafka.Message, error) {
	var m kafka.Message
	var err error

	m.Key = []byte(k)
	m.Value, err = json.Marshal(v)
	return m, err
}
