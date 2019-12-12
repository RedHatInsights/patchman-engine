package listener

import (
	"app/base/database"
	"app/base/models"
	"app/base/utils"
	"app/manager/middlewares"
	"context"
	"encoding/json"
	"github.com/RedHatInsights/patchman-clients/inventory"
	"github.com/RedHatInsights/patchman-clients/vmaas"
	"github.com/antihax/optional"
	"github.com/gin-gonic/gin"
	"github.com/segmentio/kafka-go"
	"time"
)

const INVENTORY_API_PREFIX = "/api/inventory/v1"
const VMAAS_API_PREFIX = "/api"

var (
	uploadReader    *kafka.Reader
	eventsReader    *kafka.Reader
	inventoryClient *inventory.APIClient
	vmaasClient     *vmaas.APIClient
)

func configure() {
	uploadTopic := utils.GetenvOrFail("UPLOAD_TOPIC")
	eventsTopic := utils.GetenvOrFail("EVENTS_TOPIC")

	kafkaAddress := utils.GetenvOrFail("KAFKA_ADDRESS")
	kafkaGroup := utils.GetenvOrFail("KAFKA_GROUP")

	utils.Log("KafkaAddress", kafkaAddress).Info("Connecting to kafka")

	uploadConfig := kafka.ReaderConfig{
		Brokers:  []string{kafkaAddress},
		Topic:    uploadTopic,
		GroupID:  kafkaGroup,
		MinBytes: 1,
		MaxBytes: 10e6, // 1MB
	}

	uploadReader = kafka.NewReader(uploadConfig)

	eventsConfig := uploadConfig
	eventsConfig.Topic = eventsTopic

	eventsReader = kafka.NewReader(eventsConfig)

	inventoryAddr := utils.GetenvOrFail("INVENTORY_ADDRESS")

	config := inventory.NewConfiguration()
	config.Debug = true
	config.BasePath = inventoryAddr + INVENTORY_API_PREFIX

	inventoryClient = inventory.NewAPIClient(config)

	cfg := vmaas.NewConfiguration()
	cfg.BasePath = utils.GetenvOrFail("VMAAS_ADDRESS") + VMAAS_API_PREFIX
	cfg.Debug = true

	vmaasClient = vmaas.NewAPIClient(cfg)
}

func shutdown(reader *kafka.Reader) {
	err := reader.Close()
	if err != nil {
		utils.Log("err", err.Error()).Error("unable to shutdown Kafka reader")
	}
}

func baseListener(reader *kafka.Reader, handler func(message kafka.Message)) {
	defer shutdown(reader)

	for {
		m, err := reader.ReadMessage(context.Background())
		if err != nil {
			utils.Log("err", err.Error()).Error("unable to read message from Kafka reader")
			panic(err)
		}

		go handler(m)
	}
}

func logHandler(m kafka.Message) {
	utils.Log("topic", m.Topic, "value", string(m.Value)).Info("Received message ")
}

func uploadHandler(m kafka.Message) {
	var event PlatformEvent
	utils.Log("msg", string(m.Value)).Info("Msg received")

	err := json.Unmarshal(m.Value, &event)
	if err != nil {
		utils.Log("err", err.Error()).Error("Could not deserialize host event")
		return
	}
	// We need the b64 identity in order to call the inventory
	if event.B64Identity == nil {
		utils.Log().Error("No identity provided")
		return
	}

	identity, err := utils.ParseIdentity(*event.B64Identity)
	if err != nil {
		utils.Log("err", err.Error()).Error("Could not parse identity")
		return
	}

	if !identity.IsSmartEntitled() {
		utils.Log("account", identity.Identity.AccountNumber).Info("Is not smart entitled")
		return
	}
	// Spawn handler, not blocking the receiving goroutine
	hostUploadReceived(event.Id, identity.Identity.AccountNumber, *event.B64Identity)
}

// Stores or updates the account data
func getAccountId(account string) (int, error) {
	rhAccount := models.RhAccount{
		Name: account,
	}

	err := database.OnConflictUpdate(database.Db, "name", "name").Create(&rhAccount).Error

	return rhAccount.ID, err
}

// Stores or updates base system profile
func updateSystemPlatform(inventoryId string, accountId int, updatesReq *vmaas.UpdatesRequest) (int, error) {
	updatesReqStr, err := json.Marshal(&updatesReq)
	if err != nil {
		utils.Log("err", err.Error()).Error("Serializing vmaas request")
		return 0, err
	}

	now := time.Now()

	dbHost := models.SystemPlatform{
		InventoryID:    inventoryId,
		RhAccountID:    accountId,
		FirstReported:  now,
		VmaasJSON:      string(updatesReqStr),
		JsonChecksum:   "TODO",
		LastEvaluation: nil,
		LastUpload:     &now,
	}

	err = database.OnConflictUpdate(database.Db, "inventory_id", "vmaas_json", "json_checksum", "last_evaluation", "last_upload").Create(&dbHost).Error
	if err != nil {
		utils.Log("err", err.Error()).Error("Saving host into the database")
		return 0, err
	}
	return dbHost.ID, nil
}

// We have received new upload, update stored host data, and re-evaluate the host against VMaaS
func hostUploadReceived(hostId string, account string, identity string) {
	utils.Log("hostId", hostId, "identity", identity).Debug("Downloading system profile")

	// Create new context, which has the apikey value set. This value is then used as a value for `x-rh-identity`
	ctx := context.WithValue(context.Background(), inventory.ContextAPIKey, inventory.APIKey{Prefix: "", Key: identity})

	inventoryData, res, err := inventoryClient.HostsApi.ApiHostGetHostSystemProfileById(ctx, []string{hostId}, nil)
	if err != nil {
		utils.Log("err", err.Error()).Error("Could not Download body")
		return
	}

	utils.Log("inventoryData", inventoryData, "res", res).Debug("Download complete")

	if inventoryData.Count == 0 {
		utils.Log().Info("No system details returned")
		return
	}

	for _, host := range inventoryData.Results {

		accountId, err := getAccountId(account)
		if err != nil {
			utils.Log("err", err.Error()).Error("Saving account into the database")
			return
		}

		updatesReq := vmaas.UpdatesRequest{
			PackageList: host.SystemProfile.InstalledPackages,
		}

		_, err = updateSystemPlatform(host.Id, accountId, &updatesReq)
		if err != nil {
			utils.Log("err", err.Error()).Error("Saving account into the database")
			return
		}

		callVars := vmaas.AppUpdatesHandlerV2PostPostOpts{
			UpdatesRequest: optional.NewInterface(updatesReq),
		}

		vmaasData, resp, err := vmaasClient.UpdatesApi.AppUpdatesHandlerV2PostPost(ctx, &callVars)
		if err != nil {
			utils.Log("err", err.Error()).Error("Could not make VMaaS query")
			return
		}
		utils.Log("data", vmaasData, "res", resp).Info("VMAAS query complete")
	}
}

func runMetrics() {
	// create web app
	app := gin.New()
	middlewares.Prometheus().Use(app)

	err := app.Run(":8081")
	if err != nil {
		utils.Log("err", err.Error()).Error()
		panic(err)
	}
}

func RunListener() {
	utils.Log().Info("listener starting")

	// Start a web server for handling metrics so that readiness probe works
	go runMetrics()

	configure()

	go baseListener(uploadReader, uploadHandler)
	baseListener(eventsReader, logHandler)
}
