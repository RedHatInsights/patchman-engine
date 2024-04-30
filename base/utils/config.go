package utils

import (
	"fmt"
	"strings"
	"time"

	clowder "github.com/redhatinsights/app-common-go/pkg/api/v1"
)

var CoreCfg = coreConfig{}

type coreConfig struct {
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
	DBReadReplicaHost        string
	DBReadReplicaPort        int
	DBReadReplicaEnabled     bool
	DBWorkMem                int

	// API
	PublicPort         int
	PrivatePort        int
	MetricsPort        int
	MetricsPath        string
	ResponseTimeout    time.Duration
	MaxRequestBodySize int64
	MaxHeaderCount     int
	MaxGinConnections  int
	Ratelimit          int
	LimitPageSize      bool

	// kafka
	KafkaServers           []string
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
	TemplateTopic          string

	// services
	VmaasAddress                  string
	RbacAddress                   string
	ManagerPrivateAddress         string
	ListenerPrivateAddress        string
	EvaluatorUploadPrivateAddress string
	EvaluatorRecalcPrivateAddress string

	// cloudwatch
	CloudWatchAccessKeyID     string
	CloudWatchSecretAccesskey string
	CloudWatchRegion          string
	CloudWatchLogGroup        string

	// prometheus pushgateway
	PrometheusPushGateway string

	// profiler
	ProfilerEnabled bool
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
	initProfilerFromEnv()
}

func initDBFromEnv() {
	CoreCfg.DBHost = Getenv("DB_HOST", CoreCfg.DBHost)
	CoreCfg.DBPort = GetIntEnvOrDefault("DB_PORT", CoreCfg.DBPort)
	CoreCfg.DBSslRootCert = Getenv("DB_SSLROOTCERT", CoreCfg.DBSslRootCert)
	CoreCfg.DBUser = Getenv("DB_USER", CoreCfg.DBUser)
	CoreCfg.DBPassword = Getenv("DB_PASSWD", CoreCfg.DBPassword)
	CoreCfg.DBDebug = GetBoolEnvOrDefault("DB_DEBUG", false)
	CoreCfg.DBStatementTimeoutMs = GetIntEnvOrDefault("DB_STATEMENT_TIMEOUT_MS", 0)
	CoreCfg.DBMaxConnections = GetIntEnvOrDefault("DB_MAX_CONNECTIONS", 250)
	CoreCfg.DBMaxIdleConnections = GetIntEnvOrDefault("DB_MAX_IDLE_CONNECTIONS", 50)
	CoreCfg.DBMaxConnectionLifetimeS = GetIntEnvOrDefault("DB_MAX_CONNECTION_LIFETIME_S", 60)
	CoreCfg.DBReadReplicaEnabled = GetBoolEnvOrDefault("DB_READ_REPLICA_ENABLED", false)
	if CoreCfg.DBReadReplicaEnabled {
		CoreCfg.DBReadReplicaHost = Getenv("DB_HOST_READ_REPLICA", "")
		CoreCfg.DBReadReplicaPort = GetIntEnvOrDefault("DB_PORT_READ_REPLICA", 0)
	}
	CoreCfg.DBWorkMem = GetIntEnvOrDefault("DB_WORK_MEM", 4096) // 4MB is DB default
}

func initKafkaFromEnv() {
	CoreCfg.KafkaSslCert = Getenv("KAFKA_SSL_CERT", CoreCfg.KafkaSslCert)
	CoreCfg.KafkaSslSkipVerify = GetBoolEnvOrDefault("KAFKA_SSL_SKIP_VERIFY", false)
	CoreCfg.KafkaUsername = Getenv("KAFKA_USERNAME", CoreCfg.KafkaUsername)
	CoreCfg.KafkaPassword = Getenv("KAFKA_PASSWORD", CoreCfg.KafkaPassword)
	CoreCfg.KafkaGroup = Getenv("KAFKA_GROUP", "")
	CoreCfg.KafkaReaderMinBytes = GetIntEnvOrDefault("KAFKA_READER_MIN_BYTES", 1)
	CoreCfg.KafkaReaderMaxBytes = GetIntEnvOrDefault("KAFKA_READER_MAX_BYTES", 1e6)
	CoreCfg.KafkaReaderMaxAttempts = GetIntEnvOrDefault("KAFKA_READER_MAX_ATTEMPTS", 3)
	CoreCfg.KafkaWriterMaxAttempts = GetIntEnvOrDefault("KAFKA_WRITER_MAX_ATTEMPTS", 10)
}

