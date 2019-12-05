package vmaas_sync

import (
	"app/base/utils"
	"app/manager/middlewares"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	messagesReceived = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "patchman_engine",
		Subsystem: "vmaas_sync",
		Name:      "websocket_msgs",
	})
)

func init() {
	prometheus.MustRegister(messagesReceived)
}

func runMetrics() {
	// create web app
	app := gin.New()
	middlewares.Prometheus().Use(app)
	err := app.Run(":8083")
	if err != nil {
		utils.Log("err", err.Error()).Error()
		panic(err)
	}
}

func runWebsocket(conn *websocket.Conn, handler func(data []byte, conn *websocket.Conn)) {
	defer conn.Close()

	err := conn.WriteMessage(websocket.TextMessage, []byte("subscribe-listener"))
	if err != nil {
		utils.Log("err", err.Error()).Fatal("Could not subscribe for updates")
		panic(err)
	}

	for {
		messagesReceived.Add(1)
		typ, msg, err := conn.ReadMessage()
		if err != nil {
			utils.Log("err", err.Error()).Fatal("Failed to retrive VMaaS websocket message")
			panic(err)
		}
		if typ == websocket.BinaryMessage || typ == websocket.TextMessage {
			handler(msg, conn)
		}
		// TODO: Handle control messages
	}
}

func websocketHandler(data []byte, conn *websocket.Conn) {
	utils.Log("data", string(data)).Info("Received VMaaS websocket message")
}

func RunVmaasSync() {
	go runMetrics()

	conn, _, err := websocket.DefaultDialer.Dial(utils.GetenvOrFail("VMAAS_WS_ADDRESS"), nil)

	if err != nil {
		utils.Log("err", err.Error()).Fatal("Failed to connect to VMaaS")
		panic(err)
	}
	runWebsocket(conn, websocketHandler)
}
