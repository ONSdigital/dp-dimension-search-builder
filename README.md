dp-search-builder
==================

Handles inserting of dimension options into elasticsearch once a hierarchy for an instance becomes available;
and creates an event by sending a message to the search-built kafka topic so services know when the data has successfully been inserted into elastic.

1. Consumes from the HIERARCHY_BUILT_TOPIC
2. Retrieves super parent dimension option for hierarchy via the hierarchy API.
3. Creates elastic search index `/<instance_id>_>dimension` and adds parent dimension option.
4. Retrieves all descendants (children) of the super parent and writes data to elastic search index
5. Produces a message to the SEARCH_BUILT_TOPIC

Requirements
-----------------
In order to run the service locally you will need the following:
- [Go](https://golang.org/doc/install)
- [Git](https://git-scm.com/downloads)
- [Kafka](https://kafka.apache.org/)
- [ElasticSearch](https://www.elastic.co/guide/en/elasticsearch/reference/5.4/index.html)
- [Hierarchy API](https://github.com/ONSdigital/dp-hierarchy-api)

### Getting started

* Clone the repo `go get github.com/ONSdigital/dp-search-builder`
* Run kafka and zookeeper
* Run elasticsearch
* Run the hierarchy API, see documentation [here](https://github.com/ONSdigital/dp-hierarchy-api)
* Run the application `make debug`

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
| CONSUMER_TOPIC             | hierarhy-built                       | The name of the topic to consumes messages from
| ELASTIC_SEARCH_URL         | http://localhost:9200                | The host name for elasticsearch
| EVENT_REPORTER_TOPIC       | report-events                        | The kafka topic to send errors to
| GRACEFUL_SHUTDOWN_TIMEOUT  | 5s                                   | The graceful shutdown timeout
| HEALTHCHECK_INTERVAL       | 60s                                  | The interval between healthchecks
| HEALTHCHECK_TIMEOUT        | 2s                                   | The timeout that the healthcheck allows for checked subsystems
| HIERARCHY_API_URL          | http://localhost:22600               | The host name for the Hierarchy API
| KAFKA_ADDR                 | localhost:9200                       | A list of Kafka host addresses
| KAFKA_MAX_BYTES            | 2000000                              | The max message size for kafka producer
| REQUEST_MAX_RETRIES        | 3                                    | The maximum number of attempts for a single http request due to external service failure
| PRODUCER_TOPIC             | search-built                         | The kafka topic to write messages to
| SEARCH_BUILDER_URL         | http://localhost:22900               | The host name for the search builder


### Contributing

See [CONTRIBUTING](CONTRIBUTING.md) for details.

### License

Copyright © 2016-2017, Office for National Statistics (https://www.ons.gov.uk)

Released under MIT license, see [LICENSE](LICENSE.md) for details
