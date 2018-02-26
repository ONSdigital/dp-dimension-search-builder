package models

// DimensionOption represents the json structure for loading a single document into elastic
type DimensionOption struct {
	Code             string `json:"code"`
	HasData          bool   `json:"has_data"`
	Label            string `json:"label"`
	NumberOfChildren int64  `json:"number_of_children"`
	URL              string `json:"url,omitempty"`
}
