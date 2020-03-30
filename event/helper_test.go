package event

import (
	"context"
	"errors"
	"testing"

	"github.com/ONSdigital/dp-hierarchy-api/models"

	"github.com/ONSdigital/dp-search-builder/mocks"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	instanceID = "12345678"
	dimension  = "aggregate"
	codeID     = "4321"
)

func TestSuccessfullyAddChildrenToSearchIndex(t *testing.T) {
	t.Parallel()
	Convey("Successfully add single child to search index", t, func() {
		numberOfElasticCalls := 0
		numberOfHierarchyCalls := 0
		apis := &APIs{
			hierarchyAPI: &mocks.HierarchyAPI{NumberOfCalls: &numberOfHierarchyCalls},
			elasticAPI:   &mocks.ElasticAPI{NumberOfCalls: &numberOfElasticCalls},
		}
		err := apis.addChildrenToSearchIndex(context.Background(), instanceID, dimension, codeID)

		So(err, ShouldBeNil)
		So(numberOfHierarchyCalls, ShouldEqual, 1)
		So(numberOfElasticCalls, ShouldEqual, 1)
	})

	Convey("Successfully add single child and their children to search index", t, func() {
		numberOfElasticCalls := 0
		numberOfHierarchyCalls := 0
		apis := &APIs{
			hierarchyAPI: &mocks.HierarchyAPI{NumberOfDescendants: 1, NumberOfCalls: &numberOfHierarchyCalls},
			elasticAPI:   &mocks.ElasticAPI{NumberOfCalls: &numberOfElasticCalls},
		}

		err := apis.addChildrenToSearchIndex(context.Background(), instanceID, dimension, codeID)

		So(err, ShouldBeNil)
		So(numberOfHierarchyCalls, ShouldEqual, 2)
		So(numberOfElasticCalls, ShouldEqual, 2)
	})
}

func TestFailToAddChildrenToSearchIndex(t *testing.T) {
	t.Parallel()
	Convey("When the service cannot connect to hierarchy API, fail to add single child to search index", t, func() {
		numberOfElasticCalls := 0
		numberOfHierarchyCalls := 0
		apis := &APIs{
			hierarchyAPI: &mocks.HierarchyAPI{InternalServerError: true, NumberOfCalls: &numberOfHierarchyCalls},
			elasticAPI:   &mocks.ElasticAPI{NumberOfCalls: &numberOfElasticCalls},
		}
		err := apis.addChildrenToSearchIndex(context.Background(), instanceID, dimension, codeID)

		So(err, ShouldNotBeNil)
		So(err, ShouldResemble, errors.New("Internal server error"))
		So(numberOfHierarchyCalls, ShouldEqual, 1)
		So(numberOfElasticCalls, ShouldEqual, 0)
	})

	Convey("When the service cannot connect to elasticsearch, fail to add single child to search index", t, func() {
		numberOfElasticCalls := 0
		numberOfHierarchyCalls := 0
		apis := &APIs{
			hierarchyAPI: &mocks.HierarchyAPI{NumberOfCalls: &numberOfHierarchyCalls},
			elasticAPI:   &mocks.ElasticAPI{InternalServerError: true, NumberOfCalls: &numberOfElasticCalls},
		}
		err := apis.addChildrenToSearchIndex(context.Background(), instanceID, dimension, codeID)

		So(err, ShouldNotBeNil)
		So(err, ShouldResemble, errors.New("Internal server error"))
		So(numberOfHierarchyCalls, ShouldEqual, 1)
		So(numberOfElasticCalls, ShouldEqual, 1)
	})
}

func TestSuccessfullyIterateOverChildren(t *testing.T) {
	t.Parallel()
	Convey("Iterate over children where there are none and return without an error", t, func() {
		numberOfElasticCalls := 0
		numberOfHierarchyCalls := 0
		apis := &APIs{
			hierarchyAPI: &mocks.HierarchyAPI{NumberOfCalls: &numberOfHierarchyCalls},
			elasticAPI:   &mocks.ElasticAPI{NumberOfCalls: &numberOfElasticCalls},
		}
		child := &models.Element{}
		err := apis.iterateOverChildren(context.Background(), instanceID, dimension, []*models.Element{child})

		So(err, ShouldBeNil)
		So(numberOfHierarchyCalls, ShouldEqual, 0)
		So(numberOfElasticCalls, ShouldEqual, 0)
	})

	Convey("Successfully iterate over a single child and add to search index", t, func() {
		numberOfElasticCalls := 0
		numberOfHierarchyCalls := 0
		apis := &APIs{
			hierarchyAPI: &mocks.HierarchyAPI{NumberOfCalls: &numberOfHierarchyCalls},
			elasticAPI:   &mocks.ElasticAPI{NumberOfCalls: &numberOfElasticCalls},
		}

		codeLink := models.Link{ID: "5467"}

		links := make(map[string]models.Link)
		links["code"] = codeLink

		child := &models.Element{Links: links}

		err := apis.iterateOverChildren(context.Background(), instanceID, dimension, []*models.Element{child})

		So(err, ShouldBeNil)
		So(numberOfHierarchyCalls, ShouldEqual, 1)
		So(numberOfElasticCalls, ShouldEqual, 1)
	})

	Convey("Successfully iterate over multiple children and add to search index", t, func() {
		numberOfElasticCalls := 0
		numberOfHierarchyCalls := 0
		apis := &APIs{
			hierarchyAPI: &mocks.HierarchyAPI{NumberOfCalls: &numberOfHierarchyCalls},
			elasticAPI:   &mocks.ElasticAPI{NumberOfCalls: &numberOfElasticCalls},
		}

		codeLink := models.Link{ID: "5467"}

		links := make(map[string]models.Link)
		links["code"] = codeLink

		firstChild := &models.Element{Links: links}
		secondChild := &models.Element{Links: links}

		err := apis.iterateOverChildren(context.Background(), instanceID, dimension, []*models.Element{firstChild, secondChild})

		So(err, ShouldBeNil)
		So(numberOfHierarchyCalls, ShouldEqual, 2)
		So(numberOfElasticCalls, ShouldEqual, 2)
	})
}

func TestFailToIterateOverChildren(t *testing.T) {
	t.Parallel()
	Convey("When the service cannot connect to elasticsearch, fail to add single child to search index", t, func() {
		numberOfElasticCalls := 0
		numberOfHierarchyCalls := 0
		apis := &APIs{
			hierarchyAPI: &mocks.HierarchyAPI{NumberOfCalls: &numberOfHierarchyCalls},
			elasticAPI:   &mocks.ElasticAPI{InternalServerError: true, NumberOfCalls: &numberOfElasticCalls},
		}

		codeLink := models.Link{ID: "5467"}

		links := make(map[string]models.Link)
		links["code"] = codeLink

		child := &models.Element{Links: links}

		err := apis.iterateOverChildren(context.Background(), instanceID, dimension, []*models.Element{child})

		So(err, ShouldNotBeNil)
		So(err, ShouldResemble, errors.New("Internal server error"))
		So(numberOfHierarchyCalls, ShouldEqual, 1)
		So(numberOfElasticCalls, ShouldEqual, 1)
	})
}
