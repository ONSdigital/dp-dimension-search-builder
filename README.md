dp-search-builder
==================

Refer to https://github.com/ONSdigital/dp-import#dp-import for infrastructure setup of service

### Getting started

* Run ```make debug```

### Healthcheck

The endpoint `/healthcheck` checks the connection to the database and returns
one of:

- success (200, JSON "status": "OK")
- failure (500, JSON "status": "error").

### Configuration

| Environment variable       | Default                              | Description
| -------------------------- | -------------------------------------| -----------
| BIND_ADDR                  | :22900                               | The host and port to bind to
| CONSUMER_GROUP             | dp-search-builder                    | The name of the Kafka consumer group
| CONSUMER_TOPIC             | hierarchy-built                      | The name of the Kafka consumer group
| DATASET_API_URL            | http://localhost:22000               | The host name for the Dataset API
| GRACEFUL_SHUTDOWN_TIMEOUT  | 5s                                   | The graceful shutdown timeout
| HEALTHCHECK_TIMEOUT        | 2s                                   | The timeout that the healthcheck allows for checked subsystems
| HIERARCHY_API_URL          | http://localhost:22600               | The host name for the Hierarchy API
| HIERARCHY_BUILT_TOPIC      | hierarchy-built                      | The name of the topic to consumes
| KAFKA_ADDR                 | localhost:9092                       | A list of Kafka host addresses
| KAFKA_MAX_BYTES            | 2000000                              | The max message size for kafka producer
| PRODUCER_TOPIC             | search-built                         | The name of the topic to produces messages to
| REQUEST_MAX_RETRIES        | 3                                    | The maximum number of request retries messages from
| SEARCH_BUILDER_URL         | http://localhost:22900               | The host name for the service


### Contributing

See [CONTRIBUTING](CONTRIBUTING.md) for details.

### License

Copyright © 2016-2017, Office for National Statistics (https://www.ons.gov.uk)

Released under MIT license, see [LICENSE](LICENSE.md) for details