func initTopicsFromEnv() {
	CoreCfg.EventsTopic = Getenv("EVENTS_TOPIC", "")
	CoreCfg.EvalTopic = Getenv("EVAL_TOPIC", "")
	CoreCfg.PayloadTrackerTopic = Getenv("PAYLOAD_TRACKER_TOPIC", "")
	CoreCfg.RemediationUpdateTopic = Getenv("REMEDIATIONS_UPDATE_TOPIC", "")
	CoreCfg.NotificationsTopic = Getenv("NOTIFICATIONS_TOPIC", "")
	CoreCfg.TemplateTopic = Getenv("TEMPLATE_TOPIC", "")
}

func initServicesFromEnv() {
	CoreCfg.VmaasAddress = Getenv("VMAAS_ADDRESS", CoreCfg.VmaasAddress)
	CoreCfg.RbacAddress = Getenv("RBAC_ADDRESS", CoreCfg.RbacAddress)
}

func initDBFromClowder() {
	CoreCfg.DBHost = clowder.LoadedConfig.Database.Hostname
	CoreCfg.DBName = clowder.LoadedConfig.Database.Name
	CoreCfg.DBPort = clowder.LoadedConfig.Database.Port
	CoreCfg.DBSslMode = clowder.LoadedConfig.Database.SslMode
	if clowder.LoadedConfig.Database.RdsCa != nil {
		if strings.HasPrefix(*clowder.LoadedConfig.Database.RdsCa, "-----BEGIN CERTIFICATE-----") {
			certPath, err := clowder.LoadedConfig.RdsCa()
			if err != nil {
				panic(err)
			}
			CoreCfg.DBSslRootCert = certPath
		} else {
			CoreCfg.DBSslRootCert = *clowder.LoadedConfig.Database.RdsCa
		}
	}
	CoreCfg.DBAdminUser = clowder.LoadedConfig.Database.AdminUsername
	CoreCfg.DBAdminPassword = clowder.LoadedConfig.Database.AdminPassword
	CoreCfg.DBUser = clowder.LoadedConfig.Database.Username
	CoreCfg.DBPassword = clowder.LoadedConfig.Database.Password
}

func initAPIromClowder() {
	CoreCfg.PublicPort = *clowder.LoadedConfig.PublicPort
	CoreCfg.PrivatePort = *clowder.LoadedConfig.PrivatePort
	CoreCfg.MetricsPort = clowder.LoadedConfig.MetricsPort
	CoreCfg.MetricsPath = clowder.LoadedConfig.MetricsPath
	CoreCfg.ResponseTimeout = time.Duration(GetIntEnvOrDefault("RESPONSE_TIMEOUT", 60))
	CoreCfg.MaxRequestBodySize = GetInt64EnvOrDefault("MAX_REQUEST_BODY_SIZE", 1*1024*1024)
	CoreCfg.MaxHeaderCount = GetIntEnvOrDefault("MAX_HEADER_COUNT", 50)
	CoreCfg.MaxGinConnections = GetIntEnvOrDefault("MAX_GIN_CONNECTIONS", 50)
	CoreCfg.Ratelimit = GetIntEnvOrDefault("RATELIMIT", 100)
	CoreCfg.LimitPageSize = GetBoolEnvOrDefault("LIMIT_PAGE_SIZE", true)
}

