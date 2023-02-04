package elasticsearch

import _ "embed"

//go:embed mappings.json
var mappingsJSON []byte

func GetMappingsJSON() []byte {
	return mappingsJSON
}
