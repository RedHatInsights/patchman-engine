package listener

import (
	"app/base"
	"app/base/database"
	"app/base/mqueue"
	"app/base/utils"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

func lookupTemplateSystemInventoryIDs(tx *gorm.DB, accountID int, templateID int64) ([]uuid.UUID, error) {
	var inventoryIDs []uuid.UUID
	err := tx.Table("system_patch sp").
		Select("si.inventory_id").
		Joins("JOIN system_inventory si ON si.id = sp.system_id AND si.rh_account_id = sp.rh_account_id").
		Where("sp.rh_account_id = ? AND sp.template_id = ?", accountID, templateID).
		Scan(&inventoryIDs).Error
	if err != nil {
		return nil, errors.Wrap(err, "looking up template systems")
	}
	return inventoryIDs, nil
}

func inventoryIDsToEvalData(accountID int, orgID string, inventoryIDs []uuid.UUID) mqueue.EvalDataSlice {
	evalDataList := make(mqueue.EvalDataSlice, 0, len(inventoryIDs))
	for _, id := range inventoryIDs {
		evalDataList = append(evalDataList, mqueue.EvalData{
			InventoryID: id,
			RhAccountID: accountID,
			OrgID:       &orgID,
		})
	}
	return evalDataList
}

// SendTemplateRecalc looks up systems assigned to the template and requests their re-evaluation.
// Call before unassigning systems from a deleted template so inventory IDs are still known.
func SendTemplateRecalc(accountID int, orgID string, templateID int64) error {
	if !enableTemplateAdvisoryEval {
		return nil
	}
	if orgID == "" {
		utils.LogWarn("template_id", templateID, "account_id", accountID, "skipping template recalc: missing org_id")
		return nil
	}
	inventoryIDs, err := lookupTemplateSystemInventoryIDs(database.DB, accountID, templateID)
	if err != nil {
		return err
	}
	SendTemplateSystemsRecalc(accountID, orgID, inventoryIDs)
	return nil
}

// SendTemplateSystemsRecalc requests re-evaluation of explicit systems.
func SendTemplateSystemsRecalc(accountID int, orgID string, inventoryIDs []uuid.UUID) {
	if !enableTemplateAdvisoryEval {
		return
	}
	if orgID == "" {
		utils.LogWarn("account_id", accountID, "nSystems", len(inventoryIDs),
			"skipping template systems recalc: missing org_id")
		return
	}
	if len(inventoryIDs) == 0 {
		return
	}
	err := mqueue.SendMessages(base.Context, createdSystemsWriter, inventoryIDsToEvalData(accountID, orgID, inventoryIDs))
	if err != nil {
		utils.LogError("err", err.Error(), "account_id", accountID, "nSystems", len(inventoryIDs),
			"Template systems recalc message sending failed")
	}
}
