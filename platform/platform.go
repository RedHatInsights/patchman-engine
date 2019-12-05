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

func MockUploadHandler(c *gin.Context) {
	utils.Log().Info("Mocking platform upload event")
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:  []string{"localhost:9092"},
		Topic:    "platform.upload.available",
		Balancer: &kafka.LeastBytes{},
	})

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

	// We need to format this message to not depend on listener.
	// TODO: Replace with a typed solution, once we move event code to base library
	event := fmt.Sprintf(`{ "id": "TEST-0000", "b64_identity": "%v"}`, identity)

	err = writer.WriteMessages(context.Background(), kafka.Message{
		Key:   []byte{},
		Value: []byte(event),
	})
	if err != nil {
		panic(err)
	}

	c.Status(http.StatusOK)

}

func main() {
	platformMock()
}
