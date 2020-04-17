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
	es "github.com/ONSdigital/dp-elasticsearch"
	"github.com/ONSdigital/dp-healthcheck/healthcheck"
	kafka "github.com/ONSdigital/dp-kafka"
	rchttp "github.com/ONSdigital/dp-rchttp"
	"github.com/ONSdigital/dp-search-builder/config"
	"github.com/ONSdigital/dp-search-builder/event"
	initialise "github.com/ONSdigital/dp-search-builder/initalise"
	"github.com/ONSdigital/go-ns/server"
	"github.com/ONSdigital/log.go/log"
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
	log.Namespace = "dp-search-builder"
	ctx := context.Background()

	if err := run(ctx); err != nil {
		log.Event(ctx, "application unexpectedly failed", log.ERROR, log.Error(err))
		os.Exit(1)
	}

	os.Exit(0)
}

func run(ctx context.Context) error {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	cfg, err := config.Get()
	if err != nil {
		log.Event(ctx, "failed to retrieve configuration", log.FATAL, log.Error(err))
		return err
	}

	log.Event(ctx, "config on startup", log.INFO, log.Data{"config": cfg})

	envMax, err := strconv.ParseInt(cfg.KafkaMaxBytes, 10, 32)
	if err != nil {
		log.Event(ctx, "encountered error parsing kafka max bytes", log.FATAL, log.Error(err))
		return err
	}

	// External services and their initialization state
	var serviceList initialise.ExternalServiceList

	syncConsumerGroup, err := serviceList.GetConsumer(ctx, cfg)
	if err != nil {
		log.Event(ctx, "could not initialise kafka consumer", log.FATAL, log.Error(err), log.Data{"group": cfg.ConsumerGroup, "topic": cfg.ConsumerTopic})
		return err
	}

	searchBuiltProducer, err := serviceList.GetProducer(ctx, cfg.Brokers, cfg.ProducerTopic, initialise.SearchBuilt, int(envMax))
	if err != nil {
		log.Event(ctx, "could not initialise kafka producer", log.FATAL, log.Error(err), log.Data{"topic": cfg.ProducerTopic})
		return err
	}

	searchBuilderErrProducer, err := serviceList.GetProducer(ctx, cfg.Brokers, cfg.EventReporterTopic, initialise.SearchBuilderErr, int(envMax))
	if err != nil {
		log.Event(ctx, "could not initialise kafka producer", log.FATAL, log.Error(err), log.Data{"topic": cfg.EventReporterTopic})
		return err
	}

	// Get Error reporter
	errorReporter, err := serviceList.GetImportErrorReporter(searchBuilderErrProducer, log.Namespace)
	if err != nil {
		log.Event(ctx, "error while attempting to create error reporter client", log.FATAL, log.Error(err))
		return err
	}

	// Get HealthCheck
	hc, err := serviceList.GetHealthCheck(cfg, BuildTime, GitCommit, Version)
	if err != nil {
		log.Event(ctx, "could not instantiate healthcheck", log.FATAL, log.Error(err))
		return err
	}

	hierarchyClient := hierarchy.New(cfg.HierarchyAPIURL)
	elasticSearchClient := es.NewClient(cfg.ElasticSearchAPIURL, cfg.SignElasticsearchRequests, cfg.MaxRetries)

	// Add a list of checkers to HealthCheck
	if err := registerCheckers(ctx, &hc, syncConsumerGroup, searchBuiltProducer, searchBuilderErrProducer, elasticSearchClient, *hierarchyClient); err != nil {
		return err
	}

	router := mux.NewRouter()
	router.Path("/health").HandlerFunc(hc.Handler)
	httpServer := server.New(cfg.BindAddr, router)

	// Disable auto handling of os signals by the HTTP server. This is handled
	// in the service so we can gracefully shutdown resources other than just
	// the HTTP server.
	httpServer.HandleOSSignals = false

	// a channel to signal a server error
	errorChannel := make(chan error)

	go func() {
		log.Event(ctx, "starting http server", log.INFO, log.Data{"bind_addr": cfg.BindAddr})
		if err := httpServer.ListenAndServe(); err != nil {
			errorChannel <- err
		}
	}()

	hc.Start(ctx)

	clienter := rchttp.NewClient()

	log.Event(ctx, "application started", log.INFO, log.Data{"search_builder_url": cfg.SearchBuilderURL})

	consumer := event.NewConsumer(clienter, cfg.HierarchyAPIURL, cfg.ElasticSearchAPIURL, cfg.SignElasticsearchRequests, searchBuiltProducer, errorReporter)

	// Start listening for event messages
	consumer.Consume(syncConsumerGroup)

	syncConsumerGroup.Channels().LogErrors(ctx, "error received from kafka consumer, topic: "+cfg.ConsumerTopic)
	searchBuiltProducer.Channels().LogErrors(ctx, "error received from kafka producer, topic: "+cfg.ProducerTopic)
	searchBuilderErrProducer.Channels().LogErrors(ctx, "error received from kafka producer, topic: "+cfg.EventReporterTopic)

	// block until a fatal error, signal or eventLoopDone - then proceed to shutdown
	select {
	case signal := <-signals:
		log.Event(ctx, "quitting after os signal received", log.INFO, log.Data{"signal": signal})
	case <-errorChannel:
		log.Event(ctx, "server error", log.ERROR, log.Error(errors.New("server error forcing shutdown")))
	}

	log.Event(ctx, fmt.Sprintf("shutdown with timeout: %s", cfg.GracefulShutdownTimeout), log.INFO)
	shutdownContext, shutdownCancel := context.WithTimeout(context.Background(), cfg.GracefulShutdownTimeout)

	// track shutdown gracefully closes app
	var hasShutdownError bool

	go func() {
		defer shutdownCancel()

		if serviceList.HealthCheck {
			log.Event(shutdownContext, "closing health check", log.INFO)
			hc.Stop()
			log.Event(shutdownContext, "closed health check", log.INFO)
		}

		log.Event(shutdownContext, "closing http server", log.INFO)
		err = httpServer.Shutdown(shutdownContext)
		hasShutdownError = handleShutdownError(shutdownContext, "http server", err, hasShutdownError, nil)

		// If kafka consumer exists, stop listening to it. (Will close later)
		if serviceList.Consumer {
			log.Event(shutdownContext, "closing kafka consumer listener", log.INFO, log.Data{"topic": cfg.ConsumerTopic})
			err = syncConsumerGroup.StopListeningToConsumer(shutdownContext)
			hasShutdownError = handleShutdownError(shutdownContext, "kafka consumer listener", err, hasShutdownError, log.Data{"topic": cfg.ConsumerTopic})
		}

		// If search built kafka producer exists, close it
		if serviceList.SearchBuiltProducer {
			log.Event(shutdownContext, "closing search built kafka producer", log.INFO, log.Data{"topic": cfg.ProducerTopic})
			err = searchBuiltProducer.Close(shutdownContext)
			hasShutdownError = handleShutdownError(shutdownContext, "search built kafka producer", err, hasShutdownError, log.Data{"topic": cfg.ProducerTopic})
		}

		// If search builder error kafka producer exists, close it
		if serviceList.SearchBuilderErrProducer {
			log.Event(shutdownContext, "closing search builder error kafka producer", log.INFO, log.Data{"topic": cfg.EventReporterTopic})
			err = searchBuilderErrProducer.Close(shutdownContext)
			hasShutdownError = handleShutdownError(shutdownContext, "search builder error kafka producer", err, hasShutdownError, log.Data{"topic": cfg.EventReporterTopic})
		}

		// Close consumer loop
		log.Event(shutdownContext, "closing search builder consumer loop", log.INFO)
		err = consumer.Close(shutdownContext)
		hasShutdownError = handleShutdownError(shutdownContext, "search builder consumer loop", err, hasShutdownError, nil)

		// If kafka consumer exists, close it
		if serviceList.Consumer {
			log.Event(shutdownContext, "closing kafka consumer", log.INFO, log.Data{"topic": cfg.ConsumerTopic})
			err = syncConsumerGroup.Close(shutdownContext)
			hasShutdownError = handleShutdownError(shutdownContext, "kafka consumer", err, hasShutdownError, log.Data{"topic": cfg.ConsumerTopic})
		}
	}()

	// wait for shutdown success (via cancel) or failure (timeout)
	<-shutdownContext.Done()

	if shutdownContext.Err() == context.DeadlineExceeded {
		log.Event(shutdownContext, "graceful shutdown exceeded context deadline", log.ERROR, log.Error(shutdownContext.Err()))
		return shutdownContext.Err()
	}

	if hasShutdownError {
		err = errors.New("failed to shutdown gracefully")
		log.Event(shutdownContext, "failed to shutdown gracefully ", log.ERROR, log.Error(err))
		return err
	}

	log.Event(shutdownContext, "graceful shutdown was successful", log.INFO)

	return nil
}

