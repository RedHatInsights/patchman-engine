package mqueue

import (
	"encoding/json"
)

func MessageFromJSON(k string, v interface{}) (KafkaMessage, error) {
	var m KafkaMessage
	var err error

	m.Key = []byte(k)
	m.Value, err = json.Marshal(v)
	return m, err
}
