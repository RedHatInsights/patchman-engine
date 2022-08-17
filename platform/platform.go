package platform

import (
	"app/base"
	"app/base/mqueue"
	"app/base/utils"
	"app/manager/middlewares"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/redhatinsights/platform-go-middlewares/identity"
	"modernc.org/strutil"
)

var websockets []chan string
var runUploadLoop = false

const uploadEvent = `{
		"type": "created",
		"host": {
			"id":       "00000000-0000-0000-0000-000000000100",
			"account":  "TEST-0000",
			"reporter": "puptoo",
			"tags": [
				{
					"key":   "env",
					"value": "prod"
				},
				{
					"namespace": "satellite",
					"key":       "organization",
					"value":     "rh"
				}
			],
			"system_profile": %s
		},
    "platform_metadata": {
      "request_id": "ingress-service-5f79d54bf-q5jh6/iDl0gmf6Qw-071711",
      "custom_metadata": {
        "yum_updates": {
          "releasever": "8",
          "basearch": "x86_64",
          "update_list": {
            "bash-0:4.4.20-1.el8_4.x86_64": {
              "available_updates": [
                {
                  "package": "bash-0:4.4.20-3.el8.x86_64",
                  "repository": "rhel-8-for-x86_64-baseos-rpms",
                  "basearch": "x86_64",
                  "releasever": "8",
                  "erratum": "RHBA-2022:1993"
                },
                {
                  "package": "bash-0:4.4.20-3.el8.x86_64",
                  "repository": "ubi-8-baseos",
                  "basearch": "x86_64",
                  "releasever": "8",
                  "erratum": "RHBA-2022:1993"
                },
                {
                  "package": "bash-0:4.4.23-1.fc28.x86_64",
                  "repository": "local",
                  "basearch": "x86_64",
                  "releasever": "8"
                }
              ]
            },
            "curl-0:7.61.1-18.el8_4.2.x86_64": {
              "available_updates": [
                {
                  "package": "curl-0:7.61.1-22.el8.x86_64",
                  "repository": "rhel-8-for-x86_64-baseos-rpms",
                  "basearch": "x86_64",
                  "releasever": "8",
                  "erratum": "RHSA-2021:4511"
                },
                {
                  "package": "curl-0:7.61.1-22.el8.x86_64",
                  "repository": "ubi-8-baseos",
                  "basearch": "x86_64",
                  "releasever": "8",
                  "erratum": "RHSA-2021:4511"
                }
              ]
            }
          },
          "metadata_time": "2022-05-30T14:00:25Z"
        }
      }
    }
	}`

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
	packages := makeSystemProfile("TEST-0000", randomPkgs)
	pkgJSON, err := json.Marshal(packages)
	if err != nil {
		panic(err)
	}
	event := fmt.Sprintf(uploadEvent, pkgJSON)
	SendMessageToTopic("platform.inventory.events", event)
}

func SendMessageToTopic(topic, message string) {
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
	SendMessageToTopic("platform.inventory.events", string(msg))
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
