package utils

import (
	"fmt"
	"os"
	"strings"

	clowder "github.com/redhatinsights/app-common-go/pkg/api/v1"
)

var Cfg = Config{}

type Config struct {
	// database
	DBHost                   string
	DBName                   string
	DBPort                   int
	DBSslMode                string
	DBSslRootCert            string
	DBAdminUser              string
	DBAdminPassword          string
	DBUser                   string
	DBPassword               string
	DBDebug                  bool
	DBStatementTimeoutMs     int
	DBMaxConnections         int
	DBMaxIdleConnections     int
	DBMaxConnectionLifetimeS int
}

func init() {
	initDBFromEnv()
	if clowder.IsClowderEnabled() {
		initDBFromClowder()
	}
}

func initDBFromEnv() {
	Cfg.DBHost = Getenv("DB_HOST", "UNSET")
	Cfg.DBName = Getenv("DB_NAME", "UNSET")
	Cfg.DBPort = GetIntEnvOrDefault("DB_PORT", -1)
	Cfg.DBSslMode = Getenv("DB_SSLMODE", "UNSET")
	Cfg.DBSslRootCert = Getenv("DB_SSLROOTCERT", "")
	Cfg.DBAdminUser = Getenv("DB_ADMIN_USER", "")
	Cfg.DBAdminPassword = Getenv("DB_ADMIN_PASSWD", "")
	Cfg.DBUser = Getenv("DB_USER", "UNSET")
	Cfg.DBPassword = Getenv("DB_PASSWD", "UNSET")
	Cfg.DBDebug = GetBoolEnvOrDefault("DB_DEBUG", false)
	Cfg.DBStatementTimeoutMs = GetIntEnvOrDefault("DB_STATEMENT_TIMEOUT_MS", 0)
	Cfg.DBMaxConnections = GetIntEnvOrDefault("DB_MAX_CONNECTIONS", 250)
	Cfg.DBMaxIdleConnections = GetIntEnvOrDefault("DB_MAX_IDLE_CONNECTIONS", 50)
	Cfg.DBMaxConnectionLifetimeS = GetIntEnvOrDefault("DB_MAX_CONNECTION_LIFETIME_S", 60)
}

func initDBFromClowder() {
	Cfg.DBHost = clowder.LoadedConfig.Database.Hostname
	Cfg.DBName = clowder.LoadedConfig.Database.Name
	Cfg.DBPort = clowder.LoadedConfig.Database.Port
	Cfg.DBSslMode = clowder.LoadedConfig.Database.SslMode
	if clowder.LoadedConfig.Database.RdsCa != nil {
		certPath, err := clowder.LoadedConfig.RdsCa()
		if err != nil {
			panic(err)
		}
		Cfg.DBSslRootCert = certPath
	}
	Cfg.DBAdminUser = clowder.LoadedConfig.Database.AdminUsername
	Cfg.DBAdminPassword = clowder.LoadedConfig.Database.AdminPassword
}

// PrintClowderParams Print Clowder params to export environment variables.
func PrintClowderParams() {
	if clowder.IsClowderEnabled() {
		// Database
		printDBParams()
		// API
		fmt.Printf("PUBLIC_PORT=%d\n", *clowder.LoadedConfig.PublicPort)
		fmt.Printf("PRIVATE_PORT=%d\n", *clowder.LoadedConfig.PrivatePort)
		fmt.Printf("METRICS_PORT=%d\n", clowder.LoadedConfig.MetricsPort)
		fmt.Printf("METRICS_PATH=%s\n", clowder.LoadedConfig.MetricsPath)
		// Kafka
		printKafkaParams()
		// Services (vmaas, rbac)
		printServicesParams()
		// Cloudwatch logging
		printCloudwatchParams()
	}
}

func printDBParams() {
	fmt.Printf("DB_ADMIN_USER=%s\n", Cfg.DBAdminUser)
	fmt.Printf("DB_ADMIN_PASSWD=%s\n", Cfg.DBAdminPassword)
	fmt.Printf("DB_HOST=%s\n", Cfg.DBHost)
	fmt.Printf("DB_NAME=%s\n", Cfg.DBName)
	fmt.Printf("DB_PORT=%d\n", Cfg.DBPort)
	fmt.Printf("DB_SSLMODE=%s\n", Cfg.DBSslMode)
	fmt.Printf("DB_SSLROOTCERT=%s\n", Cfg.DBSslRootCert)
}

func printKafkaParams() {
	if len(clowder.LoadedConfig.Kafka.Brokers) > 0 {
		kafkaHost := clowder.LoadedConfig.Kafka.Brokers[0].Hostname
		kafkaPort := *clowder.LoadedConfig.Kafka.Brokers[0].Port
		fmt.Printf("KAFKA_ADDRESS=%s:%d\n", kafkaHost, kafkaPort)
		brokerCfg := clowder.LoadedConfig.Kafka.Brokers[0]
		if brokerCfg.Cacert != nil {
			fmt.Println("ENABLE_KAFKA_SSL=true")
			certPath, err := clowder.LoadedConfig.KafkaCa(brokerCfg)
			if err != nil {
				panic(err)
			}
			fmt.Printf("KAFKA_SSL_CERT=%s\n", certPath)
			if brokerCfg.Sasl.Username != nil {
				fmt.Printf("KAFKA_USERNAME=%s\n", *brokerCfg.Sasl.Username)
				fmt.Printf("KAFKA_PASSWORD=%s\n", *brokerCfg.Sasl.Password)
			}
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
			if strings.Contains(endpoint.Name, "webapp") {
				fmt.Printf("VMAAS_ADDRESS=http://%s:%d\n", endpoint.Hostname, endpoint.Port)
			}
		case "rbac":
			fmt.Printf("RBAC_ADDRESS=http://%s:%d\n", endpoint.Hostname, endpoint.Port)
		}
	}

	for _, endpoint := range clowder.LoadedConfig.PrivateEndpoints {
		if endpoint.App == "vmaas" {
			if strings.Contains(endpoint.Name, "websocket") {
				fmt.Printf("VMAAS_WS_ADDRESS=ws://%s:%d\n", endpoint.Hostname, endpoint.Port)
			}
		}
	}
}

func printCloudwatchParams() {
	cwCfg := clowder.LoadedConfig.Logging.Cloudwatch
	if cwCfg == nil {
		fmt.Println("No Cloudwatch logging found")
		return
	}
	fmt.Printf("CW_AWS_ACCESS_KEY_ID=%s\n", cwCfg.AccessKeyId)
	fmt.Printf("CW_AWS_SECRET_ACCESS_KEY=%s\n", cwCfg.SecretAccessKey)
	fmt.Printf("CW_AWS_REGION=%s\n", cwCfg.Region)
	fmt.Printf("CW_AWS_LOG_GROUP=%s\n", cwCfg.LogGroup)
}
