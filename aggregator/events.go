package aggregator

import (
	"app/base/mqueue"
)

// TODO: stub - will process advisory update events in batches and update account_advisory table
func advisoryUpdateHandler(event mqueue.AdvisoryUpdateEvent) error {
	return nil
}
