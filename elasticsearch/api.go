package elasticsearch

import (
	"context"
	"encoding/json"
	"errors"

	es "github.com/ONSdigital/dp-elasticsearch"
	rchttp "github.com/ONSdigital/dp-rchttp"
	"github.com/ONSdigital/dp-search-builder/models"
	"github.com/ONSdigital/log.go/log"
)

// ErrorUnexpectedStatusCode represents the error message to be returned when
// the status received from elastic is not as expected
var ErrorUnexpectedStatusCode = errors.New("unexpected status code from api")

// API aggregates a client and URL and other common data for accessing the API
type API struct {
	clienter            rchttp.Clienter
	elasticSearchClient *es.Client
}

// NewElasticSearchAPI creates an ElasticSearchAPI object
func NewElasticSearchAPI(clienter rchttp.Clienter, elasticSearchClient *es.Client) *API {

	return &API{
		clienter:            clienter,
		elasticSearchClient: elasticSearchClient,
	}
}

// CreateSearchIndex creates a new index in elastic search
func (api *API) CreateSearchIndex(ctx context.Context, instanceID, dimension string) (int, error) {
	path := "/" + instanceID + "_" + dimension

	indexMappings, err := Asset("mappings.json")
	if err != nil {
		return 0, err
	}

	status, err := api.elasticSearchClient.CreateIndex(ctx, path, indexMappings)
	if err != nil {
		return status, err
	}

	return status, nil
}

// DeleteSearchIndex removes an index from elastic search
func (api *API) DeleteSearchIndex(ctx context.Context, instanceID, dimension string) (int, error) {
	path := "/" + instanceID + "_" + dimension

	status, err := api.elasticSearchClient.DeleteIndex(ctx, path)
	if err != nil {
		return status, err
	}

	return status, nil
}

// AddDimensionOption adds a document to an elastic search index
func (api *API) AddDimensionOption(ctx context.Context, instanceID, dimension string, dimensionOption models.DimensionOption) (int, error) {
	log.Event(ctx, "adding dimension option", log.INFO, log.Data{"dimension_option": dimensionOption})
	if dimensionOption.Code == "" {
		return 0, errors.New("missing dimension option code")
	}

	path := "/" + instanceID + "_" + dimension + "/dimension_option/" + dimensionOption.Code

	bytes, err := json.Marshal(dimensionOption)
	if err != nil {
		return 0, err
	}

	status, err := api.elasticSearchClient.AddDocument(ctx, path, bytes)
	if err != nil {
		return status, err
	}

	return status, nil
}
