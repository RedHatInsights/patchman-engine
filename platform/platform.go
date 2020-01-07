package main

import (
	"app/base/utils"
	"context"
	"fmt"
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
	InitInventory(app)
	InitVMaaS(app)

	// Control endpoint handler
	app.POST("/control/upload", MockUploadHandler)
	app.POST("/control/delete", MockDeleteHandler)
	app.POST("/control/sync", MockSyncHandler)

	app.Run(":9001")
}

func MockSyncHandler(c *gin.Context) {
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

func mockIdentity()string {
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

	err := writer.WriteMessages(context.Background(), kafka.Message{
		Key:   []byte{},
		Value: []byte(message),
	})

	if err != nil {
		panic(err)
	}

}
func MockUploadHandler(c *gin.Context) {
	utils.Log().Info("Mocking platform upload event")
	identity := mockIdentity()

	// We need to format this message to not depend on listener.
	// TODO: Replace with a typed solution, once we move event code to base library
	event := fmt.Sprintf(`{ "id": "TEST-0000", "type": "created", "b64_identity": "%v"}`, identity)
	sendMessageToTopic("platform.upload.available", event)
	c.Status(http.StatusOK)
}

func MockDeleteHandler(c *gin.Context) {
	utils.Log().Info("Mocking platform delete event")

	identity := mockIdentity();
	event := fmt.Sprintf(`{ "id": "TEST-0000", "type": "delete", "b64_identity": "%v"}`, identity)
	sendMessageToTopic("platform.inventory.events", event)
	c.Status(http.StatusOK)
}

func main() {
	platformMock()
}
