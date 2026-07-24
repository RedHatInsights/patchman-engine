package aggregator

import (
	"app/base/mqueue"
	"app/base/utils"

	"github.com/bytedance/sonic"
)

// TODO: stub - will process advisory update events in batches and update account_advisory table
func advisoryUpdateHandler(m mqueue.KafkaMessage) error {
	var event mqueue.AdvisoryUpdateEvent
	if err := sonic.Unmarshal(m.Value, &event); err != nil {
		utils.LogError("err", err, "Could not deserialize advisory update event")
		return nil
	}
	// TODO: advisory update code goes here
	_ = event
	return nil
}
