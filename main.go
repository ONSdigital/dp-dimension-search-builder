package main

import (
	"context"
	"os"
	"strconv"

	"github.com/ONSdigital/dp-reporter-client/reporter"
	"github.com/ONSdigital/dp-search-builder/config"
	"github.com/ONSdigital/dp-search-builder/elasticsearch"
	"github.com/ONSdigital/dp-search-builder/service"
	"github.com/ONSdigital/go-ns/kafka"
	"github.com/ONSdigital/go-ns/log"
	"github.com/ONSdigital/go-ns/rchttp"
)

func main() {
	log.Namespace = "dp-search-builder"

	cfg, err := config.Get()
	if err != nil {
		log.Error(err, nil)
		os.Exit(1)
	}

	envMax, err := strconv.ParseInt(cfg.KafkaMaxBytes, 10, 32)
	if err != nil {
		log.ErrorC("encountered error parsing kafka max bytes", err, nil)
		os.Exit(1)
	}

	syncConsumerGroup, err := kafka.NewSyncConsumer(cfg.Brokers, cfg.ConsumerTopic, cfg.ConsumerGroup, kafka.OffsetOldest)
	if err != nil {
		log.ErrorC("could not obtain consumer", err, nil)
		os.Exit(1)
	}

	searchBuiltProducer, err := kafka.NewProducer(cfg.Brokers, cfg.ProducerTopic, int(envMax))
	if err != nil {
		log.Error(err, nil)
		os.Exit(1)
	}

	searchBuilderErrProducer, err := kafka.NewProducer(cfg.Brokers, cfg.EventReporterTopic, int(envMax))
	if err != nil {
		log.Error(err, nil)
		os.Exit(1)
	}

	client := rchttp.DefaultClient
	elasticSearchAPI := elasticsearch.NewElasticSearchAPI(client, cfg.ElasticSearchAPIURL)
	_, status, err := elasticSearchAPI.CallElastic(context.Background(), cfg.ElasticSearchAPIURL, "GET", nil)
	if err != nil {
		log.ErrorC("failed to start up, unable to connect to elastic search instance", err, log.Data{"http_status": status})
		os.Exit(1)
	}

	errorReporter, err := reporter.NewImportErrorReporter(searchBuilderErrProducer, log.Namespace)
	if err != nil {
		log.ErrorC("error while attempting to create error reporter client", err, nil)
		os.Exit(1)
	}

	svc := &service.Service{
		BindAddr:            cfg.BindAddr,
		Consumer:            syncConsumerGroup,
		ElasticSearchURL:    cfg.ElasticSearchAPIURL,
		EnvMax:              envMax,
		ErrorReporter:       errorReporter,
		HealthcheckInterval: cfg.HealthcheckInterval,
		HealthcheckTimeout:  cfg.HealthcheckTimeout,
		HierarchyAPIURL:     cfg.HierarchyAPIURL,
		MaxRetries:          cfg.MaxRetries,
		SearchBuiltProducer: searchBuiltProducer,
		SearchBuilderURL:    cfg.SearchBuilderURL,
		Shutdown:            cfg.GracefulShutdownTimeout,
	}

	svc.Start()
}
