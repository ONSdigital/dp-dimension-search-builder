dp-search-builder
==================

### Getting started

* Run ```brew install kafka```

### Healthcheck

The endpoint `/healthcheck` checks the connection to the database and returns
one of:

- success (200, JSON "status": "OK")
- failure (500, JSON "status": "error").

### Configuration

| Environment variable       | Default                              | Description
| -------------------------- | -------------------------------------| -----------
| BIND_ADDR                  | :22900                               | The host and port to bind to
| HIERARCHY_API_URL          | http://localhost:22600               | The host name for the Hierarchy API
| DATASET_API_URL            | http://localhost:22000               | The host name for the Dataset API
| KAFKA_ADDR                 | localhost:9092                       | A list of Kafka host addresses
| CONSUMER_GROUP             | dp-search-builder                    | The name of the Kafka consumer group
| CONSUMER_TOPIC             | hierarhy-built                       | The name of the topic to consumes messages from
| GRACEFUL_SHUTDOWN_TIMEOUT  | 5s                                   | The graceful shutdown timeout
| HEALTHCHECK_TIMEOUT        | 2s                                   | The timeout that the healthcheck allows for checked subsystems

### Contributing

See [CONTRIBUTING](CONTRIBUTING.md) for details.

### License

Copyright © 2016-2017, Office for National Statistics (https://www.ons.gov.uk)

Released under MIT license, see [LICENSE](LICENSE.md) for details
