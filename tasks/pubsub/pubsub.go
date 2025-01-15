package pubsub

import (
	"app/base/database"
	"app/base/utils"
	"app/tasks"
	"os"
	"time"

	"gorm.io/gorm"
)

type SubscriptionState struct {
	Subname   string
	Status    string
	SyncState string
	RelStats  map[string]string
}

func CreatePubSub() {
	createPublication()
	createSubscription()

	var state SubscriptionState
	for {
		logProgress(&state)
		if state.Status == "streaming" {
			utils.LogInfo("Logical replica has caught up with the primary, replicating real-time changes", "STEADY")
			os.Exit(0)
		}
		time.Sleep(5 * time.Minute)
	}
}

func createPublication() {
	err := tasks.WithTx(func(tx *gorm.DB) error {
		return tx.Exec("CREATE PUBLICATION patch_pub FOR ALL TABLES").Error
	})
	if err != nil {
		utils.LogError("err", err.Error(), "Unable to create publication")
	}
}

func createSubscription() {
	err := tasks.WithTx(func(tx *gorm.DB) error {
		return tx.Exec(`
			CREATE SUBSCRIPTION patch_sub
			CONNECTION ?
			PUBLICATION patch_pub
			WITH (copy_data = true)`,
			database.DataSourceName(database.LReplicaPgConfig),
		).Error
	})
	if err != nil {
		utils.LogError("err", err.Error(), "Unable to create subscription")
	}
}

func logProgress(state *SubscriptionState) {
	var newState SubscriptionState
	newState.RelStats = make(map[string]string)

	err := tasks.WithTx(func(tx *gorm.DB) error {
		sqlDB, err := tx.DB()
		if err != nil {
			utils.LogError("err", err.Error(), "Unable to get generic sql.DB from gorm.DB")
		}

		row := sqlDB.QueryRow("SELECT subname, status, sync_state FROM pg_stat_subscription")

		if err := row.Scan(&newState.Subname, &newState.Status, &newState.SyncState); err != nil {
			utils.LogError("err", err.Error(), "Unable to scan pg_stat_subscription")
		}

		rows, err := sqlDB.Query(`
			SELECT sr.relname, srel.state
			FROM pg_subscription_rel srel
			JOIN pg_class sr ON sr.oid = srel.relid
		`)
		for rows.Next() {
			var relName, relState string
			err = rows.Scan(&relName, &relState)
			if err != nil {
				utils.LogError("err", err.Error(), "Unable to scan pg_subscription_rel")
			}
			newState.RelStats[relName] = relState
		}
		if state == nil || state.Status != newState.Status || state.SyncState != newState.SyncState {
			utils.LogInfo("subname", newState.Subname, "status", newState.Status, newState.SyncState)
		}
		for relName, relState := range newState.RelStats {
			if state == nil || state.RelStats[relName] != relState {
				utils.LogInfo("relname", relName, relState)
			}
		}
		state = &newState
		return err
	})
	if err != nil {
		utils.LogError("err", err.Error(), "Unable to log progress")
	}
}
