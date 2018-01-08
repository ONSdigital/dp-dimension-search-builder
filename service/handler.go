package service

import (
	"github.com/ONSdigital/dp-hierarchy-api/models"
	"github.com/ONSdigital/dp-search-builder/elastic"
	"github.com/ONSdigital/dp-search-builder/hierarchy"
	"github.com/ONSdigital/dp-search-builder/schema"
	"github.com/ONSdigital/go-ns/kafka"
	"github.com/ONSdigital/go-ns/log"
	"golang.org/x/net/context"
)

type hierarchyBuilder struct {
	Dimension  string `avro:"file_url"`
	InstanceID string `avro:"instance_id"`
}

type searchBuilder struct {
	Dimension  string `avro:"file_url"`
	InstanceID string `avro:"instance_id"`
}

// handleMessage handles a message by requesting dimension option data from the
// hierarchy API and sending data into search index before producing a new
// message to confirm successful completion
func (svc *Service) handleMessage(ctx context.Context, message kafka.Message) (string, string, error) {

	event, err := readMessage(message.GetData())
	if err != nil {
		log.Error(err, log.Data{"schema": "failed to unmarshal event"})
		return "", "", err
	}

	instanceID := event.InstanceID
	dimension := event.Dimension

	// Make request to Hierarchy API to get "Super Parent" for dimension
	// hierarchy and add super parent to elastic
	hierarchyAPI := hierarchy.NewHierarchyAPI(svc.HTTPClient, svc.HierarchyAPIURL)
	rootDimensionOption, err := hierarchyAPI.GetRootDimenisionOption(ctx, instanceID, dimension)
	if err != nil {
		log.Error(err, log.Data{"instance_id": instanceID, "dimension": dimension})
		return instanceID, dimension, err
	}

	// TODO Create instance dimension index with mappings/settings in elastic
	//      (if it exists already, overwrite but log out that you are overwriting)
	elasticAPI := elastic.NewElasticSearchAPI(svc.HTTPClient, svc.ElasticSearchURL)
	apiStatus, err := elasticAPI.CreateSearchIndex(ctx, instanceID, dimension)
	if err != nil {
		log.Error(err, log.Data{"status": apiStatus, "instance_id": instanceID, "dimension": dimension})
		return instanceID, dimension, err
	}

	dimensionOption := elastic.DimensionOption{
		Code:             rootDimensionOption.Links["code"].ID,
		HasData:          rootDimensionOption.HasData,
		Label:            rootDimensionOption.Label,
		NumberOfChildren: rootDimensionOption.NoOfChildren,
		URL:              rootDimensionOption.Links["code"].HRef,
	}

	// Add parent document to index
	apiStatus, err = elasticAPI.AddDimensionOption(ctx, instanceID, dimension, dimensionOption)
	if err != nil {
		log.Error(err, log.Data{"status": apiStatus, "instance_id": instanceID, "dimension": dimension})
		return instanceID, dimension, err
	}

	// loop through children to retrieve codeID's
	for _, child := range rootDimensionOption.Children {
		codeID := child.Links["self"].ID

		if err = svc.addChildrenToSearchIndex(ctx, elasticAPI, hierarchyAPI, instanceID, dimension, codeID); err != nil {
			log.Error(err, log.Data{"instance_id": instanceID, "dimension": dimension, "code_id": codeID})
			return instanceID, dimension, err
		}
	}

	produceMessage, err := schema.SearchIndexBuiltSchema.Marshal(&searchBuilder{
		Dimension:  dimension,
		InstanceID: instanceID,
	})
	if err != nil {
		return instanceID, dimension, err
	}

	// TODO Once completed with no errors, then write new message to producer
	//      `search-index-built` topic
	svc.Producer.Output() <- produceMessage

	return instanceID, dimension, nil
}

func readMessage(eventValue []byte) (*hierarchyBuilder, error) {
	var h hierarchyBuilder

	if err := schema.HierarchyBuiltSchema.Unmarshal(eventValue, &h); err != nil {
		return nil, err
	}

	return &h, nil
}

func (svc *Service) addChildrenToSearchIndex(ctx context.Context, elasticAPI *elastic.API, hierarchyAPI *hierarchy.API, instanceID, dimension, codeID string) error {
	// Get a child document for dimension hierarchy
	dimensionOption, err := hierarchyAPI.GetDimenisionOption(ctx, instanceID, dimension, codeID)
	if err != nil {
		log.Error(err, log.Data{"instance_id": instanceID, "dimension": dimension, "code_id": codeID}) // Possibly want to log this out higher up the tree
		return err
	}

	esDimensionOption := elastic.DimensionOption{
		Code:             dimensionOption.Links["code"].ID,
		HasData:          dimensionOption.HasData,
		Label:            dimensionOption.Label,
		NumberOfChildren: dimensionOption.NoOfChildren,
		URL:              dimensionOption.Links["code"].HRef,
	}

	// TODO Add child document to index
	apiStatus, err := elasticAPI.AddDimensionOption(ctx, instanceID, dimension, esDimensionOption)
	if err != nil {
		log.Error(err, log.Data{"status": apiStatus, "instance_id": instanceID, "dimension": dimension})
		return err
	}

	// Iterate through children and make request to get their data and add to
	// elastic index. This should keep looping through next set of children
	// until there are no children left
	if err = svc.iterateOverChildren(ctx, elasticAPI, hierarchyAPI, instanceID, dimension, dimensionOption.Children); err != nil {
		return err
	}

	return nil
}

func (svc *Service) iterateOverChildren(ctx context.Context, elasticAPI *elastic.API, hierarchyAPI *hierarchy.API, instanceID, dimension string, children []*models.Element) error {
	for _, child := range children {
		codeID := child.Links["self"].ID

		if err := svc.addChildrenToSearchIndex(ctx, elasticAPI, hierarchyAPI, instanceID, dimension, codeID); err != nil {
			log.Error(err, log.Data{"instance_id": instanceID, "dimension": dimension, "code_id": codeID})
			return err
		}
	}

	return nil
}
