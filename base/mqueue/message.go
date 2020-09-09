package mqueue

import (
	"encoding/json"
)

type Message struct {
	Key   []byte
	Value []byte
}

func MessageFromJSON(k string, v interface{}) (Message, error) {
	var m Message
	var err error

	m.Key = []byte(k)
	m.Value, err = json.Marshal(v)
	return m, err
}
