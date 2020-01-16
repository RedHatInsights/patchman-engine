package listener

import (
	"app/base/database"
	"app/base/utils"
)

func deleteHandler(event PlatformEvent) {
	if event.Type == nil || *event.Type != "delete" {
		return
	}

	query := database.Db.Exec("select deleted_inventory_id from delete_system(?)", event.ID)
	err := query.Error

	if err != nil {
		utils.Log("id", event.ID, "err", err.Error()).Error("Could not delete system")
		return
	}

	utils.Log("id", event.ID, "count", query.RowsAffected).Info("Systems deleted")
}
