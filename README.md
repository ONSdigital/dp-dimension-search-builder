# dp-dimension-search-builder

Handles inserting of dimension options into elasticsearch once a hierarchy for an instance becomes available;
and creates an event by sending a message to the kafka `$PRODUCER_TOPIC` so services know when the data has successfully been inserted into elastic.

1. Consumes from the `$HIERARCHY_BUILT_TOPIC`
2. Retrieves the root node of the hierarchy via the hierarchy API, to get the root dimension option
3. Creates elastic search index `/<instance_id>_<dimension>` and adds parent dimension option
4. Retrieves all nodes in the tree below the root node and writing the data to the elasticsearch index
5. Produces a message to the `$SEARCH_BUILT_TOPIC`

## Requirements

In order to run the service locally you will need the following:
- [Go](https://golang.org/doc/install)
- [Git](https://git-scm.com/downloads)
- [Kafka](https://kafka.apache.org/)
- [ElasticSearch](https://www.elastic.co/guide/en/elasticsearch/reference/5.4/index.html)
- [Hierarchy API](https://github.com/ONSdigital/dp-hierarchy-api)

Refer to https://github.com/ONSdigital/dp-import#dp-import for infrastructure setup of service

## Getting started

* Clone the repo `go get github.com/ONSdigital/dp-dimension-search-builder`
* Run zookeeper and then kafka
* Run elasticsearch
* Run the hierarchy API, see documentation [here](https://github.com/ONSdigital/dp-hierarchy-api)
* Run the application `make debug`

### Healthcheck

The endpoint `/healthcheck` checks the connection to the database and returns
one of:

- success (200, JSON "status": "OK")
- failure (500, JSON "status": "error").

### Kafka scripts

Scripts for updating and debugging Kafka can be found [here](https://github.com/ONSdigital/dp-data-tools)(dp-data-tools)

### Configuration

| Environment variable         | Default                              | Description
| ---------------------------- | -------------------------------------| -----------
| BIND_ADDR                    | :22900                               | The host and port to bind to
| CONSUMER_GROUP               | dp-dimension-search-builder          | The name of the Kafka consumer group
| ELASTIC_SEARCH_URL           | http://localhost:10200               | The host name for elasticsearch
| EVENT_REPORTER_TOPIC         | report-events                        | The kafka topic to send errors to
| GRACEFUL_SHUTDOWN_TIMEOUT    | 5s                                   | The graceful shutdown timeout
| HEALTHCHECK_INTERVAL         | 30s                                  | The time between calling healthcheck endpoints for check subsystems
| HEALTHCHECK_CRITICAL_TIMEOUT | 90s                                  | The time taken for the health changes from warning state to critical due to subsystem check failures
| HIERARCHY_API_URL            | http://localhost:22600               | The host name for the Hierarchy API
| HIERARCHY_BUILT_TOPIC        | hierarchy-built                      | The name of the topic to consume messages from
| PRODUCER_TOPIC               | dimension-search-built               | The name of the topic to produces messages to
| KAFKA_ADDR                   | localhost:9092                       | A list of Kafka host addresses
| KAFKA_MAX_BYTES              | 2000000                              | The max message size for kafka producer
| KAFKA_VERSION                | "1.0.2"                              | The kafka version that this service expects to connect to
| KAFKA_OFFSET_OLDEST          | true                                 | sets kafka offset to oldest if `true`
| KAFKA_SEC_PROTO              | _unset_                              | if set to `TLS`, kafka connections will use TLS [1]
| KAFKA_SEC_CLIENT_KEY         | _unset_                              | PEM for the client key [1]
| KAFKA_SEC_CLIENT_CERT        | _unset_                              | PEM for the client certificate [1]
| KAFKA_SEC_CA_CERTS           | _unset_                              | CA cert chain for the server cert [1]
| KAFKA_SEC_SKIP_VERIFY        | false                                | ignores server certificate issues if `true` [1]
| REQUEST_MAX_RETRIES          | 3                                    | The maximum number of request retries messages from
| SEARCH_BUILDER_URL           | http://localhost:22900               | The host name for the service
| SIGN_ELASTICSEARCH_REQUESTS  | false                                | Boolean flag to identify whether elasticsearch requests via elastic API need to be signed if elasticsearch cluster is running in aws

**Notes:**

1. For more info, see the [kafka TLS examples documentation](https://github.com/ONSdigital/dp-kafka/tree/main/examples#tls)

### Contributing

See [CONTRIBUTING](CONTRIBUTING.md) for details.

### License

Copyright © 2016-2021, Office for National Statistics (https://www.ons.gov.uk)

Released under MIT license, see [LICENSE](LICENSE.md) for details
