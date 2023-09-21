package elasticsearch

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/ONSdigital/dp-dimension-search-builder/models"
	"github.com/ONSdigital/dp-elasticsearch/v2/elasticsearch"
	"github.com/ONSdigital/dp-net/v2/http"
	"github.com/ONSdigital/log.go/v2/log"
)

// ErrorUnexpectedStatusCode represents the error message to be returned when
// the status received from elastic is not as expected
var ErrorUnexpectedStatusCode = errors.New("unexpected status code from api")

// API aggregates a client and URL and other common data for accessing the API
type API struct {
	clienter            http.Clienter
	elasticSearchClient *elasticsearch.Client
}

// NewElasticSearchAPI creates an ElasticSearchAPI object
func NewElasticSearchAPI(clienter http.Clienter, elasticSearchClient *elasticsearch.Client) *API {

	return &API{
		clienter:            clienter,
		elasticSearchClient: elasticSearchClient,
	}
}

// CreateSearchIndex creates a new index in elastic search
func (api *API) CreateSearchIndex(ctx context.Context, instanceID, dimension string) (int, error) {
	indexName := instanceID + "_" + dimension

	indexMappings := GetMappingsJSON()

	status, err := api.elasticSearchClient.CreateIndex(ctx, indexName, indexMappings)
	if err != nil {
		return status, err
	}

	return status, nil
}

// DeleteSearchIndex removes an index from elastic search
func (api *API) DeleteSearchIndex(ctx context.Context, instanceID, dimension string) (int, error) {
	indexName := instanceID + "_" + dimension

	status, err := api.elasticSearchClient.DeleteIndex(ctx, indexName)
	if err != nil {
		return status, err
	}

	return status, nil
}

// AddDimensionOption adds a document to an elastic search index
func (api *API) AddDimensionOption(ctx context.Context, instanceID, dimension string, dimensionOption models.DimensionOption) (int, error) {
	log.Info(ctx, "adding dimension option", log.Data{"dimension_option": dimensionOption})
	if dimensionOption.Code == "" {
		return 0, errors.New("missing dimension option code")
	}

	indexName := instanceID + "_" + dimension
	documentID := dimensionOption.Code

	document, err := json.Marshal(dimensionOption)
	if err != nil {
		return 0, err
	}

	status, err := api.elasticSearchClient.AddDocument(ctx, indexName, "_doc", documentID, document)
	if err != nil {
		return status, err
	}

	return status, nil
}
