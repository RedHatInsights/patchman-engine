package vmaas_sync //nolint:golint,stylecheck

import (
	"app/base/utils"
	"github.com/gorilla/websocket"
	"time"
)

type Handler func(data []byte, conn *websocket.Conn) error

func runWebsocket(conn *websocket.Conn, handler Handler) error {
	defer conn.Close()

	err := conn.WriteMessage(websocket.TextMessage, []byte("subscribe-listener"))
	if err != nil {
		utils.Log("err", err.Error()).Fatal("Could not subscribe for updates")
		return err
	}

	for {
		typ, msg, err := conn.ReadMessage()
		if err != nil {
			utils.Log("err", err.Error()).Fatal("Failed to retrieve VMaaS websocket message")
			messagesReceivedCnt.WithLabelValues("error-read-msg").Inc()
			return err
		}
		utils.Log("messageType", typ).Info("websocket message received")

		if typ == websocket.BinaryMessage || typ == websocket.TextMessage {
			err = handler(msg, conn)
			if err != nil {
				messagesReceivedCnt.WithLabelValues("error-handled").Inc()
				return err
			}
			messagesReceivedCnt.WithLabelValues("handled").Inc()
			continue
		}

		if typ == websocket.PingMessage {
			err = conn.WriteMessage(websocket.PongMessage, msg)
			if err != nil {
				messagesReceivedCnt.WithLabelValues("error-ping-pong").Inc()
				return err
			}
			messagesReceivedCnt.WithLabelValues("ping-pong").Inc()
			continue
		}

		if typ == websocket.CloseMessage {
			messagesReceivedCnt.WithLabelValues("close").Inc()
			return nil
		}
		messagesReceivedCnt.WithLabelValues("unhandled").Inc()
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

	go RunMetrics()

	go runDebugAPI()

	go RunSystemCulling()

	// Continually try to reconnect
	for {
		conn, _, err := websocket.DefaultDialer.Dial(utils.GetenvOrFail("VMAAS_WS_ADDRESS"), nil)
		if err != nil {
			utils.Log("err", err.Error()).Fatal("Failed to connect to VMaaS")
		}

		err = runWebsocket(conn, websocketHandler)
		if err != nil {
			utils.Log("err", err.Error()).Error("Websocket error occurred, waiting")
		}
		time.Sleep(2 * time.Second)
	}
}
