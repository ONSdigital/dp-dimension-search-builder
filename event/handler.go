package event

import (
	"context"
	"errors"

	"github.com/ONSdigital/dp-import/events"
	kafka "github.com/ONSdigital/dp-kafka"
	"github.com/ONSdigital/dp-search-builder/elasticsearch"
	"github.com/ONSdigital/dp-search-builder/hierarchy"
	"github.com/ONSdigital/dp-search-builder/models"
	"github.com/ONSdigital/log.go/log"
)

type hierarchyBuilder struct {
	Dimension  string `avro:"dimension_name"`
	InstanceID string `avro:"instance_id"`
}

type searchBuilder struct {
	Dimension  string `avro:"dimension_name"`
	InstanceID string `avro:"instance_id"`
}

// handleMessage handles a message by requesting dimension option data from the
// hierarchy API and sending data into search index before producing a new
// message to confirm successful completion
func (c *Consumer) handleMessage(ctx context.Context, message kafka.Message) (string, string, error) {
	if message == nil {
		err := errors.New("received empty message")
		return "", "", err
	}
	event, err := readMessage(message.GetData())
	if err != nil {
		log.Event(ctx, "failed to marshal event message", log.ERROR, log.Error(err))
		return "", "", err
	}

	instanceID := event.InstanceID
	dimension := event.Dimension

	apis := &APIs{
		hierarchyAPI: hierarchy.NewHierarchyAPI(c.Service.HTTPClienter, c.Service.HierarchyAPIURL),
		elasticAPI:   elasticsearch.NewElasticSearchAPI(c.Service.HTTPClienter, c.Service.ElasticSearchClient),
	}

	// Make request to Hierarchy API to get "Super Parent" for dimension
	// hierarchy and add super parent to elastic
	rootDimensionOption, err := apis.hierarchyAPI.GetRootDimensionOption(ctx, instanceID, dimension)
	if err != nil {
		log.Event(ctx, "failed request to hierarchy api", log.ERROR, log.Error(err), log.Data{"instance_id": instanceID, "dimension": dimension})
		return instanceID, dimension, err
	}

	// Create instance dimension index with mappings/settings in elastic
	// delete index if it already exists
	apiStatus, err := apis.elasticAPI.DeleteSearchIndex(ctx, instanceID, dimension)
	if err != nil {
		if apiStatus != 404 {
			log.Event(ctx, "unable to remove index before creating new one", log.ERROR, log.Error(err), log.Data{"status": apiStatus, "instance_id": instanceID, "dimension": dimension})
			return instanceID, dimension, err
		}
	} else {
		log.Event(ctx, "index removed before creating new one", log.INFO, log.Data{"status": apiStatus, "instance_id": instanceID, "dimension": dimension})
	}

	// create index
	apiStatus, err = apis.elasticAPI.CreateSearchIndex(ctx, instanceID, dimension)
	if err != nil {
		log.Event(ctx, "failed to create search index", log.ERROR, log.Error(err), log.Data{"status": apiStatus, "instance_id": instanceID, "dimension": dimension})
		return instanceID, dimension, err
	}

	dimensionOption := models.DimensionOption{
		Code:             rootDimensionOption.Links["code"].ID,
		HasData:          rootDimensionOption.HasData,
		Label:            rootDimensionOption.Label,
		NumberOfChildren: rootDimensionOption.NoOfChildren,
		URL:              rootDimensionOption.Links["code"].HRef,
	}

	// Add root node document to index
	apiStatus, err = apis.elasticAPI.AddDimensionOption(ctx, instanceID, dimension, dimensionOption)
	if err != nil {
		log.Event(ctx, "failed to add root (super parent) dimension option", log.ERROR, log.Error(err), log.Data{"status": apiStatus, "instance_id": instanceID, "dimension": dimension})
		return instanceID, dimension, err
	}

	// loop through children to retrieve codeID's
	for _, child := range rootDimensionOption.Children {
		codeID := child.Links["code"].ID

		if err = apis.addChildrenToSearchIndex(ctx, instanceID, dimension, codeID); err != nil {
			log.Event(ctx, "failed to add children dimension options", log.ERROR, log.Error(err), log.Data{"instance_id": instanceID, "dimension": dimension, "code_id": codeID})
			return instanceID, dimension, err
		}
	}

	produceMessage, err := events.SearchIndexBuiltSchema.Marshal(&searchBuilder{
		Dimension:  dimension,
		InstanceID: instanceID,
	})
	if err != nil {
		return instanceID, dimension, err
	}

	// Once completed with no errors, then write new message to producer
	// `search-index-built` topic
	c.Service.SearchBuiltProducer.Channels().Output <- produceMessage

	return instanceID, dimension, nil
}

func readMessage(eventValue []byte) (*hierarchyBuilder, error) {
	var h hierarchyBuilder

	if err := events.HierarchyBuiltSchema.Unmarshal(eventValue, &h); err != nil {
		return nil, err
	}

	return &h, nil
}
