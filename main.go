package main

import (
	"os"
	"strconv"

	"github.com/ONSdigital/dp-search-builder/config"
	"github.com/ONSdigital/dp-search-builder/service"
	"github.com/ONSdigital/go-ns/kafka"
	"github.com/ONSdigital/go-ns/log"
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

	svc := &service.Service{
		EnvMax:             envMax,
		BindAddr:           cfg.BindAddr,
		Consumer:           syncConsumerGroup,
		HealthcheckTimeout: cfg.HealthcheckTimeout,
		HierarchyAPIURL:    cfg.HierarchyAPIURL,
		MaxRetries:         cfg.MaxRetries,
		Producer:           searchBuiltProducer,
		SearchBuilderURL:   cfg.SearchBuilderURL,
		Shutdown:           cfg.GracefulShutdownTimeout,
	}

	svc.Start()
}
