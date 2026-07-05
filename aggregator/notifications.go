package aggregator

import (
	"app/base/mqueue"
	"app/base/utils"
)

var notificationsPublisher mqueue.Writer

func configureNotifications() {
	if topic := utils.CoreCfg.NotificationsTopic; topic != "" {
		notificationsPublisher = mqueue.NewKafkaWriterFromEnv(topic)
	}
}
