package utils

import (
	"fmt"
	"strings"
	"time"

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
	PublicPort      int
	PrivatePort     int
	MetricsPort     int
	MetricsPath     string
	ResponseTimeout time.Duration

	// kafka
	KafkaAddress           string
	KafkaSslEnabled        bool
	KafkaSslCert           string
	KafkaSslSkipVerify     bool
	KafkaSaslType          *string
	KafkaUsername          string
	KafkaPassword          string
	KafkaGroup             string
	KafkaReaderMinBytes    int
	KafkaReaderMaxBytes    int
	KafkaReaderMaxAttempts int
	KafkaWriterMaxAttempts int
	EventsTopic            string
	EvalTopic              string
	PayloadTrackerTopic    string
	RemediationUpdateTopic string
	NotificationsTopic     string

	// services
	VmaasAddress string
	RbacAddress  string

	// cloudwatch
	CloudWatchAccessKeyID     string
	CloudWatchSecretAccesskey string
	CloudWatchRegion          string
	CloudWatchLogGroup        string

	// prometheus pushgateway
	PrometheusPushGateway string
}

func init() {
	// topics have to be set first, they may be translated via clowder
	initTopicsFromEnv()
	if clowder.IsClowderEnabled() {
		initDBFromClowder()
		initAPIromClowder()
		initKafkaFromClowder()
		initServicesFromClowder()
		initCloudwatchFromClowder()
	}
	// init non-clowder setting and allow local overwrites
	initDBFromEnv()
	initKafkaFromEnv()
	initServicesFromEnv()
	initPrometheusPushGatewayFromEnv()
}

func initDBFromEnv() {
	Cfg.DBHost = Getenv("DB_HOST", Cfg.DBHost)
	Cfg.DBPort = GetIntEnvOrDefault("DB_PORT", Cfg.DBPort)
	Cfg.DBSslRootCert = Getenv("DB_SSLROOTCERT", Cfg.DBSslRootCert)
	Cfg.DBUser = Getenv("DB_USER", Cfg.DBUser)
	Cfg.DBPassword = Getenv("DB_PASSWD", Cfg.DBPassword)
	Cfg.DBDebug = GetBoolEnvOrDefault("DB_DEBUG", false)
	Cfg.DBStatementTimeoutMs = GetIntEnvOrDefault("DB_STATEMENT_TIMEOUT_MS", 0)
	Cfg.DBMaxConnections = GetIntEnvOrDefault("DB_MAX_CONNECTIONS", 250)
	Cfg.DBMaxIdleConnections = GetIntEnvOrDefault("DB_MAX_IDLE_CONNECTIONS", 50)
	Cfg.DBMaxConnectionLifetimeS = GetIntEnvOrDefault("DB_MAX_CONNECTION_LIFETIME_S", 60)
}

func initKafkaFromEnv() {
	Cfg.KafkaSslCert = Getenv("KAFKA_SSL_CERT", Cfg.KafkaSslCert)
	Cfg.KafkaSslSkipVerify = GetBoolEnvOrDefault("KAFKA_SSL_SKIP_VERIFY", false)
	Cfg.KafkaUsername = Getenv("KAFKA_USERNAME", Cfg.KafkaUsername)
	Cfg.KafkaPassword = Getenv("KAFKA_PASSWORD", Cfg.KafkaPassword)
	Cfg.KafkaGroup = Getenv("KAFKA_GROUP", "")
	Cfg.KafkaReaderMinBytes = GetIntEnvOrDefault("KAFKA_READER_MIN_BYTES", 1)
	Cfg.KafkaReaderMaxBytes = GetIntEnvOrDefault("KAFKA_READER_MAX_BYTES", 1e6)
	Cfg.KafkaReaderMaxAttempts = GetIntEnvOrDefault("KAFKA_READER_MAX_ATTEMPTS", 3)
	Cfg.KafkaWriterMaxAttempts = GetIntEnvOrDefault("KAFKA_WRITER_MAX_ATTEMPTS", 10)
}

func initTopicsFromEnv() {
	Cfg.EventsTopic = Getenv("EVENTS_TOPIC", "")
	Cfg.EvalTopic = Getenv("EVAL_TOPIC", "")
	Cfg.PayloadTrackerTopic = Getenv("PAYLOAD_TRACKER_TOPIC", "")
	Cfg.RemediationUpdateTopic = Getenv("REMEDIATIONS_UPDATE_TOPIC", "")
	Cfg.NotificationsTopic = Getenv("NOTIFICATIONS_TOPIC", "")
}

func initServicesFromEnv() {
	Cfg.VmaasAddress = Getenv("VMAAS_ADDRESS", Cfg.VmaasAddress)
	Cfg.RbacAddress = Getenv("RBAC_ADDRESS", Cfg.RbacAddress)
}

func initDBFromClowder() {
	Cfg.DBHost = clowder.LoadedConfig.Database.Hostname
	Cfg.DBName = clowder.LoadedConfig.Database.Name
	Cfg.DBPort = clowder.LoadedConfig.Database.Port
	Cfg.DBSslMode = clowder.LoadedConfig.Database.SslMode
	if clowder.LoadedConfig.Database.RdsCa != nil {
		if strings.HasPrefix(*clowder.LoadedConfig.Database.RdsCa, "-----BEGIN CERTIFICATE-----") {
			certPath, err := clowder.LoadedConfig.RdsCa()
			if err != nil {
				panic(err)
			}
			Cfg.DBSslRootCert = certPath
		} else {
			Cfg.DBSslRootCert = *clowder.LoadedConfig.Database.RdsCa
		}
	}
	Cfg.DBAdminUser = clowder.LoadedConfig.Database.AdminUsername
	Cfg.DBAdminPassword = clowder.LoadedConfig.Database.AdminPassword
	Cfg.DBUser = clowder.LoadedConfig.Database.Username
	Cfg.DBPassword = clowder.LoadedConfig.Database.Password
}

