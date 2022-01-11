package initialise

import (
	"context"
	"fmt"

	"github.com/ONSdigital/dp-dimension-search-builder/config"
	"github.com/ONSdigital/dp-healthcheck/healthcheck"
	kafka "github.com/ONSdigital/dp-kafka/v2"
	"github.com/ONSdigital/dp-reporter-client/reporter"
)

// ExternalServiceList represents a list of services
type ExternalServiceList struct {
	Consumer                 bool
	SearchBuiltProducer      bool
	SearchBuilderErrProducer bool
	ElasticSearch            bool
	ErrorReporter            bool
	HealthCheck              bool
}

// KafkaProducerName : Type for kafka producer name used by iota constants
type KafkaProducerName int

// Possible names of Kafa Producers
const (
	SearchBuilt = iota
	SearchBuilderErr
)

var kafkaProducerNames = []string{"SearchBuilt", "SearchBuilderErr"}

var bufferSize = 1

// Values of the kafka producers names
func (k KafkaProducerName) String() string {
	return kafkaProducerNames[k]
}

// GetConsumer returns a kafka consumer, which might not be initialised yet.
func (e *ExternalServiceList) GetConsumer(ctx context.Context, kafkaConfig config.KafkaConfig) (kafkaConsumer *kafka.ConsumerGroup, err error) {

	kafkaOffset := kafka.OffsetNewest

	if kafkaConfig.OffsetOldest {
		kafkaOffset = kafka.OffsetOldest
	}

	cgConfig := &kafka.ConsumerGroupConfig{
		Offset:       &kafkaOffset,
		KafkaVersion: &kafkaConfig.Version,
	}
	if kafkaConfig.SecProtocol == config.KafkaTLSProtocolFlag {
		cgConfig.SecurityConfig = kafka.GetSecurityConfig(
			kafkaConfig.SecCACerts,
			kafkaConfig.SecClientCert,
			kafkaConfig.SecClientKey,
			kafkaConfig.SecSkipVerify,
		)
	}

	cgChannels := kafka.CreateConsumerGroupChannels(bufferSize)
	kafkaConsumer, err = kafka.NewConsumerGroup(
		ctx,
		kafkaConfig.BindAddr,
		kafkaConfig.ConsumerTopic,
		kafkaConfig.ConsumerGroup,
		cgChannels,
		cgConfig,
	)
	if err != nil {
		return
	}

	e.Consumer = true
	return
}

// GetProducer returns a kafka producer, which might not be initialised yet.
func (e *ExternalServiceList) GetProducer(ctx context.Context, kafkaConfig config.KafkaConfig, topic string, name KafkaProducerName, envMax int) (kafkaProducer *kafka.Producer, err error) {
	pChannels := kafka.CreateProducerChannels()
	pConfig := &kafka.ProducerConfig{
		KafkaVersion:    &kafkaConfig.Version,
		MaxMessageBytes: &envMax,
	}
	if kafkaConfig.SecProtocol == config.KafkaTLSProtocolFlag {
		pConfig.SecurityConfig = kafka.GetSecurityConfig(
			kafkaConfig.SecCACerts,
			kafkaConfig.SecClientCert,
			kafkaConfig.SecClientKey,
			kafkaConfig.SecSkipVerify,
		)
	}

	kafkaProducer, err = kafka.NewProducer(ctx, kafkaConfig.BindAddr, topic, pChannels, pConfig)
	if err != nil {
		return
	}

	switch {
	case name == SearchBuilt:
		e.SearchBuiltProducer = true
	case name == SearchBuilderErr:
		e.SearchBuilderErrProducer = true
	default:
		err = fmt.Errorf("kafka producer name not recognised: '%s'. Valid names: %v", name.String(), kafkaProducerNames)
	}

	return
}

// GetImportErrorReporter returns an ErrorImportReporter to send error reports to the import-reporter (only if ObservationsImportedErrProducer is available)
func (e *ExternalServiceList) GetImportErrorReporter(searchBuilderErrProducer reporter.KafkaProducer, serviceName string) (errorReporter reporter.ImportErrorReporter, err error) {
	if !e.SearchBuilderErrProducer {
		return reporter.ImportErrorReporter{},
			fmt.Errorf("cannot create ImportErrorReporter because kafka producer '%s' is not available", kafkaProducerNames[SearchBuilderErr])
	}

	errorReporter, err = reporter.NewImportErrorReporter(searchBuilderErrProducer, serviceName)
	if err != nil {
		return
	}

	e.ErrorReporter = true
	return
}

// GetHealthCheck creates a healthcheck with versionInfo
func (e *ExternalServiceList) GetHealthCheck(cfg *config.Config, buildTime, gitCommit, version string) (healthcheck.HealthCheck, error) {

	// Create healthcheck object with versionInfo
	versionInfo, err := healthcheck.NewVersionInfo(buildTime, gitCommit, version)
	if err != nil {
		return healthcheck.HealthCheck{}, err
	}
	hc := healthcheck.New(versionInfo, cfg.HealthCheckCriticalTimeout, cfg.HealthCheckInterval)

	e.HealthCheck = true

	return hc, nil
}
