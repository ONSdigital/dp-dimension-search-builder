package mocks

import (
	"context"

	"github.com/ONSdigital/dp-hierarchy-api/models"
)

// HierarchyAPI represents a list of error flags to set error in mocked hierarchy API
type HierarchyAPI struct {
	InternalServerError bool
	NumberOfDescendants int
	NumberOfCalls       *int
}

// GetRootDimensionOption represents the mocked version of getting the root dimension option for a hierarchy from the hierarchy API
func (api *HierarchyAPI) GetRootDimensionOption(ctx context.Context, instanceID, dimension string) (rootDimensionOption *models.Response, err error) {
	*api.NumberOfCalls++
	if api.InternalServerError {
		return nil, errorInternalServer
	}

	return &models.Response{}, nil
}

// GetDimensionOption represents the mocked version of getting the root dimension option for a hierarchy from the hierarchy API
func (api *HierarchyAPI) GetDimensionOption(ctx context.Context, instanceID, dimension, codeID string) (rootDimensionOption *models.Response, err error) {
	*api.NumberOfCalls++
	if api.InternalServerError {
		return nil, errorInternalServer
	}

	if api.NumberOfDescendants != 0 {

		codeLink := models.Link{
			ID:   "5432",
			HRef: "http://localhost:9090/hierarchiy-api/hierarchies/" + instanceID + "/SpecialAggregate/5432",
		}

		selfLink := models.Link{
			HRef: "http://localhost:9090/hierarchiy-api/hierarchies/" + instanceID + "/SpecialAggregate/5432",
		}

		links := make(map[string]models.Link)
		links["code"] = codeLink
		links["self"] = selfLink

		child := &models.Element{
			Label:        "Special Aggregate",
			NoOfChildren: 0,
			Links:        links,
			HasData:      true,
		}

		children := []*models.Element{child}

		// Lower the number of descendants
		api.NumberOfDescendants--

		return &models.Response{
			Children: children,
		}, nil
	}

	return &models.Response{}, nil
}
