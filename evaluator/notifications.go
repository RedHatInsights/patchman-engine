package evaluator

import (
	"app/base"
	"app/base/database"
	"app/base/mqueue"
	ntf "app/base/notification"
	"app/base/utils"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"time"
)

const NewAdvisoryEvent = "new-advisory"

var notificationsPublisher mqueue.Writer

func configureNotifications() {
	if topic := utils.Cfg.NotificationsTopic; topic != "" {
		notificationsPublisher = mqueue.NewKafkaWriterFromEnv(topic)
	}
}

func getUnnotifiedAdvisories(tx *gorm.DB, accountID int, newAdvs SystemAdvisoryMap) ([]ntf.Advisory, error) {
	unAdvs := make([]ntf.Advisory, 0, len(newAdvs))

	advIDs := make([]int, 0, len(newAdvs))
	for _, a := range newAdvs {
		advIDs = append(advIDs, a.AdvisoryID)
	}

	var advNames []string
	err := tx.Table("advisory_account_data as acd").
		Select("am.name").
		Joins("inner join advisory_metadata am on am.id = acd.advisory_id").
		Where("acd.rh_account_id = ? AND acd.advisory_id IN (?) AND acd.notified IS NULL", accountID, advIDs).
		Scan(&advNames).Error
	if err != nil {
		return nil, errors.Wrap(err, "querying unnotified advisories from DB failed")
	}

	if len(advNames) == 0 {
		return nil, nil
	}

	for _, n := range advNames {
		if a, ok := newAdvs[n]; ok {
			unAdvs = append(
				unAdvs,
				ntf.Advisory{
					AdvisoryID:   a.AdvisoryID,
					AdvisoryName: a.Advisory.Name,
					AdvisoryType: database.AdvisoryTypes[a.Advisory.AdvisoryTypeID],
					Synopsis:     a.Advisory.Synopsis,
				})
		}
	}

	return unAdvs, nil
}

func publishNewAdvisoriesNotification(tx *gorm.DB, inventoryID, accountName string,
	accountID int, newAdvisories SystemAdvisoryMap) error {
	if notificationsPublisher == nil {
		return nil
	}

	advisories, err := getUnnotifiedAdvisories(tx, accountID, newAdvisories)
	if err != nil {
		return errors.Wrap(err, "getting unnotified advisories failed")
	}
	if advisories == nil {
		return nil
	}

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

	advisoryIDs := make([]int, 0, len(advisories))
	for _, a := range advisories {
		advisoryIDs = append(advisoryIDs, a.AdvisoryID)
	}

	err = tx.Table("advisory_account_data").
		Where("rh_account_id = ? AND advisory_id IN (?)", accountID, advisoryIDs).
		Update("notified", time.Now()).Error
	if err != nil {
		return errors.Wrap(err, "updating notified column failed")
	}

	return nil
}
