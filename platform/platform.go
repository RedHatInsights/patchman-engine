package platform

import (
	"app/base"
	"app/base/mqueue"
	"app/base/utils"
	"app/base/vmaas"
	"app/manager/middlewares"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/redhatinsights/identity"
	"modernc.org/strutil"
)

var runUploadLoop = false

const uploadEvent = `{
		"type": "created",
		"host": {
			"id":       "00000000-0000-0000-0000-000000000100",
			"org_id":   "org_1",
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
		"yum_updates_s3url": "http://platform:9001/yum_updates",
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

var yumUpdates = `{
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
}`

func platformMock() {
	utils.LogInfo("Platform mock starting")
	app := gin.New()
	app.Use(middlewares.RequestResponseLogger())
	app.Use(gzip.Gzip(gzip.DefaultCompression))
	initVMaaS(app)
	initRbac(app)

	// Control endpoint handler
	app.POST("/control/upload", mockUploadHandler)
	app.POST("/control/delete", mockDeleteHandler)
	app.POST("/control/toggle_upload", mockToggleUpload)

	// Mock yum_updates_s3
	app.GET("/yum_updates", mockYumUpdatesS3)

	err := utils.RunServer(base.Context, app, 9001)
	if err != nil {
		panic(err)
	}
}

func mockIdentity() string {
	ident := identity.Identity{
		Type:  "User",
		OrgID: "org_1",
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
	utils.LogInfo("Mocking platform upload event")
	upload(false)
	c.Status(http.StatusOK)
}

func mockDeleteHandler(c *gin.Context) {
	utils.LogInfo("Mocking platform delete event")

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

func mockYumUpdatesS3(c *gin.Context) {
	utils.LogInfo("Mocking S3 for providing yum updates")
	updates := vmaas.UpdatesV3Response{}
	if err := json.Unmarshal([]byte(yumUpdates), &updates); err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, &updates)
}

func uploader() {
	var i int
	for {
		if runUploadLoop {
			upload(true)
			i++
			utils.LogInfo("iteration", i, "upload loop running")
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
