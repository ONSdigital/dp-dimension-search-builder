package hierarchy

import (
	"context"

	"github.com/ONSdigital/dp-hierarchy-api/models"
)

// APIer - An interface used to access the HierarchyAPI
type APIer interface {
	GetRootDimensionOption(ctx context.Context, instanceID, dimension string) (*models.Response, error)
	GetDimensionOption(ctx context.Context, instanceID, dimension, codeID string) (*models.Response, error)
}
