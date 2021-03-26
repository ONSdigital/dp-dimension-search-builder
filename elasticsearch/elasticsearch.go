package elasticsearch

import (
	"context"

	"github.com/ONSdigital/dp-dimension-search-builder/models"
)

// APIer - An interface used to access the ElasticAPI
type APIer interface {
	CreateSearchIndex(ctx context.Context, instanceID, dimension string) (int, error)
	DeleteSearchIndex(ctx context.Context, instanceID, dimension string) (int, error)
	AddDimensionOption(ctx context.Context, instanceID, dimension string, dimensionOption models.DimensionOption) (int, error)
}
