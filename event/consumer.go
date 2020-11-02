package event

import (
	"context"
	"errors"
	"fmt"

	es "github.com/ONSdigital/dp-elasticsearch"
	kafka "github.com/ONSdigital/dp-kafka"
	rchttp "github.com/ONSdigital/dp-rchttp"
	"github.com/ONSdigital/dp-reporter-client/reporter"
	"github.com/ONSdigital/log.go/log"
)

// Consumer consumes event messages.
type Consumer struct {
	Service Service
	closing chan eventClose
	closed  chan bool
}

// Service contains service configuration for consumer
type Service struct {
	//SignElasticsearchRequests bool
	ErrorReporter       reporter.ImportErrorReporter
	HierarchyAPIURL     string
	HTTPClienter        rchttp.Clienter
	SearchBuiltProducer *kafka.Producer
	//ElasticsearchURL          string
	ElasticSearchClient *es.Client
}

type eventClose struct {
	ctx context.Context
}

// NewConsumer returns a new consumer instance.
func NewConsumer(clienter rchttp.Clienter, hierarchyAPIURL string, elasticSearchClient *es.Client,
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
func (consumer *Consumer) Consume(messageConsumer *kafka.ConsumerGroup) {

	// eventLoop
	go func() {
		defer close(consumer.closed)

		for {
			select {
			case msg := <-messageConsumer.Channels().Upstream:
				ctx := context.Background()

				instanceID, dimension, err := consumer.handleMessage(ctx, msg)
				logData := log.Data{"func": "service.Start.eventLoop", "instance_id": instanceID, "dimension": dimension, "kafka_offset": msg.Offset()}
				if err != nil {
					log.Event(ctx, "event failed to process", log.ERROR, log.Error(err), logData)

					if len(instanceID) == 0 {
						log.Event(ctx, "instance_id is empty errorReporter.Notify will not be called", log.ERROR, log.Error(err), logData)
					} else {
						message := fmt.Sprintf("event failed to process, dimension is [%s]", dimension)
						err = consumer.Service.ErrorReporter.Notify(instanceID, message, err)
						if err != nil {
							log.Event(ctx, "ErrorProducer.Notify returned an error", log.ERROR, log.Error(err), logData)
						}
					}
				} else {
					log.Event(ctx, "event successfully processed", log.INFO, logData)
				}

				messageConsumer.CommitAndRelease(msg)
			case eventClose := <-consumer.closing:
				log.Event(eventClose.ctx, "closing event consumer loop", log.INFO)
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
		log.Event(ctx, "successfully closed event consumer", log.INFO)
		return nil
	case <-ctx.Done():
		log.Event(ctx, "shutdown context time exceeded, skipping graceful shutdown of event consumer", log.INFO)
		return errors.New("Shutdown context timed out")
	}
}
