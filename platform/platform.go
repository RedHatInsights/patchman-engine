package main

import (
	"app/base"
	"app/base/utils"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/segmentio/kafka-go"
	"net/http"
)

var websockets []chan string

func AddWebsocket() chan string {
	ws := make(chan string, 100)
	websockets = append(websockets, ws)
	return ws
}

func platformMock() {
	utils.Log().Info("Platform mock starting")
	app := gin.New()
	InitVMaaS(app)
	InitRbac(app)

	// Control endpoint handler
	app.POST("/control/upload", MockUploadHandler)
	app.POST("/control/delete", MockDeleteHandler)
	app.POST("/control/sync", MockSyncHandler)

	err := utils.RunServer(base.Context, app, ":9001")
	if err != nil {
		panic(err)
	}
}

func MockSyncHandler(_ *gin.Context) {
	utils.Log().Info("Mocking VMaaS sync event")
	// Force connected websocket clients to refresh
	for _, ws := range websockets {
		ws <- "sync"
	}
}

func mockKafkaWriter(topic string) *kafka.Writer {
	return kafka.NewWriter(kafka.WriterConfig{
		Brokers:  []string{"localhost:9092"},
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
func MockUploadHandler(c *gin.Context) {
	utils.Log().Info("Mocking platform upload event")

	event := map[string]interface{}{
		"type": "created",
		"host": map[string]interface{}{
			"id":             "TEST-0000",
			"account":        "TEST-0000",
			"system_profile": makeSystemProfile("TEST-0000"),
		},
	}
	msg, err := json.Marshal(event)
	if err != nil {
		panic(err)
	}
	sendMessageToTopic("platform.inventory.host-egress", string(msg))
	c.Status(http.StatusOK)
}

func MockDeleteHandler(c *gin.Context) {
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

func main() {
	platformMock()
}
