package vmaas_sync //nolint:golint,stylecheck

import (
	"app/base"
	"app/base/core"
	"app/base/mqueue"
	"app/base/utils"
	"github.com/RedHatInsights/patchman-clients/vmaas"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"os"
	"time"
)

// Should be < 5000
const SyncBatchSize = 1000

var (
	vmaasClient              *vmaas.APIClient
	evalWriter               mqueue.Writer
	enabledRepoBasedReeval   bool
	advisoryPageSize         int
	packagesPageSize         int
	enableRecalcMessagesSend bool
	deleteCulledSystemsLimit int
	enableSyncOnStart        bool
	enableRecalcOnStart      bool
	enableCulledSystemDelete bool
	enableSystemStaling      bool
	enableTurnpikeAuth       bool
)

func configure() {
	core.ConfigureApp()
	traceAPI := utils.GetenvOrFail("LOG_LEVEL") == "trace"

	cfg := vmaas.NewConfiguration()
	cfg.Servers[0].URL = utils.GetenvOrFail("VMAAS_ADDRESS") + base.VMaaSAPIPrefix
	cfg.Debug = traceAPI

	vmaasClient = vmaas.NewAPIClient(cfg)

	evalTopic := utils.GetenvOrFail("EVAL_TOPIC")
	evalWriter = mqueue.WriterFromEnv(evalTopic)
	enabledRepoBasedReeval = utils.GetBoolEnvOrFail("ENABLE_REPO_BASED_RE_EVALUATION")
	enableRecalcMessagesSend = utils.GetBoolEnvOrDefault("ENABLE_RECALC_MESSAGES_SEND", true)
	enableSyncOnStart = utils.GetBoolEnvOrDefault("ENABLE_SYNC_ON_START", false)
	enableRecalcOnStart = utils.GetBoolEnvOrDefault("ENABLE_RECALC_ON_START", false)
	enableCulledSystemDelete = utils.GetBoolEnvOrDefault("ENABLE_CULLED_SYSTEM_DELETE", true)
	enableSystemStaling = utils.GetBoolEnvOrDefault("ENABLE_SYSTEM_STALING", true)
	enableTurnpikeAuth = utils.GetBoolEnvOrDefault("ENABLE_TURNPIKE_AUTH", false)

	advisoryPageSize = utils.GetIntEnvOrDefault("ERRATA_PAGE_SIZE", 500)
	packagesPageSize = utils.GetIntEnvOrDefault("PACKAGES_PAGE_SIZE", 5000)

	deleteCulledSystemsLimit = utils.GetIntEnvOrDefault("DELETE_CULLED_SYSTEMS_LIMIT", 1000)
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
		typ, msg, err := conn.ReadMessage()
		if err != nil {
			utils.Log("err", err.Error()).Error("Failed to retrieve VMaaS websocket message")
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

func websocketHandler(data []byte, _ *websocket.Conn) error {
	text := string(data)
	utils.Log("data", string(data)).Info("Received VMaaS websocket message")

	if text == "webapps-refreshed" {
		err := syncData()
		if err != nil {
			// This probably means programming error, better to exit with nonzero error code, so the error is noticed
			utils.Log("err", err.Error()).Fatal("vmaas data sync failed")
		}

		err = sendReevaluationMessages()
		if err != nil {
			utils.Log("err", err.Error()).Error("re-evaluation sending routine failed")
		}
	}
	return nil
}

func syncData() error {
	err := syncAdvisories()
	if err != nil {
		return errors.Wrap(err, "Failed to sync advisories")
	}

	err = syncRepos()
	if err != nil {
		return errors.Wrap(err, "Failed to sync repos")
	}
	return nil
}

func handleContextCancel(fn func()) {
	go func() {
		<-base.Context.Done()
		utils.Log().Info("stopping vmaas_sync")
		fn()
	}()
}

func waitAndExit() {
	time.Sleep(time.Second) // give some time to close eventual db connections
	os.Exit(0)
}

func syncAndRecalcOnStartIfSet() {
	if enableSyncOnStart {
		err := syncAdvisories()
		if err != nil {
			utils.Log("err", err.Error()).Error("unable to sync advisories on start")
		}
	}

	if enableRecalcOnStart {
		err := sendReevaluationMessages()
		if err != nil {
			utils.Log("err", err.Error()).Error("unable to send reevaluation msgs on start")
		}
	}
}

func RunVmaasSync() {
	handleContextCancel(waitAndExit)
	configure()

	go RunMetrics()

	go runAdminAPI()

	go RunSystemCulling()

	syncAndRecalcOnStartIfSet() // sync advisories and re-calc on start if configured

	// Continually try to reconnect
	for {
		conn, _, err := websocket.DefaultDialer.DialContext(base.Context,
			utils.GetenvOrFail("VMAAS_WS_ADDRESS"), nil)
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
