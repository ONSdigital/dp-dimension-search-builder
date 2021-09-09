package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/ONSdigital/dp-api-clients-go/hierarchy"
	"github.com/ONSdigital/dp-dimension-search-builder/config"
	"github.com/ONSdigital/dp-dimension-search-builder/event"
	initialise "github.com/ONSdigital/dp-dimension-search-builder/initalise"
	"github.com/ONSdigital/dp-elasticsearch/v2/elasticsearch"
	"github.com/ONSdigital/dp-healthcheck/healthcheck"
	kafka "github.com/ONSdigital/dp-kafka/v2"
	"github.com/ONSdigital/dp-net/http"
	"github.com/ONSdigital/log.go/v2/log"
	"github.com/gorilla/mux"
)

var (
	// BuildTime represents the time in which the service was built
	BuildTime string
	// GitCommit represents the commit (SHA-1) hash of the service that is running
	GitCommit string
	// Version represents the version of the service that is running
	Version string
)

func main() {
	log.Namespace = "dp-dimension-search-builder"
	ctx := context.Background()

	if err := run(ctx); err != nil {
		log.Error(ctx, "application unexpectedly failed", err)
		os.Exit(1)
	}

	os.Exit(0)
}

func run(ctx context.Context) error {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	cfg, err := config.Get()
	if err != nil {
		log.Fatal(ctx, "failed to retrieve configuration", err)
		return err
	}

	log.Info(ctx, "config on startup", log.Data{"config": cfg})

	envMax, err := strconv.ParseInt(cfg.KafkaConfig.MaxBytes, 10, 32)
	if err != nil {
		log.Fatal(ctx, "encountered error parsing kafka max bytes", err)
		return err
	}

	// External services and their initialization state
	var serviceList initialise.ExternalServiceList

	syncConsumerGroup, err := serviceList.GetConsumer(ctx, cfg.KafkaConfig)
	if err != nil {
		log.Fatal(ctx, "could not initialise kafka consumer", err, log.Data{"group": cfg.KafkaConfig.ConsumerGroup, "topic": cfg.KafkaConfig.ConsumerTopic})
		return err
	}

	searchBuiltProducer, err := serviceList.GetProducer(ctx, cfg.KafkaConfig, cfg.KafkaConfig.ProducerTopic, initialise.SearchBuilt, int(envMax))
	if err != nil {
		log.Fatal(ctx, "could not initialise kafka producer", err, log.Data{"topic": cfg.KafkaConfig.ProducerTopic})
		return err
	}

	searchBuilderErrProducer, err := serviceList.GetProducer(ctx, cfg.KafkaConfig, cfg.KafkaConfig.EventReporterTopic, initialise.SearchBuilderErr, int(envMax))
	if err != nil {
		log.Fatal(ctx, "could not initialise kafka producer", err, log.Data{"topic": cfg.KafkaConfig.EventReporterTopic})
		return err
	}

	// Get Error reporter
	errorReporter, err := serviceList.GetImportErrorReporter(searchBuilderErrProducer, log.Namespace)
	if err != nil {
		log.Fatal(ctx, "error while attempting to create error reporter client", err)
		return err
	}

	// Get HealthCheck
	hc, err := serviceList.GetHealthCheck(cfg, BuildTime, GitCommit, Version)
	if err != nil {
		log.Fatal(ctx, "could not instantiate healthcheck", err)
		return err
	}

	hierarchyClient := hierarchy.New(cfg.HierarchyAPIURL)
	elasticSearchClient := elasticsearch.NewClient(cfg.ElasticSearchAPIURL, cfg.SignElasticsearchRequests, cfg.MaxRetries)

	// Add a list of checkers to HealthCheck
	if err := registerCheckers(ctx, &hc, syncConsumerGroup, searchBuiltProducer, searchBuilderErrProducer, elasticSearchClient, *hierarchyClient); err != nil {
		return err
	}

	router := mux.NewRouter()
	router.Path("/health").HandlerFunc(hc.Handler)
	httpServer := http.NewServer(cfg.BindAddr, router)

	// Disable auto handling of os signals by the HTTP server. This is handled
	// in the service so we can gracefully shutdown resources other than just
	// the HTTP server.
	httpServer.HandleOSSignals = false

	// a channel to signal a server error
	errorChannel := make(chan error)

	go func() {
		log.Info(ctx, "starting http server", log.Data{"bind_addr": cfg.BindAddr})
		if err := httpServer.ListenAndServe(); err != nil {
			errorChannel <- err
		}
	}()

	hc.Start(ctx)

	clienter := http.NewClient()

	log.Info(ctx, "application started", log.Data{"search_builder_url": cfg.SearchBuilderURL})

	consumer := event.NewConsumer(clienter, cfg.HierarchyAPIURL, elasticSearchClient, searchBuiltProducer, errorReporter)

	// Start listening for event messages
	consumer.Consume(syncConsumerGroup)

	syncConsumerGroup.Channels().LogErrors(ctx, "error received from kafka consumer, topic: "+cfg.KafkaConfig.ConsumerTopic)
	searchBuiltProducer.Channels().LogErrors(ctx, "error received from kafka producer, topic: "+cfg.KafkaConfig.ProducerTopic)
	searchBuilderErrProducer.Channels().LogErrors(ctx, "error received from kafka producer, topic: "+cfg.KafkaConfig.EventReporterTopic)

	// block until a fatal error, signal or eventLoopDone - then proceed to shutdown
	select {
	case signal := <-signals:
		log.Info(ctx, "quitting after os signal received", log.Data{"signal": signal})
	case <-errorChannel:
		log.Error(ctx, "server error", errors.New("server error forcing shutdown"))
	}

	log.Info(ctx, fmt.Sprintf("shutdown with timeout: %s", cfg.GracefulShutdownTimeout))
	shutdownContext, shutdownCancel := context.WithTimeout(context.Background(), cfg.GracefulShutdownTimeout)

	// track shutdown gracefully closes app
	var hasShutdownError bool

	go func() {
		defer shutdownCancel()

		if serviceList.HealthCheck {
			log.Info(shutdownContext, "closing health check")
			hc.Stop()
			log.Info(shutdownContext, "closed health check")
		}

		log.Info(shutdownContext, "closing http server")
		err = httpServer.Shutdown(shutdownContext)
		hasShutdownError = handleShutdownError(shutdownContext, "http server", err, hasShutdownError, nil)

		// If kafka consumer exists, stop listening to it. (Will close later)
		if serviceList.Consumer {
			log.Info(shutdownContext, "closing kafka consumer listener", log.Data{"topic": cfg.KafkaConfig.ConsumerTopic})
			err = syncConsumerGroup.StopListeningToConsumer(shutdownContext)
			hasShutdownError = handleShutdownError(shutdownContext, "kafka consumer listener", err, hasShutdownError, log.Data{"topic": cfg.KafkaConfig.ConsumerTopic})
		}

		// If search built kafka producer exists, close it
		if serviceList.SearchBuiltProducer {
			log.Info(shutdownContext, "closing search built kafka producer", log.Data{"topic": cfg.KafkaConfig.ProducerTopic})
			err = searchBuiltProducer.Close(shutdownContext)
			hasShutdownError = handleShutdownError(shutdownContext, "search built kafka producer", err, hasShutdownError, log.Data{"topic": cfg.KafkaConfig.ProducerTopic})
		}

		// If dimension search builder error kafka producer exists, close it
		if serviceList.SearchBuilderErrProducer {
			log.Info(shutdownContext, "closing dimension search builder error kafka producer", log.Data{"topic": cfg.KafkaConfig.EventReporterTopic})
			err = searchBuilderErrProducer.Close(shutdownContext)
			hasShutdownError = handleShutdownError(shutdownContext, "dimension search builder error kafka producer", err, hasShutdownError, log.Data{"topic": cfg.KafkaConfig.EventReporterTopic})
		}

		// Close consumer loop
		log.Info(shutdownContext, "closing dimension search builder consumer loop")
		err = consumer.Close(shutdownContext)
		hasShutdownError = handleShutdownError(shutdownContext, "dimension search builder consumer loop", err, hasShutdownError, nil)

		// If kafka consumer exists, close it
		if serviceList.Consumer {
			log.Info(shutdownContext, "closing kafka consumer", log.Data{"topic": cfg.KafkaConfig.ConsumerTopic})
			err = syncConsumerGroup.Close(shutdownContext)
			hasShutdownError = handleShutdownError(shutdownContext, "kafka consumer", err, hasShutdownError, log.Data{"topic": cfg.KafkaConfig.ConsumerTopic})
		}
	}()

	// wait for shutdown success (via cancel) or failure (timeout)
	<-shutdownContext.Done()

	if shutdownContext.Err() == context.DeadlineExceeded {
		log.Error(shutdownContext, "graceful shutdown exceeded context deadline", shutdownContext.Err())
		return shutdownContext.Err()
	}

	if hasShutdownError {
		err = errors.New("failed to shutdown gracefully")
		log.Error(shutdownContext, "failed to shutdown gracefully ", err)
		return err
	}

	log.Info(shutdownContext, "graceful shutdown was successful")

	return nil
}

