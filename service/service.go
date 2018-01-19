package service

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/net/context"

	"github.com/ONSdigital/dp-reporter-client/reporter"
	hierarchyHealthCheck "github.com/ONSdigital/go-ns/clients/hierarchy"
	"github.com/ONSdigital/go-ns/elasticsearch"
	"github.com/ONSdigital/go-ns/healthcheck"
	"github.com/ONSdigital/go-ns/kafka"
	"github.com/ONSdigital/go-ns/log"
	"github.com/ONSdigital/go-ns/rchttp"
)

// Service represents the necessary config for dp-dimension-extractor
type Service struct {
	EnvMax                   int64
	BindAddr                 string
	Consumer                 *kafka.ConsumerGroup
	ElasticSearchURL         string
	ErrorReporter            reporter.ImportErrorReporter
	HealthcheckInterval      time.Duration
	HealthcheckTimeout       time.Duration
	HierarchyAPIURL          string
	HTTPClient               *rchttp.Client
	MaxRetries               int
	SearchBuiltProducer      kafka.Producer
	SearchBuilderErrProducer kafka.Producer
	SearchBuilderURL         string
	Shutdown                 time.Duration
}

// Start handles consumption of events
func (svc *Service) Start() {
	log.Info("application started", log.Data{"search_builder_url": svc.SearchBuilderURL})

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	eventLoopDone := make(chan bool)
	errorChannel := make(chan error, 1)

	svc.HTTPClient = rchttp.DefaultClient

	eventLoopContext, eventLoopCancel := context.WithCancel(context.Background())

	healthChecker := healthcheck.NewServer(
		svc.BindAddr,
		svc.HealthcheckInterval,
		errorChannel,
		hierarchyHealthCheck.New(svc.HierarchyAPIURL),
		elasticsearch.NewHealthCheckClient(svc.ElasticSearchURL),
	)

	// eventLoop
	go func() {
		defer close(eventLoopDone)
		for {
			select {
			case <-eventLoopContext.Done():
				log.Trace("Event loop context done", log.Data{"eventLoopContextErr": eventLoopContext.Err()})
				return
			case message := <-svc.Consumer.Incoming():

				instanceID, dimension, err := svc.handleMessage(eventLoopContext, message)
				logData := log.Data{"func": "service.Start.eventLoop", "instance_id": instanceID, "dimension": dimension, "kafka_offset": message.Offset()}
				if err != nil {
					log.ErrorC("event failed to process", err, logData)

					if len(instanceID) == 0 {
						log.ErrorC("instance_id is empty errorReporter.Notify will not be called", err, logData)
					} else {
						message := fmt.Sprintf("event failed to process, dimension is [%s]", dimension)
						err = svc.ErrorReporter.Notify(instanceID, message, err)
						if err != nil {
							log.ErrorC("ErrorProducer.Notify returned an error", err, logData)
						}
					}

				} else {
					log.Debug("event successfully processed", logData)
				}

				svc.Consumer.CommitAndRelease(message)
				log.Debug("message committed", logData)
			}
		}
	}()

	// block until a fatal error, signal or eventLoopDone - then proceed to shutdown
	select {
	case <-eventLoopDone:
		log.Debug("after eventLoopDone was closed", nil)
	case signal := <-signals:
		log.Debug("quitting after os signal received", log.Data{"signal": signal})
	case consumerError := <-svc.Consumer.Errors():
		log.Error(errors.New("aborting consumer"), log.Data{"message_received": consumerError})
	case searchBuiltProducerError := <-svc.SearchBuiltProducer.Errors():
		log.Error(errors.New("aborting producer"), log.Data{"message_received": searchBuiltProducerError})
	case searchBuilderProducerError := <-svc.SearchBuilderErrProducer.Errors():
		log.Error(errors.New("aborting producer"), log.Data{"message_received": searchBuilderProducerError})
	case <-errorChannel:
		log.Error(errors.New("server error forcing shutdown"), nil)
	}

	// give the app `Timeout` seconds to close gracefully before killing it.
	ctx, cancel := context.WithTimeout(context.Background(), svc.Shutdown)

	go func() {
		log.Debug("Stopping kafka consumer listener", nil)
		svc.Consumer.StopListeningToConsumer(ctx)
		log.Debug("Stopped kafka consumer listener", nil)
		eventLoopCancel()
		<-eventLoopDone

		log.Debug("Closing kafka producers", nil)
		svc.SearchBuiltProducer.Close(ctx)
		svc.SearchBuilderErrProducer.Close(ctx)
		log.Debug("Closed kafka producers", nil)
		log.Debug("Closing kafka consumer", nil)
		svc.Consumer.Close(ctx)
		log.Debug("Closed kafka consumer", nil)

		if err := healthChecker.Close(ctx); err != nil {
			log.ErrorC("Failed to gracefully close healthchecker", err, nil)
		}

		log.Info("Done shutdown - cancelling timeout context", nil)

		cancel() // stop timer
	}()

	// wait for timeout or success (via cancel)
	<-ctx.Done()
	if ctx.Err() == context.DeadlineExceeded {
		log.Error(ctx.Err(), nil)
	} else {
		log.Info("Done shutdown gracefully", log.Data{"context": ctx.Err()})
	}
	os.Exit(1)
}
