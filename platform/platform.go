package platform

import (
	"app/base"
	"app/base/utils"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/segmentio/kafka-go"
	"net/http"
	"time"
)

var websockets []chan string
var runUploadLoop = false

func addWebsocket() chan string {
	ws := make(chan string, 100)
	websockets = append(websockets, ws)
	return ws
}

func platformMock() {
	utils.Log().Info("Platform mock starting")
	app := gin.New()
	initVMaaS(app)
	initRbac(app)

	// Control endpoint handler
	app.POST("/control/upload", mockUploadHandler)
	app.POST("/control/delete", mockDeleteHandler)
	app.POST("/control/sync", mockSyncHandler)
	app.POST("/control/toggle_upload", mockToggleUpload)

	err := utils.RunServer(base.Context, app, ":9001")
	if err != nil {
		panic(err)
	}
}

func mockSyncHandler(_ *gin.Context) {
	utils.Log().Info("Mocking VMaaS sync event")
	// Force connected websocket clients to refresh
	for _, ws := range websockets {
		ws <- "sync"
	}
}

func mockKafkaWriter(topic string) *kafka.Writer {
	kafkaAddress := utils.GetenvOrFail("KAFKA_ADDRESS")
	return kafka.NewWriter(kafka.WriterConfig{
		Brokers:  []string{kafkaAddress},
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	})
}

func mockIdentity() string {
	identity, err := utils.Identity{
		Entitlements: map[string]utils.Entitlement{
			"smart_management": {Entitled: true},
		},
		Identity: utils.IdentityDetail{
			AccountNumber: "0",
			Type:          "User",
		},
	}.Encode()

	if err != nil {
		panic(err)
	}
	return identity
}

func upload(randomPkgs bool) {
	event := map[string]interface{}{
		"type": "created",
		"host": map[string]interface{}{
			"id":             "TEST-0000",
			"account":        "TEST-0000",
			"system_profile": makeSystemProfile("TEST-0000", randomPkgs),
		},
	}
	msg, err := json.Marshal(event)
	if err != nil {
		panic(err)
	}
	sendMessageToTopic("platform.inventory.host-egress", string(msg))
}

func sendMessageToTopic(topic, message string) {
	writer := mockKafkaWriter(topic)

	err := writer.WriteMessages(base.Context, kafka.Message{
		Key:   []byte{},
		Value: []byte(message),
	})

	if err != nil {
		panic(err)
	}
}

func mockUploadHandler(c *gin.Context) {
	utils.Log().Info("Mocking platform upload event")
	upload(false)
	c.Status(http.StatusOK)
}

func mockDeleteHandler(c *gin.Context) {
	utils.Log().Info("Mocking platform delete event")

	identity := mockIdentity()
	event := map[string]interface{}{
		"id":           "TEST-0000",
		"type":         "delete",
		"b64_identity": identity,
	}
	msg, err := json.Marshal(event)
	if err != nil {
		panic(err)
	}
	sendMessageToTopic("platform.inventory.events", string(msg))
	c.Status(http.StatusOK)
}

func mockToggleUpload(c *gin.Context) {
	runUploadLoop = !runUploadLoop
	c.JSON(http.StatusOK, fmt.Sprintf("%v", runUploadLoop))
}

func uploader() {
	var i int
	for {
		if runUploadLoop {
			upload(true)
			i++
			utils.Log("iteration", i).Info("upload loop running")
			time.Sleep(time.Millisecond * 10)
		} else {
			i = 0
			time.Sleep(time.Second)
		}
	}
}

func RunPlatformMock() {
	go uploader()
	platformMock()
}
