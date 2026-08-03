package evaluator

import (
	"app/base"
	"app/base/models"
	"app/base/mqueue"
	ntf "app/base/notification"
	"app/base/utils"
	"time"

	"github.com/bytedance/sonic"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

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

	err := tx.Table("advisory_account_data as acd").
		Select("am.id as advisory_id, am.name as advisory_name, at.name as advisory_type, am.synopsis").
		Joins("inner join advisory_metadata am on am.id = acd.advisory_id").
		Joins("inner join advisory_type at on at.id = am.advisory_type_id").
		Where("acd.rh_account_id = ? AND acd.advisory_id IN (?)"+
			"AND acd.notified IS NULL AND acd.systems_installable > 0", accountID, advIDs).
		Order("am.name ASC").
		Scan(&unAdvs).Error
	if err != nil {
		return nil, errors.Wrap(err, "querying unnotified advisories from DB failed")
	}

	return unAdvs, nil
}

func getSystemTags(tx *gorm.DB, system *models.SystemPlatformV2) ([]ntf.SystemTag, error) {
	if system == nil {
		return nil, nil
	}

	var tags []ntf.SystemTag
	var tagsJSON string
	err := tx.Table("system_inventory").
		Select("tags").
		Where("rh_account_id = ?", system.Inventory.RhAccountID).
		Where("id = ?", system.InternalSystemID()).
		Scan(&tagsJSON).Error
	if err != nil {
		return nil, errors.Wrap(err, "system tags query failed")
	}
	if err = sonic.Unmarshal([]byte(tagsJSON), &tags); err != nil {
		return nil, errors.Wrap(err, "system tags unmarshal failed")
	}

	return tags, nil
}

func markAdvisoriesNotified(tx *gorm.DB, accountID int, advisoryIDs []int64) error {
	if len(advisoryIDs) == 0 {
		return nil
	}
	err := tx.Table("advisory_account_data").
		Where("rh_account_id = ? AND advisory_id IN (?)", accountID, advisoryIDs).
		Update("notified", time.Now()).Error
	if err != nil {
		return errors.Wrap(err, "updating notified column failed")
	}
	return nil
}

// publishNewAdvisoriesNotification publishes instant new-advisory notifications unless
// skipPublish is true. In both cases, matching advisory_account_data rows are marked notified
// when there is something to notify about (so skipPublish still prevents later flood).
func publishNewAdvisoriesNotification(tx *gorm.DB, system *models.SystemPlatformV2, orgID string,
	newAdvisories SystemAdvisoryMap, skipPublish bool) error {
	defer utils.ObserveSecondsSince(time.Now(), evaluationPartDuration.WithLabelValues("advisory-notification-publish"))

	advisories, err := getUnnotifiedAdvisories(tx, system.Inventory.RhAccountID, newAdvisories)
	if err != nil {
		return errors.Wrap(err, "getting unnotified advisories failed")
	}
	if len(advisories) == 0 {
		return nil
	}

	advisoryIDs := make([]int64, 0, len(advisories))
	for _, a := range advisories {
		advisoryIDs = append(advisoryIDs, a.AdvisoryID)
	}

	if skipPublish {
		utils.LogInfo("inventoryID", system.GetInventoryID(), "advisoryIDs", advisoryIDs, "orgID", orgID,
			"skipping advisory notification publish")
		return markAdvisoriesNotified(tx, system.Inventory.RhAccountID, advisoryIDs)
	}

	if notificationsPublisher == nil {
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

	notif, err := ntf.MakeNotification(&system.Inventory, tags, orgID, ntf.NewAdvisoryEvent, events)
	if err != nil {
		return errors.Wrap(err, "creating notification failed")
	}

	msg, err := mqueue.MessageFromJSON(system.GetInventoryID().String(), notif, nil)
	if err != nil {
		return errors.Wrap(err, "creating message from notification failed")
	}

	err = notificationsPublisher.WriteMessages(base.Context, msg)
	if err != nil {
		return errors.Wrap(err, "writing message to notifications publisher failed")
	}

	utils.LogInfo("inventoryID", system.GetInventoryID(), "advisoryIDs", advisoryIDs, "orgID", orgID,
		"notification sent successfully")

	return markAdvisoriesNotified(tx, system.Inventory.RhAccountID, advisoryIDs)
}
