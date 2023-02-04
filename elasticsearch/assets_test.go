package elasticsearch_test

import (
	"encoding/json"
	"testing"

	"github.com/ONSdigital/dp-dimension-search-builder/elasticsearch"
	. "github.com/smartystreets/goconvey/convey"
)

func TestGetMappings_ValidJson(t *testing.T) {
	Convey("File `mappings.json` is valid jason", t, func() {
		Convey("When we get the mappings json", func() {
			mappingsJSON := elasticsearch.GetMappingsJSON()
			Convey("Then the json returned should be valid", func() {
				So(json.Valid(mappingsJSON), ShouldBeTrue)
			})
		})
	})
}
