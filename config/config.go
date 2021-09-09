package config

import (
	"time"

	"github.com/kelseyhightower/envconfig"
)

// KafkaTLSProtocolFlag informs service to use TLS protocol for kafka
const KafkaTLSProtocolFlag = "TLS"

// Config is the filing resource handler config
type Config struct {
	AwsRegion                  string        `envconfig:"AWS_REGION"`
	AwsService                 string        `envconfig:"AWS_SERVICE"`
	BindAddr                   string        `envconfig:"BIND_ADDR"`
	ElasticSearchAPIURL        string        `envconfig:"ELASTIC_SEARCH_URL"`
	GracefulShutdownTimeout    time.Duration `envconfig:"GRACEFUL_SHUTDOWN_TIMEOUT"`
	HealthCheckInterval        time.Duration `envconfig:"HEALTHCHECK_INTERVAL"`
	HealthCheckCriticalTimeout time.Duration `envconfig:"HEALTHCHECK_CRITICAL_TIMEOUT"`
	HierarchyAPIURL            string        `envconfig:"HIERARCHY_API_URL"`
	KafkaConfig                KafkaConfig
	MaxRetries                 int    `envconfig:"REQUEST_MAX_RETRIES"`
	SearchBuilderURL           string `envconfig:"SEARCH_BUILDER_URL"`
	SignElasticsearchRequests  bool   `envconfig:"SIGN_ELASTICSEARCH_REQUESTS"`
}

// KafkaConfig contains the config required to connect to Kafka
type KafkaConfig struct {
	BindAddr           []string `envconfig:"KAFKA_ADDR"                 json:"-"`
	Version            string   `envconfig:"KAFKA_VERSION"`
	MaxBytes           string   `envconfig:"KAFKA_MAX_BYTES"`
	SecProtocol        string   `envconfig:"KAFKA_SEC_PROTO"`
	SecCACerts         string   `envconfig:"KAFKA_SEC_CA_CERTS"`
	SecClientCert      string   `envconfig:"KAFKA_SEC_CLIENT_CERT"`
	SecClientKey       string   `envconfig:"KAFKA_SEC_CLIENT_KEY"       json:"-"`
	SecSkipVerify      bool     `envconfig:"KAFKA_SEC_SKIP_VERIFY"`
	OffsetOldest       bool     `envconfig:"KAFKA_OFFSET_OLDEST"`
	ConsumerGroup      string   `envconfig:"CONSUMER_GROUP"`
	ConsumerTopic      string   `envconfig:"HIERARCHY_BUILT_TOPIC"`
	EventReporterTopic string   `envconfig:"EVENT_REPORTER_TOPIC"`
	ProducerTopic      string   `envconfig:"PRODUCER_TOPIC"`
}

var cfg *Config

func getDefaultConfig() *Config {
	return &Config{
		AwsRegion:                  "eu-west-1",
		AwsService:                 "es",
		BindAddr:                   ":22900",
		ElasticSearchAPIURL:        "http://localhost:10200",
		GracefulShutdownTimeout:    5 * time.Second,
		HealthCheckInterval:        30 * time.Second,
		HealthCheckCriticalTimeout: 90 * time.Second,
		HierarchyAPIURL:            "http://localhost:22600",
		KafkaConfig: KafkaConfig{
			BindAddr:           []string{"localhost:9092"},
			MaxBytes:           "2000000",
			Version:            "1.0.2",
			SecProtocol:        "",
			SecCACerts:         "",
			SecClientCert:      "",
			SecClientKey:       "",
			SecSkipVerify:      false,
			OffsetOldest:       true,
			ConsumerGroup:      "dp-dimension-search-builder",
			ConsumerTopic:      "hierarchy-built",
			EventReporterTopic: "report-events",
			ProducerTopic:      "dimension-search-built",
		},
		MaxRetries:                3,
		SearchBuilderURL:          "http://localhost:22900",
		SignElasticsearchRequests: false,
	}
}

// Get configures the application and returns the configuration
func Get() (*Config, error) {
	if cfg != nil {
		return cfg, nil
	}

	cfg = getDefaultConfig()

	if err := envconfig.Process("", cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}
