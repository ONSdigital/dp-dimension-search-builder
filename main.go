package main

import (
	"context"
	"os"
	"strconv"

	"github.com/ONSdigital/dp-search-builder/config"
	"github.com/ONSdigital/dp-search-builder/elastic"
	"github.com/ONSdigital/dp-search-builder/service"
	"github.com/ONSdigital/go-ns/kafka"
	"github.com/ONSdigital/go-ns/log"
	"github.com/ONSdigital/go-ns/rchttp"
)

func main() {
	cfg, err := config.Get()
	if err != nil {
		log.Error(err, nil)
		os.Exit(1)
	}

	log.Namespace = "dp-search-builder"

	envMax, err := strconv.ParseInt(cfg.KafkaMaxBytes, 10, 32)
	if err != nil {
		log.ErrorC("encountered error parsing kafka max bytes", err, nil)
		os.Exit(1)
	}

	syncConsumerGroup, err := kafka.NewSyncConsumer(cfg.Brokers, cfg.ConsumerTopic, cfg.ConsumerGroup, kafka.OffsetNewest)
	if err != nil {
		log.ErrorC("could not obtain consumer", err, nil)
		os.Exit(1)
	}

	searchBuiltProducer, err := kafka.NewProducer(cfg.Brokers, cfg.ProducerTopic, int(envMax))
	if err != nil {
		log.Error(err, nil)
		os.Exit(1)
	}

	client := rchttp.DefaultClient
	elasticSearchAPI := elastic.NewElasticSearchAPI(client, cfg.ElasticSearchAPIURL)
	_, status, err := elasticSearchAPI.CallElastic(context.Background(), cfg.ElasticSearchAPIURL, "GET", nil)
	if err != nil {
		log.ErrorC("failed to start up, unable to connect to elastic search instance", err, log.Data{"http_status": status})
		os.Exit(1)
	}

	svc := &service.Service{
		BindAddr:            cfg.BindAddr,
		Consumer:            syncConsumerGroup,
		ElasticSearchURL:    cfg.ElasticSearchAPIURL,
		EnvMax:              envMax,
		HealthcheckInterval: cfg.HealthcheckInterval,
		HealthcheckTimeout:  cfg.HealthcheckTimeout,
		HierarchyAPIURL:     cfg.HierarchyAPIURL,
		MaxRetries:          cfg.MaxRetries,
		Producer:            searchBuiltProducer,
		Shutdown:            cfg.GracefulShutdownTimeout,
	}

	svc.Start()
}
