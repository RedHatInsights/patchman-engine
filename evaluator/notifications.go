package evaluator

import (
	"app/base"
	"app/base/database"
	"app/base/models"
	"app/base/mqueue"
	ntf "app/base/notification"
	"app/base/utils"
	"time"

	"github.com/bytedance/sonic"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const NewAdvisoryEvent = "new-advisory"

var notificationsPublisher mqueue.Writer

func configureNotifications() {
	if topic := utils.CoreCfg.NotificationsTopic; topic != "" {
		notificationsPublisher = mqueue.NewKafkaWriterFromEnv(topic)
	}
}

func getUnnotifiedAdvisories(tx *gorm.DB, accountID int, newAdvs SystemAdvisoryMap) ([]ntf.Advisory, error) {
	unAdvs := make([]ntf.Advisory, 0, len(newAdvs))

	advIDs := make([]int64, 0, len(newAdvs))
	for _, a := range newAdvs {
		advIDs = append(advIDs, a.AdvisoryID)
	}

	var advNames []string
	err := tx.Table("advisory_account_data as acd").
		Select("am.name").
		Joins("inner join advisory_metadata am on am.id = acd.advisory_id").
		Where("acd.rh_account_id = ? AND acd.advisory_id IN (?)"+
			"AND acd.notified IS NULL AND acd.systems_installable > 0", accountID, advIDs).
		Order("am.name ASC").
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

func getSystemTags(tx *gorm.DB, system *models.SystemPlatform) ([]ntf.SystemTag, error) {
	if system == nil {
		return nil, nil
	}

	var tags []ntf.SystemTag
	var tagsJSON string
	err := tx.Table("system_inventory").
		Select("tags").
		Where("rh_account_id = ?", system.RhAccountID).
		Where("id = ?", system.ID).
		Scan(&tagsJSON).Error
	if err != nil {
		return nil, errors.Wrap(err, "system tags query failed")
	}
	if err = sonic.Unmarshal([]byte(tagsJSON), &tags); err != nil {
		return nil, errors.Wrap(err, "system tags unmarshal failed")
	}

	return tags, nil
}

func publishNewAdvisoriesNotification(tx *gorm.DB, system *models.SystemPlatform, orgID string,
	newAdvisories SystemAdvisoryMap) error {
	if notificationsPublisher == nil {
		return nil
	}

	tStart := time.Now()
	defer utils.ObserveSecondsSince(tStart, evaluationPartDuration.WithLabelValues("advisory-notification-publish"))

	advisories, err := getUnnotifiedAdvisories(tx, system.RhAccountID, newAdvisories)
	if err != nil {
		return errors.Wrap(err, "getting unnotified advisories failed")
	}
	if advisories == nil {
		return nil
	}

	events := make([]ntf.Event, 0, len(advisories))
	for _, advisory := range advisories {
		// At least empty metadata required to avoid NPE further on at the time of writing.
		events = append(events, ntf.Event{Payload: advisory, Metadata: ntf.Metadata{}})
	}

	tags, err := getSystemTags(tx, system)
	if err != nil {
		return errors.Wrap(err, "getting system tags failed")
	}

	notif, err := ntf.MakeNotification(system, tags, orgID, NewAdvisoryEvent, events)
	if err != nil {
		return errors.Wrap(err, "creating notification failed")
	}

	msg, err := mqueue.MessageFromJSON(system.InventoryID, notif, nil)
	if err != nil {
		return errors.Wrap(err, "creating message from notification failed")
	}

	err = notificationsPublisher.WriteMessages(base.Context, msg)
	if err != nil {
		return errors.Wrap(err, "writing message to notifications publisher failed")
	}

	advisoryIDs := make([]int64, 0, len(advisories))
	for _, a := range advisories {
		advisoryIDs = append(advisoryIDs, a.AdvisoryID)
	}

	utils.LogInfo("inventoryID", system.InventoryID, "advisoryIDs", advisoryIDs, "orgID", orgID,
		"notification sent successfully")

	err = tx.Table("advisory_account_data").
		Where("rh_account_id = ? AND advisory_id IN (?)", system.RhAccountID, advisoryIDs).
		Update("notified", time.Now()).Error
	if err != nil {
		return errors.Wrap(err, "updating notified column failed")
	}

	return nil
}
