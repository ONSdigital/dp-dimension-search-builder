package schema

import "github.com/ONSdigital/go-ns/avro"

var hierarchyBuilt = `{
  "type": "record",
  "name": "hierarchy-built",
  "fields": [
    {"name": "instance_id", "type": "string"},
    {"name": "dimension_name", "type": "string"}
  ]
}`

// HierarchyBuiltSchema is the Avro schema for each
// hierarchy built that becomes available
var HierarchyBuiltSchema *avro.Schema = &avro.Schema{
	Definition: hierarchyBuilt,
}

var searchIndexBuilt = `{
  "type": "record",
  "name": "search-index-built",
  "fields": [
    {"name": "instance_id", "type": "string"},
    {"name": "dimension_name", "type": "string"}
  ]
}`

// SearchIndexBuiltSchema is the Avro schema for each dimension hierarchy successfuly sent to elastic
var SearchIndexBuiltSchema *avro.Schema = &avro.Schema{
	Definition: searchIndexBuilt,
}
