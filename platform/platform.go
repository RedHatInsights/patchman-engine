package platform

import (
	"app/base"
	"app/base/mqueue"
	"app/base/utils"
	"app/manager/middlewares"
	"encoding/json"
	"fmt"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/redhatinsights/platform-go-middlewares/identity"
	"modernc.org/strutil"
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
	app.Use(middlewares.RequestResponseLogger())
	app.Use(gzip.Gzip(gzip.DefaultCompression))
	initVMaaS(app)
	initRbac(app)

	// Control endpoint handler
	app.POST("/control/upload", mockUploadHandler)
	app.POST("/control/delete", mockDeleteHandler)
	app.POST("/control/sync", mockSyncHandler)
	app.POST("/control/toggle_upload", mockToggleUpload)

	err := utils.RunServer(base.Context, app, 9001)
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

func mockIdentity() string {
	ident := identity.Identity{
		Type:          "User",
		AccountNumber: "0",
	}
	js, err := json.Marshal(&ident)
	if err != nil {
		panic(err)
	}
	return string(strutil.Base64Encode(js))
}

func upload(randomPkgs bool) {
	event := map[string]interface{}{
		"type": "created",
		"host": map[string]interface{}{
			"id":       "00000000-0000-0000-0000-000000000100",
			"account":  "TEST-0000",
			"reporter": "puptoo",
			"tags": []map[string]string{
				{
					"key":   "env",
					"value": "prod",
				},
				{
					"namespace": "satellite",
					"key":       "organization",
					"value":     "rh",
				},
			},
			"system_profile": makeSystemProfile("TEST-0000", randomPkgs),
		},
	}
	msg, err := json.Marshal(event)
	if err != nil {
		panic(err)
	}
	sendMessageToTopic("platform.inventory.events", string(msg))
}

func sendMessageToTopic(topic, message string) {
	writer := mqueue.NewKafkaWriterFromEnv(topic)

	err := writer.WriteMessages(base.Context, mqueue.KafkaMessage{
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