func initKafkaFromClowder() {
	if len(clowder.LoadedConfig.Kafka.Brokers) > 0 {
		CoreCfg.KafkaSaslType = nil
		CoreCfg.KafkaServers = clowder.KafkaServers
		brokerCfg := clowder.LoadedConfig.Kafka.Brokers[0]
		kafkaHost := brokerCfg.Hostname
		kafkaPort := *brokerCfg.Port
		CoreCfg.KafkaAddress = fmt.Sprintf("%s:%d", kafkaHost, kafkaPort)
		if brokerCfg.SecurityProtocol != nil && strings.Contains(*brokerCfg.SecurityProtocol, "SSL") {
			CoreCfg.KafkaSslEnabled = true
		}
		if brokerCfg.Cacert != nil && len(*brokerCfg.Cacert) > 0 {
			if strings.HasPrefix(*brokerCfg.Cacert, "-----BEGIN CERTIFICATE-----") {
				certPath, err := clowder.LoadedConfig.KafkaCa(brokerCfg)
				if err != nil {
					panic(err)
				}
				CoreCfg.KafkaSslCert = certPath
			} else {
				CoreCfg.KafkaSslCert = *brokerCfg.Cacert
			}
		}
		if brokerCfg.Sasl != nil && brokerCfg.Sasl.Username != nil {
			CoreCfg.KafkaUsername = *brokerCfg.Sasl.Username
			CoreCfg.KafkaPassword = *brokerCfg.Sasl.Password
			CoreCfg.KafkaSaslType = brokerCfg.Sasl.SaslMechanism
		}

		// translate kafka topic names
		translateTopic(&CoreCfg.EventsTopic)
		translateTopic(&CoreCfg.EvalTopic)
		translateTopic(&CoreCfg.PayloadTrackerTopic)
		translateTopic(&CoreCfg.RemediationUpdateTopic)
		translateTopic(&CoreCfg.NotificationsTopic)
		translateTopic(&CoreCfg.TemplateTopic)
	}
}

type Endpoint clowder.DependencyEndpoint
type PrivateEndpoint clowder.PrivateDependencyEndpoint

func initServicesFromClowder() {
	webappName := "webapp-service"
	if GetBoolEnvOrDefault("USE_VMAAS_GO", false) {
		webappName = "webapp-go"
	}
	for _, endpoint := range clowder.LoadedConfig.Endpoints {
		endpoint := endpoint
		switch endpoint.App {
		case "vmaas":
			if strings.Contains(endpoint.Name, webappName) {
				CoreCfg.VmaasAddress = (*Endpoint)(&endpoint).buildURL()
			}
		case "rbac":
			CoreCfg.RbacAddress = (*Endpoint)(&endpoint).buildURL()
		}
	}

	for _, e := range clowder.LoadedConfig.PrivateEndpoints {
		e := e // re-assign iteration variable to use a new memory pointer
		if e.App == "patchman" {
			switch e.Name {
			case "manager":
				CoreCfg.ManagerPrivateAddress = (*PrivateEndpoint)(&e).buildURL()
			case "listener":
				CoreCfg.ListenerPrivateAddress = (*PrivateEndpoint)(&e).buildURL()
			case "evaluator-upload":
				CoreCfg.EvaluatorUploadPrivateAddress = (*PrivateEndpoint)(&e).buildURL()
			case "evaluator-recalc":
				CoreCfg.EvaluatorRecalcPrivateAddress = (*PrivateEndpoint)(&e).buildURL()
			}
		}
	}
}

func (e *Endpoint) buildURL() string {
	port := e.Port
	scheme := "http"
	if clowder.LoadedConfig.TlsCAPath != nil {
		scheme += "s"
		if e.TlsPort != nil {
			port = *e.TlsPort
		}
	}
	return fmt.Sprintf("%s://%s:%d", scheme, e.Hostname, port)
}

func (e *PrivateEndpoint) buildURL() string {
	port := e.Port
	scheme := "http"
	if clowder.LoadedConfig.TlsCAPath != nil {
		scheme += "s"
		if e.TlsPort != nil {
			port = *e.TlsPort
		}
	}
	return fmt.Sprintf("%s://%s:%d", scheme, e.Hostname, port)
}

func initCloudwatchFromClowder() {
	cwCfg := clowder.LoadedConfig.Logging.Cloudwatch
	if cwCfg != nil {
		CoreCfg.CloudWatchAccessKeyID = cwCfg.AccessKeyId
		CoreCfg.CloudWatchSecretAccesskey = cwCfg.SecretAccessKey
		CoreCfg.CloudWatchRegion = cwCfg.Region
		CoreCfg.CloudWatchLogGroup = cwCfg.LogGroup
	}
}