// registerCheckers adds the checkers for the provided clients to the healthcheck object
func registerCheckers(ctx context.Context, hc *healthcheck.HealthCheck,
	kafkaConsumer *kafka.ConsumerGroup,
	searchBuiltProducer *kafka.Producer,
	searchBuilderErrProducer *kafka.Producer,
	elasticsearchClient *es.Client,
	hierarchyClient hierarchy.Client) (err error) {

	hasErrors := false

	if err = hc.AddCheck("Kafka Consumer", kafkaConsumer.Checker); err != nil {
		hasErrors = true
		log.Event(ctx, "error adding check for kafka consumer", log.ERROR, log.Error(err))
	}

	if err = hc.AddCheck("Kafka Search Built Producer", searchBuiltProducer.Checker); err != nil {
		hasErrors = true
		log.Event(ctx, "error adding check for kafka search built producer", log.ERROR, log.Error(err))
	}

	if err = hc.AddCheck("Kafka Error Producer", searchBuilderErrProducer.Checker); err != nil {
		hasErrors = true
		log.Event(ctx, "error adding check for kafka error producer", log.ERROR, log.Error(err))
	}

	if err = hc.AddCheck("Elasticsearch", elasticsearchClient.Checker); err != nil {
		hasErrors = true
		log.Event(ctx, "error adding check for elasticsearch client", log.ERROR, log.Error(err))
	}

	if err = hc.AddCheck("Hierarchy API", hierarchyClient.Checker); err != nil {
		hasErrors = true
		log.Event(ctx, "error adding check for hierarchy client", log.ERROR, log.Error(err))
	}

	if hasErrors {
		return errors.New("Error(s) registering checkers for healthcheck")
	}

	return nil
}

func handleShutdownError(ctx context.Context, service string, err error, hasError bool, logData log.Data) bool {

	if err != nil {
		log.Event(ctx, "unsuccessfully closed "+service, log.ERROR, log.Error(err), logData)
		return true
	}

	log.Event(ctx, "closed "+service, log.INFO, logData)

	return hasError
}