func initAPIromClowder() {
	Cfg.PublicPort = *clowder.LoadedConfig.PublicPort
	Cfg.PrivatePort = *clowder.LoadedConfig.PrivatePort
	Cfg.MetricsPort = clowder.LoadedConfig.MetricsPort
	Cfg.MetricsPath = clowder.LoadedConfig.MetricsPath
	Cfg.ResponseTimeout = time.Duration(GetIntEnvOrDefault("RESPONSE_TIMEOUT", 60))
}

func initKafkaFromClowder() {
	if len(clowder.LoadedConfig.Kafka.Brokers) > 0 {
		Cfg.KafkaSaslType = nil
		brokerCfg := clowder.LoadedConfig.Kafka.Brokers[0]
		kafkaHost := brokerCfg.Hostname
		kafkaPort := *brokerCfg.Port
		Cfg.KafkaAddress = fmt.Sprintf("%s:%d", kafkaHost, kafkaPort)
		if brokerCfg.Cacert != nil && len(*brokerCfg.Cacert) > 0 {
			Cfg.KafkaSslEnabled = true
			if strings.HasPrefix(*brokerCfg.Cacert, "-----BEGIN CERTIFICATE-----") {
				certPath, err := clowder.LoadedConfig.KafkaCa(brokerCfg)
				if err != nil {
					panic(err)
				}
				Cfg.KafkaSslCert = certPath
			} else {
				Cfg.KafkaSslCert = *brokerCfg.Cacert
			}
			if brokerCfg.Sasl.Username != nil {
				Cfg.KafkaUsername = *brokerCfg.Sasl.Username
				Cfg.KafkaPassword = *brokerCfg.Sasl.Password
				Cfg.KafkaSaslType = brokerCfg.Sasl.SaslMechanism
			}
		}

		// translate kafka topic names
		translateTopic(&Cfg.EventsTopic)
		translateTopic(&Cfg.EvalTopic)
		translateTopic(&Cfg.PayloadTrackerTopic)
		translateTopic(&Cfg.RemediationUpdateTopic)
		translateTopic(&Cfg.NotificationsTopic)
	}
}

func initServicesFromClowder() {
	webappName := "webapp-service"
	if GetBoolEnvOrDefault("USE_VMAAS_GO", false) {
		webappName = "webapp-go"
	}
	for _, endpoint := range clowder.LoadedConfig.Endpoints {
		switch endpoint.App {
		case "vmaas":
			if strings.Contains(endpoint.Name, webappName) {
				Cfg.VmaasAddress = fmt.Sprintf("http://%s:%d", endpoint.Hostname, endpoint.Port)
			}
		case "rbac":
			Cfg.RbacAddress = fmt.Sprintf("http://%s:%d", endpoint.Hostname, endpoint.Port)
		}
	}
}

func initCloudwatchFromClowder() {
	cwCfg := clowder.LoadedConfig.Logging.Cloudwatch
	if cwCfg != nil {
		Cfg.CloudWatchAccessKeyID = cwCfg.AccessKeyId
		Cfg.CloudWatchSecretAccesskey = cwCfg.SecretAccessKey
		Cfg.CloudWatchRegion = cwCfg.Region
		Cfg.CloudWatchLogGroup = cwCfg.LogGroup
	}
}

func initPrometheusPushGatewayFromEnv() {
	Cfg.PrometheusPushGateway = Getenv("PROMETHEUS_PUSHGATEWAY", "pushgateway")
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
	fmt.Printf("PAYLOAD_TRACKER_TOPIC=%s\n", Cfg.PayloadTrackerTopic)
	fmt.Printf("REMEDIATIONS_UPDATE_TOPIC=%s\n", Cfg.RemediationUpdateTopic)
	fmt.Printf("NOTIFICATIONS_TOPIC=%s\n", Cfg.NotificationsTopic)
}

func printServicesParams() {
	fmt.Printf("VMAAS_ADDRESS=http://%s\n", Cfg.VmaasAddress)
	fmt.Printf("RBAC_ADDRESS=http://%s\n", Cfg.RbacAddress)
}

func printCloudwatchParams() {
	cwCfg := clowder.LoadedConfig.Logging.Cloudwatch
	if cwCfg == nil {
		fmt.Println("No Cloudwatch logging found")
		return
	}
	fmt.Printf("CW_AWS_ACCESS_KEY_ID=%s\n", Cfg.CloudWatchAccessKeyID)
	fmt.Printf("CW_AWS_SECRET_ACCESS_KEY=%s\n", Cfg.CloudWatchSecretAccesskey)
	fmt.Printf("CW_AWS_REGION=%s\n", Cfg.CloudWatchRegion)
	fmt.Printf("CW_AWS_LOG_GROUP=%s\n", Cfg.CloudWatchLogGroup)
}

func translateTopic(topic *string) {
	if v, ok := clowder.KafkaTopics[*topic]; ok {
		*topic = v.Name
	}
}
