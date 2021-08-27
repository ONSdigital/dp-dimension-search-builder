package config_test

import (
	"testing"
	"time"

	"github.com/ONSdigital/dp-dimension-search-builder/config"
	. "github.com/smartystreets/goconvey/convey"
)

func TestSpec(t *testing.T) {
	Convey("Given an environment with no environment variables set", t, func() {
		cfg, err := config.Get()

		Convey("When the config values are retrieved", func() {

			Convey("There should be no error returned", func() {
				So(err, ShouldBeNil)
			})

			Convey("The values should be set to the expected defaults", func() {
				So(cfg.BindAddr, ShouldEqual, ":22900")
				So(cfg.Brokers[0], ShouldEqual, "localhost:9092")
				So(cfg.ConsumerGroup, ShouldEqual, "dp-dimension-search-builder")
				So(cfg.ConsumerTopic, ShouldEqual, "hierarchy-built")
				So(cfg.ElasticSearchAPIURL, ShouldEqual, "http://localhost:10200")
				So(cfg.EventReporterTopic, ShouldEqual, "report-events")
				So(cfg.GracefulShutdownTimeout, ShouldEqual, 5*time.Second)
				So(cfg.HealthCheckInterval, ShouldEqual, 30*time.Second)
				So(cfg.HealthCheckCriticalTimeout, ShouldEqual, 90*time.Second)
				So(cfg.HierarchyAPIURL, ShouldEqual, "http://localhost:22600")
				So(cfg.KafkaMaxBytes, ShouldEqual, "2000000")
				So(cfg.KafkaVersion, ShouldEqual, "1.0.2")
				So(cfg.KafkaSecProtocol, ShouldEqual, "")
				So(cfg.KafkaOffsetOldest, ShouldBeTrue)
				So(cfg.ProducerTopic, ShouldEqual, "dimension-search-built")
				So(cfg.MaxRetries, ShouldEqual, 3)
				So(cfg.SearchBuilderURL, ShouldEqual, "http://localhost:22900")

			})
		})
	})
}
