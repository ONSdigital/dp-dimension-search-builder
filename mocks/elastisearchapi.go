package mocks

import (
	"context"
	"errors"

	"github.com/ONSdigital/dp-dimension-search-builder/models"
)

// ElasticAPI represents a list of error flags to set error in mocked elastic API
type ElasticAPI struct {
	InternalServerError bool
	NumberOfCalls       *int
}

var (
	errorInternalServer = errors.New("Internal server error")
)

// CreateSearchIndex represents the mocked version of creating a search index
func (api *ElasticAPI) CreateSearchIndex(ctx context.Context, instanceID, dimension string) (int, error) {
	*api.NumberOfCalls++

	if api.InternalServerError {
		return 0, errorInternalServer
	}

	return 201, nil
}

// DeleteSearchIndex represents the mocked version of deleting a search index
func (api *ElasticAPI) DeleteSearchIndex(ctx context.Context, instanceID, dimension string) (int, error) {
	*api.NumberOfCalls++
	if api.InternalServerError {
		return 0, errorInternalServer
	}

	return 200, nil
}

// AddDimensionOption represents the mocked version of adding a dimension option to an existing index
func (api *ElasticAPI) AddDimensionOption(ctx context.Context, instanceID, dimension string, dimensionOption models.DimensionOption) (int, error) {
	*api.NumberOfCalls++
	if api.InternalServerError {
		return 0, errorInternalServer
	}

	return 201, nil
}
