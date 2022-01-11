package event

import (
	"context"

	"github.com/ONSdigital/dp-dimension-search-builder/elasticsearch"
	"github.com/ONSdigital/dp-dimension-search-builder/hierarchy"
	"github.com/ONSdigital/dp-dimension-search-builder/models"
	hierarchyModel "github.com/ONSdigital/dp-hierarchy-api/models"
	"github.com/ONSdigital/log.go/v2/log"
)

// APIs represent a list of API interfaces used by service
type APIs struct {
	hierarchyAPI hierarchy.APIer
	elasticAPI   elasticsearch.APIer
}

func (apis *APIs) addChildrenToSearchIndex(ctx context.Context, instanceID, dimension, codeID string) error {
	// Get a child document for dimension hierarchy
	dimensionOption, err := apis.hierarchyAPI.GetDimensionOption(ctx, instanceID, dimension, codeID)
	if err != nil {
		// Possibly want to log this out higher up the tree
		log.Error(ctx, "failed to retrieve dimension option", err, log.Data{"instance_id": instanceID, "dimension": dimension, "code_id": codeID})
		return err
	}

	esDimensionOption := models.DimensionOption{
		Code:             dimensionOption.Links["code"].ID,
		HasData:          dimensionOption.HasData,
		Label:            dimensionOption.Label,
		NumberOfChildren: dimensionOption.NoOfChildren,
		URL:              dimensionOption.Links["self"].HRef,
	}

	// Add child document to index
	apiStatus, err := apis.elasticAPI.AddDimensionOption(ctx, instanceID, dimension, esDimensionOption)
	if err != nil {
		log.Error(ctx, "failed to add child document to index", err, log.Data{"status": apiStatus, "instance_id": instanceID, "dimension": dimension})
		return err
	}

	// Iterate through children and make request to get their data and add to
	// elastic index. This should keep looping through next set of children
	// until there are no children left
	if err = apis.iterateOverChildren(ctx, instanceID, dimension, dimensionOption.Children); err != nil {
		return err
	}

	return nil
}

func (apis *APIs) iterateOverChildren(ctx context.Context, instanceID, dimension string, children []*hierarchyModel.Element) error {
	for _, child := range children {
		codeID := child.Links["code"].ID
		if codeID != "" {

			if err := apis.addChildrenToSearchIndex(ctx, instanceID, dimension, codeID); err != nil {
				log.Error(ctx, "failed to add child docs to search index", err, log.Data{"instance_id": instanceID, "dimension": dimension, "code_id": codeID})
				return err
			}
		}
	}

	return nil
}
