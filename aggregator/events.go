package aggregator

import (
	"app/base/mqueue"
)

// TODO: stub - will process advisory update events in batches and update account_advisory table
func advisoryUpdateHandler(m mqueue.KafkaMessage) error {
	return nil
}
