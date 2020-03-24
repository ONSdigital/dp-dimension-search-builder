package config

import (
	"time"

	"github.com/kelseyhightower/envconfig"
)

// Config is the filing resource handler config
type Config struct {
	BindAddr                  string        `envconfig:"BIND_ADDR"`
	Brokers                   []string      `envconfig:"KAFKA_ADDR"`
	ConsumerGroup             string        `envconfig:"CONSUMER_GROUP"`
	ConsumerTopic             string        `envconfig:"HIERARCHY_BUILT_TOPIC"`
	ElasticSearchAPIURL       string        `envconfig:"ELASTIC_SEARCH_URL"`
	EventReporterTopic        string        `envconfig:"EVENT_REPORTER_TOPIC"`
	GracefulShutdownTimeout   time.Duration `envconfig:"GRACEFUL_SHUTDOWN_TIMEOUT"`
	HealthcheckInterval       time.Duration `envconfig:"HEALTHCHECK_INTERVAL"`
	HealthcheckTimeout        time.Duration `envconfig:"HEALTHCHECK_TIMEOUT"`
	HierarchyAPIURL           string        `envconfig:"HIERARCHY_API_URL"`
	KafkaMaxBytes             string        `envconfig:"KAFKA_MAX_BYTES"`
	MaxRetries                int           `envconfig:"REQUEST_MAX_RETRIES"`
	ProducerTopic             string        `envconfig:"PRODUCER_TOPIC"`
	SearchBuilderURL          string        `envconfig:"SEARCH_BUILDER_URL"`
	SignElasticsearchRequests bool          `envconfig:"SIGN_ELASTICSEARCH_REQUESTS"`
}

var cfg *Config

// Get configures the application and returns the configuration
func Get() (*Config, error) {
	if cfg != nil {
		return cfg, nil
	}

	cfg = &Config{
		BindAddr:                  ":22900",
		Brokers:                   []string{"localhost:9092"},
		ConsumerGroup:             "dp-search-builder",
		ConsumerTopic:             "hierarchy-built",
		ElasticSearchAPIURL:       "http://localhost:10200",
		EventReporterTopic:        "report-events",
		GracefulShutdownTimeout:   5 * time.Second,
		HealthcheckInterval:       time.Minute,
		HealthcheckTimeout:        2 * time.Second,
		HierarchyAPIURL:           "http://localhost:22600",
		KafkaMaxBytes:             "2000000",
		MaxRetries:                3,
		ProducerTopic:             "search-built",
		SearchBuilderURL:          "http://localhost:22900",
		SignElasticsearchRequests: false,
	}

	return cfg, envconfig.Process("", cfg)
}
