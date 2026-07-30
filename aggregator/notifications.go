package aggregator

import (
	"app/base"
	"app/base/database"
	"app/base/mqueue"
	ntf "app/base/notification"
	"app/base/utils"
	"time"

	"gorm.io/gorm"
)

var notificationsPublisher mqueue.Writer

func configureNotifications() {
	if topic := utils.CoreCfg.NotificationsTopic; topic != "" {
		notificationsPublisher = mqueue.NewKafkaWriterFromEnv(topic)
	}
}

func getUnnotifiedAdvisories(tx *gorm.DB, rhAccountID int, advisoryIDs []int64) ([]ntf.Advisory, error) {
	var advisories []ntf.Advisory
	err := tx.Table("account_advisory aa").
		Select("DISTINCT am.id as advisory_id, am.name as advisory_name, at.name as advisory_type, am.synopsis").
		Joins("INNER JOIN advisory_metadata am ON am.id = aa.advisory_id").
		Joins("INNER JOIN advisory_type at ON at.id = am.advisory_type_id").
		Where("aa.rh_account_id = ? AND aa.advisory_id IN (?) AND aa.notified IS NULL AND aa.systems_installable > 0",
			rhAccountID, advisoryIDs).
		Order("am.name ASC").
		Scan(&advisories).Error
	return advisories, err
}

func publishNewAdvisoryNotification(rhAccountID int, advisoryIDs []int64) error {
	if notificationsPublisher == nil || !enableNotifications {
		return nil
	}

	tx := database.DB.WithContext(base.Context).Begin()
	defer tx.Rollback() //nolint:errcheck

	advisories, err := getUnnotifiedAdvisories(tx, rhAccountID, advisoryIDs)
	if err != nil {
		return err
	}
	if len(advisories) == 0 {
		return nil
	}

	var orgID string
	err = tx.Table("rh_account").Select("org_id").Where("id = ?", rhAccountID).Scan(&orgID).Error
	if err != nil {
		return err
	}

	events := make([]ntf.Event, 0, len(advisories))
	for _, advisory := range advisories {
		events = append(events, ntf.Event{Payload: advisory, Metadata: ntf.Metadata{}})
	}

	notif, err := ntf.MakeAccountNotification(orgID, ntf.NewAdvisoryEvent, events)
	if err != nil {
		return err
	}

	msg, err := mqueue.MessageFromJSON(orgID, notif, nil)
	if err != nil {
		return err
	}

	err = notificationsPublisher.WriteMessages(base.Context, msg)
	if err != nil {
		return err
	}

	notifiedIDs := make([]int64, 0, len(advisories))
	for _, a := range advisories {
		notifiedIDs = append(notifiedIDs, a.AdvisoryID)
	}

	err = markAdvisoriesNotified(tx, rhAccountID, notifiedIDs)
	if err != nil {
		return err
	}

	err = tx.Commit().Error
	if err != nil {
		return err
	}

	utils.LogInfo("rh_account_id", rhAccountID, "org_id", orgID, "advisory_count", len(advisories),
		"new advisory notification sent")

	return nil
}

func markAdvisoriesNotified(tx *gorm.DB, rhAccountID int, advisoryIDs []int64) error {
	return tx.Table("account_advisory").
		Where("rh_account_id = ? AND advisory_id IN (?)", rhAccountID, advisoryIDs).
		Update("notified", time.Now()).Error
}