func initPrometheusPushGatewayFromEnv() {
	CoreCfg.PrometheusPushGateway = Getenv("PROMETHEUS_PUSHGATEWAY", "pushgateway")
}

func initProfilerFromEnv() {
	CoreCfg.ProfilerEnabled = GetBoolEnvOrDefault("ENABLE_PROFILER", false)
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
	fmt.Printf("DB_ADMIN_USER=%s\n", CoreCfg.DBAdminUser)
	fmt.Printf("DB_ADMIN_PASSWD=%s\n", CoreCfg.DBAdminPassword)
	fmt.Printf("DB_HOST=%s\n", CoreCfg.DBHost)
	fmt.Printf("DB_NAME=%s\n", CoreCfg.DBName)
	fmt.Printf("DB_PORT=%d\n", CoreCfg.DBPort)
	fmt.Printf("DB_SSLMODE=%s\n", CoreCfg.DBSslMode)
	fmt.Printf("DB_SSLROOTCERT=%s\n", CoreCfg.DBSslRootCert)
}

func printAPIParams() {
	fmt.Printf("PUBLIC_PORT=%d\n", CoreCfg.PublicPort)
	fmt.Printf("PRIVATE_PORT=%d\n", CoreCfg.PrivatePort)
	fmt.Printf("METRICS_PORT=%d\n", CoreCfg.MetricsPort)
	fmt.Printf("METRICS_PATH=%s\n", CoreCfg.MetricsPath)
}

func printKafkaParams() {
	fmt.Printf("KAFKA_ADDRESS=%s\n", CoreCfg.KafkaAddress)
	if CoreCfg.KafkaSslEnabled {
		fmt.Println("ENABLE_KAFKA_SSL=true")
		fmt.Printf("KAFKA_SSL_CERT=%s\n", CoreCfg.KafkaSslCert)
		if CoreCfg.KafkaUsername != "" {
			fmt.Printf("KAFKA_USERNAME=%s\n", CoreCfg.KafkaUsername)
			fmt.Printf("KAFKA_PASSWORD=%s\n", CoreCfg.KafkaPassword)
		}
	}
	fmt.Printf("EVENTS_TOPIC=%s\n", CoreCfg.EventsTopic)
	fmt.Printf("EVAL_TOPIC=%s\n", CoreCfg.EvalTopic)
	fmt.Printf("PAYLOAD_TRACKER_TOPIC=%s\n", CoreCfg.PayloadTrackerTopic)
	fmt.Printf("REMEDIATIONS_UPDATE_TOPIC=%s\n", CoreCfg.RemediationUpdateTopic)
	fmt.Printf("NOTIFICATIONS_TOPIC=%s\n", CoreCfg.NotificationsTopic)
	fmt.Printf("TEMPLATE_TOPIC=%s\n", CoreCfg.TemplateTopic)
}

func printServicesParams() {
	fmt.Printf("VMAAS_ADDRESS=http://%s\n", CoreCfg.VmaasAddress)
	fmt.Printf("RBAC_ADDRESS=http://%s\n", CoreCfg.RbacAddress)
}

func printCloudwatchParams() {
	cwCfg := clowder.LoadedConfig.Logging.Cloudwatch
	if cwCfg == nil {
		fmt.Println("No Cloudwatch logging found")
		return
	}
	fmt.Printf("CW_AWS_ACCESS_KEY_ID=%s\n", CoreCfg.CloudWatchAccessKeyID)
	fmt.Printf("CW_AWS_SECRET_ACCESS_KEY=%s\n", CoreCfg.CloudWatchSecretAccesskey)
	fmt.Printf("CW_AWS_REGION=%s\n", CoreCfg.CloudWatchRegion)
	fmt.Printf("CW_AWS_LOG_GROUP=%s\n", CoreCfg.CloudWatchLogGroup)
}

func translateTopic(topic *string) {
	if v, ok := clowder.KafkaTopics[*topic]; ok {
		*topic = v.Name
	}
}
