package vmaas_sync

import (
	"app/base/utils"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	ginprometheus "github.com/zsais/go-gin-prometheus"
)

func runMetrics() {
	// create web app
	app := gin.New()

	prometheus := ginprometheus.NewPrometheus("gin")
	prometheus.Use(app)
	err := app.Run(":8083")
	if err != nil {
		utils.Log("err", err.Error()).Error()
		panic(err)
	}
}

func runWebsocket(conn *websocket.Conn, handler func(data []byte, conn *websocket.Conn)) {
	defer conn.Close()

	err:= conn.WriteMessage(websocket.TextMessage, []byte("subscribe-listener"))
	if err != nil {
		utils.Log("err", err.Error()).Fatal("Could not subscribe for updates")
		panic(err)
	}

	for {
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
