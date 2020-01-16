// nolint:lll
// inspired by: https://github.com/RedHatInsights/insights-ingress-go/blob/3ea33a8d793c2154f7cfa12057ca005c5f6031fa/logger/logger.go
//              https://github.com/kdar/logrus-cloudwatchlogs
package utils

import (
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	lc "github.com/redhatinsights/platform-go-middlewares/logging/cloudwatch"
	log "github.com/sirupsen/logrus"
)

// Try to init CloudWatch logging
func trySetupCloudWatchLogging() {
	key := os.Getenv("CW_AWS_ACCESS_KEY_ID")
	if key == "" {
		log.Info("config for aws CloudWatch not loaded")
		return
	}

	secret := GetenvOrFail("CW_AWS_SECRET_ACCESS_KEY")
	region := Getenv("CW_AWS_REGION", "us-east-1")
	group := Getenv("CW_AWS_LOG_GROUP", "platform-dev")

	hostname, err := os.Hostname()
	if err != nil {
		Log("err", err.Error()).Error("unable to get hostname to set CloudWatch log_stream")
		return
	}

	log.SetFormatter(&log.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.999Z",
		FieldMap: log.FieldMap{
			log.FieldKeyTime:  "@timestamp",
			log.FieldKeyLevel: "level",
			log.FieldKeyMsg:   "message",
		},
	})

	cred := credentials.NewStaticCredentials(key, secret, "")
	awsconf := aws.NewConfig().WithRegion(region).WithCredentials(cred)
	hook, err := lc.NewBatchingHook(group, hostname, awsconf, 10*time.Second)
	if err != nil {
		Log("err", err.Error()).Error("unable to setup CloudWatch logging")
		return
	}
	log.AddHook(hook)
	log.Info("CloudWatch logging configured")
}
