package utils

import (
	"fmt"
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

	// API
	PublicPort  int
	PrivatePort int
	MetricsPort int
	MetricsPath string

	// kafka
	KafkaAddress           string
	KafkaSslEnabled        bool
	KafkaSslCert           string
	KafkaUsername          string
	KafkaPassword          string
	EventsTopic            string
	EvalTopic              string
	RemediationUpdateTopic string

	// services
	VmaasAddress   string
	RbacAddress    string
	VmaasWsAddress string
}

func init() {
	initDBFromEnv()
	initAPIFromEnv()
	initKafkaFromEnv()
	initServicesFromEnv()
	if clowder.IsClowderEnabled() {
		initDBFromClowder()
		initAPIromClowder()
		initKafkaFromClowder()
		initServicesFromClowder()
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

func initAPIFromEnv() {
	Cfg.PublicPort = GetIntEnvOrDefault("PUBLIC_PORT", -1)
	Cfg.PrivatePort = GetIntEnvOrDefault("PRIVATE_PORT", -1)
	Cfg.MetricsPort = GetIntEnvOrDefault("METRICS_PORT", -1)
	Cfg.MetricsPath = Getenv("METRICS_PATH", "/metrics")
}

func initKafkaFromEnv() {
	Cfg.KafkaAddress = Getenv("KAFKA_ADDRESS", "")
	Cfg.KafkaSslEnabled = GetBoolEnvOrDefault("ENABLE_KAFKA_SSL", false)
	Cfg.KafkaSslCert = Getenv("KAFKA_SSL_CERT", "")
	Cfg.KafkaUsername = Getenv("KAFKA_USERNAME", "")
	Cfg.KafkaPassword = Getenv("KAFKA_PASSWORD", "")
	Cfg.EventsTopic = Getenv("EVENTS_TOPIC", "")
	Cfg.EvalTopic = Getenv("EVAL_TOPIC", "")
	Cfg.RemediationUpdateTopic = Getenv("REMEDIATIONS_UPDATE_TOPIC", "")
}

func initServicesFromEnv() {
	Cfg.VmaasAddress = Getenv("VMAAS_ADDRESS", "")
	Cfg.RbacAddress = Getenv("RBAC_ADDRESS", "")
	Cfg.VmaasWsAddress = Getenv("VMAAS_WS_ADDRESS", "")
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

func initAPIromClowder() {
	Cfg.PublicPort = *clowder.LoadedConfig.PublicPort
	Cfg.PrivatePort = *clowder.LoadedConfig.PrivatePort
	Cfg.MetricsPort = clowder.LoadedConfig.MetricsPort
	Cfg.MetricsPath = clowder.LoadedConfig.MetricsPath
}

func initKafkaFromClowder() {
	if len(clowder.LoadedConfig.Kafka.Brokers) > 0 {
		kafkaHost := clowder.LoadedConfig.Kafka.Brokers[0].Hostname
		kafkaPort := *clowder.LoadedConfig.Kafka.Brokers[0].Port
		Cfg.KafkaAddress = fmt.Sprintf("%s:%d", kafkaHost, kafkaPort)
		brokerCfg := clowder.LoadedConfig.Kafka.Brokers[0]
		if brokerCfg.Cacert != nil {
			Cfg.KafkaSslEnabled = true
			certPath, err := clowder.LoadedConfig.KafkaCa(brokerCfg)
			if err != nil {
				panic(err)
			}
			Cfg.KafkaSslCert = certPath
			if brokerCfg.Sasl.Username != nil {
				Cfg.KafkaUsername = *brokerCfg.Sasl.Username
				Cfg.KafkaPassword = *brokerCfg.Sasl.Password
			}
		}

		// translate kafka topic names
		if Cfg.EventsTopic != "" {
			Cfg.EventsTopic = clowder.KafkaTopics[Cfg.EventsTopic].Name
		}
		if Cfg.EvalTopic != "" {
			Cfg.EvalTopic = clowder.KafkaTopics[Cfg.EvalTopic].Name
		}
		if Cfg.RemediationUpdateTopic != "" {
			Cfg.RemediationUpdateTopic = clowder.KafkaTopics[Cfg.RemediationUpdateTopic].Name
		}
	}
}

func initServicesFromClowder() {
	for _, endpoint := range clowder.LoadedConfig.Endpoints {
		switch endpoint.App {
		case "vmaas":
			if strings.Contains(endpoint.Name, "webapp") {
				Cfg.VmaasAddress = fmt.Sprintf("http://%s:%d", endpoint.Hostname, endpoint.Port)
			}
		case "rbac":
			Cfg.RbacAddress = fmt.Sprintf("http://%s:%d", endpoint.Hostname, endpoint.Port)
		}
	}
	for _, endpoint := range clowder.LoadedConfig.PrivateEndpoints {
		if endpoint.App == "vmaas" {
			if strings.Contains(endpoint.Name, "websocket") {
				Cfg.VmaasWsAddress = fmt.Sprintf("ws://%s:%d", endpoint.Hostname, endpoint.Port)
			}
		}
	}
}

// PrintClowderParams Print Clowder params to export environment variables.
func PrintClowderParams() {
	if clowder.IsClowderEnabled() {
		// Database
		printDBParams()
		// API
		printAPIParams()
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

func printAPIParams() {
	fmt.Printf("PUBLIC_PORT=%d\n", Cfg.PublicPort)
	fmt.Printf("PRIVATE_PORT=%d\n", Cfg.PrivatePort)
	fmt.Printf("METRICS_PORT=%d\n", Cfg.MetricsPort)
	fmt.Printf("METRICS_PATH=%s\n", Cfg.MetricsPath)
}

func printKafkaParams() {
	fmt.Printf("KAFKA_ADDRESS=%s\n", Cfg.KafkaAddress)
	if Cfg.KafkaSslEnabled {
		fmt.Println("ENABLE_KAFKA_SSL=true")
		fmt.Printf("KAFKA_SSL_CERT=%s\n", Cfg.KafkaSslCert)
		if Cfg.KafkaUsername != "" {
			fmt.Printf("KAFKA_USERNAME=%s\n", Cfg.KafkaUsername)
			fmt.Printf("KAFKA_PASSWORD=%s\n", Cfg.KafkaPassword)
		}
	}
	fmt.Printf("EVENTS_TOPIC=%s\n", Cfg.EventsTopic)
	fmt.Printf("EVAL_TOPIC=%s\n", Cfg.EvalTopic)
	fmt.Printf("REMEDIATIONS_UPDATE_TOPIC=%s\n", Cfg.RemediationUpdateTopic)
}

func printServicesParams() {
	fmt.Printf("VMAAS_ADDRESS=http://%s\n", Cfg.VmaasAddress)
	fmt.Printf("RBAC_ADDRESS=http://%s\n", Cfg.RbacAddress)
	fmt.Printf("VMAAS_WS_ADDRESS=ws://%s\n", Cfg.VmaasWsAddress)
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
