package evaluator

import (
	"app/base"
	"app/base/database"
	"app/base/mqueue"
	ntf "app/base/notification"
	"app/base/utils"
	"github.com/pkg/errors"
)

const NewAdvisoryEvent = "new-advisory"

var notificationsPublisher mqueue.Writer

func configureNotifications() {
	if topic := utils.Cfg.NotificationsTopic; topic != "" {
		notificationsPublisher = mqueue.NewKafkaWriterFromEnv(topic)
	}
}

func getNotificationAdvisories(newAdvisories SystemAdvisoryMap) []ntf.Advisory {
	advisories := make([]ntf.Advisory, 0, len(newAdvisories))

	for _, a := range newAdvisories {
		advisory := ntf.Advisory{
			AdvisoryName: a.Advisory.Name,
			AdvisoryType: database.AdvisoryTypes[a.AdvisoryID],
			Synopsis:     a.Advisory.Synopsis,
		}

		advisories = append(advisories, advisory)
	}

	return advisories
}

func publishNewAdvisoriesNotification(inventoryID, accountName string, newAdvisories SystemAdvisoryMap) error {
	if notificationsPublisher == nil {
		return nil
	}

	advisories := getNotificationAdvisories(newAdvisories)
	events := make([]ntf.Event, 0, len(advisories))
	for _, advisory := range advisories {
		events = append(events, ntf.Event{Payload: advisory})
	}

	msg, err := mqueue.MessageFromJSON(
		inventoryID,
		ntf.MakeNotification(inventoryID, accountName, NewAdvisoryEvent, events))
	if err != nil {
		return errors.Wrap(err, "creating message from notification failed")
	}

	err = notificationsPublisher.WriteMessages(base.Context, msg)
	if err != nil {
		return errors.Wrap(err, "writing message to notifications publisher failed")
	}

	return nil
}
