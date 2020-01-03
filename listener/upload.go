package listener

import (
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/RedHatInsights/patchman-clients/inventory"
	"github.com/RedHatInsights/patchman-clients/vmaas"
	"github.com/segmentio/kafka-go"
	"time"
)

func uploadHandler(m kafka.Message) {
	utils.Log("msg", string(m.Value)).Info("Msg received")
	event, identity, err := parseUploadMessage(m)
	if err != nil {
		utils.Log("err", err.Error()).Error("unable to parse upload msg")
		return
	}
	hostUploadReceived(event.Id, identity.Identity.AccountNumber, *event.B64Identity)
}

func parseUploadMessage(m kafka.Message) (*PlatformEvent, *utils.Identity, error) {
	var event PlatformEvent
	err := json.Unmarshal(m.Value, &event)
	if err != nil {
		utils.Log("err", err.Error()).Error("Could not deserialize host event")
		return nil, nil, errors.New("Could not deserialize host event")
	}
	// We need the b64 identity in order to call the inventory
	if event.B64Identity == nil {
		utils.Log().Error("No identity provided")
		return nil, nil, errors.New("No identity provided")
	}

	identity, err := utils.ParseIdentity(*event.B64Identity)
	if err != nil {
		utils.Log("err", err.Error()).Error("Could not parse identity")
		return nil, nil, errors.New("Could not parse identity")
	}

	if !identity.IsSmartEntitled() {
		utils.Log("account", identity.Identity.AccountNumber).Info("Is not smart entitled")
		return nil, nil, errors.New("Is not smart entitled")
	}
	return &event, identity, nil
}

// Stores or updates the account data, returning the account id
func getOrCreateAccount(account string) (int, error) {
	rhAccount := models.RhAccount{
		Name: account,
	}

	err := database.OnConflictUpdate(database.Db, "name", "name").Create(&rhAccount).Error
	return rhAccount.ID, err
}

// Stores or updates base system profile, returing internal system id
func updateSystemPlatform(inventoryId string, accountId int, updatesReq *vmaas.UpdatesRequest) (int, error) {
	updatesReqJSON, err := json.Marshal(&updatesReq)
	if err != nil {
		utils.Log("err", err.Error()).Error("Serializing vmaas request")
		return 0, err
	}

	hash := sha256.New()
	hash.Write(updatesReqJSON)
	jsonChecksum := hex.EncodeToString(hash.Sum([]byte{}))

	now := time.Now()

	dbHost := models.SystemPlatform{
		InventoryID:    inventoryId,
		RhAccountID:    accountId,
		VmaasJSON:      string(updatesReqJSON),
		JsonChecksum:   jsonChecksum,
		LastEvaluation: nil,
		LastUpload:     &now,
	}

	tx := database.OnConflictUpdate(database.Db, "inventory_id", "vmaas_json", "json_checksum", "last_evaluation", "last_upload")
	err = tx.Create(&dbHost).Error

	if err != nil {
		utils.Log("err", err.Error()).Error("Saving host into the database")
		return 0, err
	}
	return dbHost.ID, nil
}

// We have received new upload, update stored host data, and re-evaluate the host against VMaaS
func hostUploadReceived(hostId string, account string, identity string) {
	utils.Log("hostId", hostId).Debug("Downloading system profile")

	apiKey := inventory.APIKey{Prefix: "", Key: identity}
	// Create new context, which has the apikey value set. This is then used as a value for `x-rh-identity`
	ctx := context.WithValue(context.Background(), inventory.ContextAPIKey, apiKey)

	inventoryData, res, err := inventoryClient.HostsApi.ApiHostGetHostSystemProfileById(ctx, []string{hostId}, nil)
	if err != nil {
		utils.Log("err", err.Error()).Error("Could not inventory system profile")
		return
	}

	utils.Log("res", res).Debug("System profile download complete")

	if inventoryData.Count == 0 {
		utils.Log().Info("No system details returned")
		return
	}

	// We only process one host per message here
	host := inventoryData.Results[0]
	// Ensure we have account stored
	accountId, err := getOrCreateAccount(account)
	if err != nil {
		utils.Log("err", err.Error()).Error("Saving account into the database")
		return
	}

	// Prepare VMaaS request
	updatesReq := vmaas.UpdatesRequest{
		PackageList: host.SystemProfile.InstalledPackages,
	}

	systemId, err := updateSystemPlatform(host.Id, accountId, &updatesReq)
	if err != nil {
		utils.Log("err", err.Error()).Error("Saving system into the database")
		return
	}

	// Evaluation part
	utils.Log("systemId", systemId).Info("TODO - evaluate")
	// TODO call evaluate, later it will be moved to separate component - evaluator
}
