package config

import (
	"errors"
	"os"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGet(t *testing.T) {
	Convey("Given a clean environment", t, func() {
		os.Clearenv()
		cfg = nil

		Convey("When the config values are retrieved", func() {
			cfg, err := Get()

			Convey("Then there should be no error returned", func() {
				So(err, ShouldBeNil)

				Convey("And the values should be set to the expected defaults", func() {
					So(cfg.AwsRegion, ShouldEqual, "eu-west-1")
					So(cfg.AwsSdkSigner, ShouldBeFalse)
					So(cfg.AwsService, ShouldEqual, "es")
					So(cfg.BindAddr, ShouldEqual, ":22900")
					So(cfg.ElasticSearchAPIURL, ShouldEqual, "http://localhost:10200")
					So(cfg.GracefulShutdownTimeout, ShouldEqual, 5*time.Second)
					So(cfg.HealthCheckInterval, ShouldEqual, 30*time.Second)
					So(cfg.HealthCheckCriticalTimeout, ShouldEqual, 90*time.Second)
					So(cfg.HierarchyAPIURL, ShouldEqual, "http://localhost:22600")
					So(cfg.KafkaConfig.BindAddr[0], ShouldEqual, "localhost:9092")
					So(cfg.KafkaConfig.MaxBytes, ShouldEqual, "2000000")
					So(cfg.KafkaConfig.Version, ShouldEqual, "1.0.2")
					So(cfg.KafkaConfig.SecProtocol, ShouldEqual, "")
					So(cfg.KafkaConfig.SecCACerts, ShouldEqual, "")
					So(cfg.KafkaConfig.SecClientCert, ShouldEqual, "")
					So(cfg.KafkaConfig.SecClientKey, ShouldEqual, "")
					So(cfg.KafkaConfig.SecSkipVerify, ShouldEqual, false)
					So(cfg.KafkaConfig.OffsetOldest, ShouldBeTrue)
					So(cfg.KafkaConfig.ConsumerGroup, ShouldEqual, "dp-dimension-search-builder")
					So(cfg.KafkaConfig.ConsumerTopic, ShouldEqual, "hierarchy-built")
					So(cfg.KafkaConfig.EventReporterTopic, ShouldEqual, "report-events")
					So(cfg.KafkaConfig.ProducerTopic, ShouldEqual, "dimension-search-built")
					So(cfg.MaxRetries, ShouldEqual, 3)
					So(cfg.SearchBuilderURL, ShouldEqual, "http://localhost:22900")
					So(cfg.SignElasticsearchRequests, ShouldBeFalse)
				})
			})
		})

		Convey("When configuration is called with an invalid security setting", func() {
			defer os.Clearenv()
			os.Setenv("KAFKA_SEC_PROTO", "ssl")
			cfg, err := Get()

			Convey("Then an error should be returned", func() {
				So(cfg, ShouldBeNil)
				So(err, ShouldNotBeNil)
				So(err, ShouldResemble, errors.New("kafka config validation errors: KAFKA_SEC_PROTO has invalid value"))
			})
		})
	})

	Convey("Given config already exists", t, func() {
		cfg = getDefaultConfig()

		Convey("When the config values are retrieved", func() {
			config, err := Get()

			Convey("Then there should be no error returned", func() {
				So(err, ShouldBeNil)

				Convey("And the existing config should be returned", func() {
					So(config, ShouldResemble, cfg)
				})
			})
		})
	})
}
