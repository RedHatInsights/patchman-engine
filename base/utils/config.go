package utils

import (
	"fmt"
	clowder "github.com/redhatinsights/app-common-go/pkg/api/v1"
	"os"
)

// IsClowderEnabled Check env variable CLOWDER_ENABLED = "true".
func IsClowderEnabled() bool {
	clowderEnabled := GetBoolEnvOrDefault("CLOWDER_ENABLED", false)
	return clowderEnabled
}

// PrintClowderParams Print Clowder params to export environment variables.
func PrintClowderParams() {
	fmt.Println("Trying to export variables from Clowder")
	if IsClowderEnabled() {
		fmt.Println("Clowder config enabled, exporting variables..")
		// Database
		fmt.Printf("DB_ADMIN_USER=%s\n", clowder.LoadedConfig.Database.AdminUsername)
		fmt.Printf("DB_ADMIN_PASSWD=%s\n", clowder.LoadedConfig.Database.AdminPassword)
		fmt.Printf("DB_HOST=%s\n", clowder.LoadedConfig.Database.Hostname)
		fmt.Printf("DB_NAME=%s\n", clowder.LoadedConfig.Database.Name)
		fmt.Printf("DB_PORT=%d\n", clowder.LoadedConfig.Database.Port)
		fmt.Printf("DB_SSLMODE=%s\n", clowder.LoadedConfig.Database.SslMode)
		// API
		fmt.Printf("PUBLIC_PORT=%d\n", *clowder.LoadedConfig.PublicPort)
		fmt.Printf("PRIVATE_PORT=%d\n", *clowder.LoadedConfig.PrivatePort)
		fmt.Printf("METRICS_PORT=%d\n", clowder.LoadedConfig.MetricsPort)
		fmt.Printf("METRICS_PATH=%s\n", clowder.LoadedConfig.MetricsPath)
		// Kafka
		printKafkaParams()
		// Services (vmaas, rbac)
		printServicesParams()
		fmt.Println("...done")
	} else {
		fmt.Println("Clowder not enabled")
	}
}

func printKafkaParams() {
	if len(clowder.LoadedConfig.Kafka.Brokers) > 0 {
		kafkaHost := clowder.LoadedConfig.Kafka.Brokers[0].Hostname
		kafkaPort := *clowder.LoadedConfig.Kafka.Brokers[0].Port
		fmt.Printf("KAFKA_ADDRESS=%s:%d\n", kafkaHost, kafkaPort)
		kafkaCert := clowder.LoadedConfig.Kafka.Brokers[0].Cacert
		if kafkaCert != nil {
			fmt.Printf("KAFKA_SSL_CERT=%s\n", *kafkaCert)
		}
		topics := []string{"EVENTS_TOPIC", "EVAL_TOPIC", "REMEDIATIONS_UPDATE_TOPIC"}
		for _, topic := range topics {
			topicValue := os.Getenv(topic)
			if len(topicValue) > 0 {
				fmt.Printf("%s=%s\n", topic, clowder.KafkaTopics[topicValue].Name)
			}
		}
	}
}

func printServicesParams() {
	for _, endpoint := range clowder.LoadedConfig.Endpoints {
		switch endpoint.App {
		case "vmaas":
			if endpoint.Name == "webapp" {
				fmt.Printf("VMAAS_ADDRESS=http://%s:%d\n", endpoint.Hostname, endpoint.Port)
			}
		case "rbac":
			fmt.Printf("RBAC_ADDRESS=http://%s:%d\n", endpoint.Hostname, endpoint.Port)
		}
	}

	for _, endpoint := range clowder.LoadedConfig.PrivateEndpoints {
		if endpoint.App == "vmaas" {
			if endpoint.Name == "websocket" {
				fmt.Printf("VMAAS_WS_ADDRESS=ws://%s:%d\n", endpoint.Hostname, endpoint.Port)
			}
		}
	}
}
