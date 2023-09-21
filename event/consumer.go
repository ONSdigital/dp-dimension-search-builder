package event

import (
	"context"
	"errors"
	"fmt"

	"github.com/ONSdigital/dp-elasticsearch/v2/elasticsearch"
	kafka "github.com/ONSdigital/dp-kafka/v2"
	"github.com/ONSdigital/dp-net/v2/http"
	"github.com/ONSdigital/dp-reporter-client/reporter"
	"github.com/ONSdigital/log.go/v2/log"
)

// Consumer consumes event messages.
type Consumer struct {
	Service Service
	closing chan eventClose
	closed  chan bool
}

// Service contains service configuration for consumer
type Service struct {
	ErrorReporter       reporter.ImportErrorReporter
	HierarchyAPIURL     string
	HTTPClienter        http.Clienter
	SearchBuiltProducer *kafka.Producer
	ElasticSearchClient *elasticsearch.Client
}

type eventClose struct {
	ctx context.Context
}

// NewConsumer returns a new consumer instance.
func NewConsumer(clienter http.Clienter, hierarchyAPIURL string, elasticSearchClient *elasticsearch.Client,
	searchBuiltProducer *kafka.Producer, errorReporter reporter.ImportErrorReporter) *Consumer {

	service := Service{
		ErrorReporter:       errorReporter,
		HierarchyAPIURL:     hierarchyAPIURL,
		HTTPClienter:        clienter,
		SearchBuiltProducer: searchBuiltProducer,
		ElasticSearchClient: elasticSearchClient,
	}

	consumer := &Consumer{
		Service: service,
		closing: make(chan eventClose),
		closed:  make(chan bool),
	}

	return consumer
}

// Consume handles consumption of events
func (consumer *Consumer) Consume(ctx context.Context, messageConsumer *kafka.ConsumerGroup) {

	// eventLoop
	go func() {
		defer close(consumer.closed)

		for {
			select {
			case msg := <-messageConsumer.Channels().Upstream:
				instanceID, dimension, err := consumer.handleMessage(ctx, msg)
				logData := log.Data{"func": "service.Start.eventLoop", "instance_id": instanceID, "dimension": dimension, "kafka_offset": msg.Offset()}
				if err != nil {
					log.Error(ctx, "event failed to process", err, logData)

					if len(instanceID) == 0 {
						log.Error(ctx, "instance_id is empty errorReporter.Notify will not be called", err, logData)
					} else {
						message := fmt.Sprintf("event failed to process, dimension is [%s]", dimension)
						err = consumer.Service.ErrorReporter.Notify(instanceID, message, err)
						if err != nil {
							log.Error(ctx, "ErrorProducer.Notify returned an error", err, logData)
						}
					}
				} else {
					log.Info(ctx, "event successfully processed", logData)
				}

				msg.CommitAndRelease()

			case eventClose := <-consumer.closing:
				log.Info(eventClose.ctx, "closing event consumer loop")
				close(consumer.closing)
				return
			}
		}
	}()
}

// Close safely closes the consumer and releases all resources
func (consumer *Consumer) Close(ctx context.Context) (err error) {

	if ctx == nil {
		ctx = context.Background()
	}

	consumer.closing <- eventClose{ctx: ctx}

	select {
	case <-consumer.closed:
		log.Info(ctx, "successfully closed event consumer")
		return nil
	case <-ctx.Done():
		log.Info(ctx, "shutdown context time exceeded, skipping graceful shutdown of event consumer")
		return errors.New("Shutdown context timed out")
	}
}
