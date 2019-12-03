package listener

import (
	"app/base/utils"
	"app/manager/middlewares"
	"context"
	"encoding/json"
	"github.com/RedHatInsights/patchman-clients/inventory"
	"github.com/RedHatInsights/patchman-clients/vmaas"
	"github.com/antihax/optional"
	"github.com/gin-gonic/gin"
	"github.com/segmentio/kafka-go"
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
		// Spawn handler, not blocking the receiving thread
		go handler(m)
	}
}

func logHandler(m kafka.Message) {
	utils.Log().Info("Received message [", m.Topic, "] ", string(m.Value))
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
	if event.B64Identity != nil && event.Url != nil {
		identity, err := utils.ParseIdentity(*event.B64Identity)
		if err != nil {
			utils.Log("err", err.Error()).Error("Could not parse identity")
			return
		}


		if identity.IsSmartEntitled() {
			utils.Log("url", *event.Url).Error("Downloading system profile")
			downloadSystemProfile(event.Id, *event.B64Identity)
		} else {
			utils.Log("account", identity.Identity.AccountNumber).Info("Is not smart entitled")
		}
	} else {
		utils.Log("event", event).Info("Not a valid upload request")
	}
}

func downloadSystemProfile(hostId string, identity string) {
	utils.Log("hostId", hostId, "identity", identity).Error("Downloading system profile")

	// Create new context, which has the apikey value set. This value is then used as a value for `x-rh-identity`
	ctx := context.WithValue(context.Background(), inventory.ContextAPIKey, inventory.APIKey{Prefix: "", Key: identity})

	data, res, err := inventoryClient.HostsApi.ApiHostGetHostSystemProfileById(ctx, []string{hostId}, nil)

	if err != nil {
		utils.Log("err", err.Error()).Error("Could not Download body")
		return
	}

	utils.Log("data", data, "res", res).Error("Download complete")

	vars := vmaas.AppUpdatesHandlerV2PostPostOpts{
		UpdatesRequest: optional.NewInterface(vmaas.UpdatesRequest{
			// TODO: Proper logic here. This is just a test
			PackageList: data.Results[0].SystemProfile.InstalledPackages,
		}),
	}

	vmaasData, resp, err := vmaasClient.UpdatesApi.AppUpdatesHandlerV2PostPost(ctx, &vars)
	utils.Log("data", vmaasData, "res", resp).Error("VMAAS query complete")
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
	go baseListener(eventsReader, logHandler)

	// Just block. Any error will panic and kill the process.
	<-make(chan bool)

}