// registerCheckers adds the checkers for the provided clients to the healthcheck object
func registerCheckers(ctx context.Context, hc *healthcheck.HealthCheck,
	kafkaConsumer *kafka.ConsumerGroup,
	searchBuiltProducer *kafka.Producer,
	searchBuilderErrProducer *kafka.Producer,
	elasticsearchClient *elasticsearch.Client,
	hierarchyClient hierarchy.Client) (err error) {

	hasErrors := false

	if err = hc.AddCheck("Kafka Consumer", kafkaConsumer.Checker); err != nil {
		hasErrors = true
		log.Error(ctx, "error adding check for kafka consumer", err)
	}

	if err = hc.AddCheck("Kafka Search Built Producer", searchBuiltProducer.Checker); err != nil {
		hasErrors = true
		log.Error(ctx, "error adding check for kafka search built producer", err)
	}

	if err = hc.AddCheck("Kafka Error Producer", searchBuilderErrProducer.Checker); err != nil {
		hasErrors = true
		log.Error(ctx, "error adding check for kafka error producer", err)
	}

	if err = hc.AddCheck("Elasticsearch", elasticsearchClient.Checker); err != nil {
		hasErrors = true
		log.Error(ctx, "error adding check for elasticsearch client", err)
	}

	if err = hc.AddCheck("Hierarchy API", hierarchyClient.Checker); err != nil {
		hasErrors = true
		log.Error(ctx, "error adding check for hierarchy client", err)
	}

	if hasErrors {
		return errors.New("Error(s) registering checkers for healthcheck")
	}

	return nil
}

func handleShutdownError(ctx context.Context, service string, err error, hasError bool, logData log.Data) bool {

	if err != nil {
		log.Error(ctx, "unsuccessfully closed "+service, err, logData)
		return true
	}

	log.Info(ctx, "closed "+service, logData)

	return hasError
}
