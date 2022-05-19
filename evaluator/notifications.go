package evaluator

import (
	"app/base"
	"app/base/models"
	"app/base/mqueue"
	ntf "app/base/notification"
	"app/base/utils"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const NewAdvisoriesEvent = "new-advisories"

var notificationsPublisher mqueue.Writer

func configureNotifications() {
	if topic := utils.Cfg.NotificationsTopic; topic != "" {
		notificationsPublisher = mqueue.NewKafkaWriterFromEnv(topic)
	}
}

func getAdvisoryTypeFromID(tx *gorm.DB, id int) (string, error) {
	var advisoryType string
	err := tx.Model(models.AdvisoryType{}).Select("name").Where("id = ?", id).Take(&advisoryType).Error
	if err != nil || advisoryType == "" {
		return "", errors.Wrap(err, "querying advisory type failed")
	}

	return advisoryType, nil
}

func getNotificationAdvisories(tx *gorm.DB, newAdvisories SystemAdvisoryMap) ([]ntf.Advisory, error) {
	advisories := make([]ntf.Advisory, 0, len(newAdvisories))

	for _, a := range newAdvisories {
		advisoryType, err := getAdvisoryTypeFromID(tx, a.Advisory.AdvisoryTypeID)
		if err != nil {
			return nil, errors.Wrap(err, "creation of advisory notification failed")
		}

		advisory := ntf.Advisory{
			AdvisoryName: a.Advisory.Name,
			AdvisoryType: advisoryType,
			Synopsis:     a.Advisory.Synopsis,
		}

		advisories = append(advisories, advisory)
	}

	return advisories, nil
}

func publishNewAdvisoriesNotification(tx *gorm.DB, inventoryID string, accountID int,
	newAdvisories SystemAdvisoryMap) error {
	if notificationsPublisher == nil {
		return nil
	}

	advisories, err := getNotificationAdvisories(tx, newAdvisories)
	if err != nil {
		return errors.Wrap(err, "publishing advisories notification failed")
	}

	events := make([]ntf.Event, 0, len(advisories))
	for _, advisory := range advisories {
		events = append(events, ntf.Event{Payload: advisory})
	}

	msg, err := mqueue.MessageFromJSON(
		inventoryID,
		ntf.MakeNotification(accountID, inventoryID, NewAdvisoriesEvent, events))
	if err != nil {
		return errors.Wrap(err, "creating message from notification failed")
	}

	err = notificationsPublisher.WriteMessages(base.Context, msg)
	if err != nil {
		return errors.Wrap(err, "writing message to notifications publisher failed")
	}

	return nil
}
