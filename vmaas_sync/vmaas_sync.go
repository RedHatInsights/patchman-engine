package vmaas_sync

import (
	"app/base/utils"
	"app/manager/middlewares"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"time"
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

type Handler func(data []byte, conn *websocket.Conn) error

func runWebsocket(conn *websocket.Conn, handler Handler) error {
	defer conn.Close()

	err := conn.WriteMessage(websocket.TextMessage, []byte("subscribe-listener"))
	if err != nil {
		utils.Log("err", err.Error()).Fatal("Could not subscribe for updates")
		return err
	}

	for {
		messagesReceived.Add(1)
		typ, msg, err := conn.ReadMessage()
		if err != nil {
			utils.Log("err", err.Error()).Fatal("Failed to retrive VMaaS websocket message")
			return err
		}
		if typ == websocket.BinaryMessage || typ == websocket.TextMessage {
			err = handler(msg, conn)
			if err != nil {
				return err
			}
		}
		if typ == websocket.PingMessage {
			err = conn.WriteMessage(websocket.PongMessage, msg)
			if err != nil {
				return err
			}
		}
		if typ == websocket.CloseMessage {
			return nil
		}
	}
}

func websocketHandler(data []byte, conn *websocket.Conn) error {
	text := string(data)
	utils.Log("data", string(data)).Info("Received VMaaS websocket message")

	if text == "webapps-refreshed" {
		err := syncAdvisories()
		if err != nil {
			// This probably means programming error, better to exit with nonzero error code, so the error is noticed
			utils.Log("err", err.Error()).Fatal("Failed to sync advisories")
		}
		// TODO: Cause re-evaluation of systems
	}
	return nil
}

func RunVmaasSync() {
	configure()

	go runMetrics()

	// Continually try to reconnect
	for {
		conn, _, err := websocket.DefaultDialer.Dial(utils.GetenvOrFail("VMAAS_WS_ADDRESS"), nil)
		if err != nil {
			utils.Log("err", err.Error()).Fatal("Failed to connect to VMaaS")
		}

		err = runWebsocket(conn, websocketHandler)
		if err != nil {
			utils.Log("err", err.Error()).Error("Websocket error occured, waiting")
		}
		time.Sleep(2 * time.Second)
	}
}
