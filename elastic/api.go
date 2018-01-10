package elastic

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/ONSdigital/go-ns/log"
	"github.com/ONSdigital/go-ns/rchttp"
)

// ErrorUnexpectedStatusCode represents the error message to be returned when
// the status received from elastic is not as expected
var ErrorUnexpectedStatusCode = errors.New("unexpected status code from api")

// DimensionOption represents the json structure for loading a single document into elastic
type DimensionOption struct {
	Code             string `json:"code"`
	HasData          bool   `json:"has_data"`
	Label            string `json:"label"`
	NumberOfChildren int64  `json:"number_of_children"`
	URL              string `json:"url,omitempty"`
}

// API aggregates a client and URL and other common data for accessing the API
type API struct {
	client *rchttp.Client
	url    string
}

// NewElasticSearchAPI creates an HierarchyAPI object
func NewElasticSearchAPI(client *rchttp.Client, elasticSearchAPIURL string) *API {
	return &API{
		client: client,
		url:    elasticSearchAPIURL,
	}
}

// CreateSearchIndex ...
func (api *API) CreateSearchIndex(ctx context.Context, instanceID, dimension string) (int, error) {
	path := api.url + "/" + instanceID + "_" + dimension

	indexMappings, err := ioutil.ReadFile("./mappings.json")
	if err != nil {
		return 0, err
	}

	_, status, err := api.CallElastic(ctx, path, "PUT", indexMappings)
	if err != nil {
		return status, err
	}

	return status, nil
}

// AddDimensionOption ...
func (api *API) AddDimensionOption(ctx context.Context, instanceID, dimension string, dimensionOption DimensionOption) (int, error) {
	path := api.url + "/" + instanceID + "_" + dimension + "/_create"

	bytes, err := json.Marshal(dimensionOption)
	if err != nil {
		return 0, err
	}

	_, status, err := api.CallElastic(ctx, path, "POST", bytes)
	if err != nil {
		return status, err
	}

	return status, nil
}

// CallElastic ...
func (api *API) CallElastic(ctx context.Context, path, method string, payload interface{}) ([]byte, int, error) {
	logData := log.Data{"URL": path, "method": method}

	URL, err := url.Parse(path)
	if err != nil {
		log.ErrorC("failed to create url for elastic call", err, logData)
		return nil, 0, err
	}
	path = URL.String()
	logData["URL"] = path

	var req *http.Request

	if payload != nil {
		req, err = http.NewRequest(method, path, bytes.NewReader(payload.([]byte)))
		req.Header.Add("Content-type", "application/json")
		logData["payload"] = string(payload.([]byte))
	} else {
		req, err = http.NewRequest(method, path, nil)
	}
	// check req, above, didn't error
	if err != nil {
		log.ErrorC("failed to create request for call to elastic", err, logData)
		return nil, 0, err
	}

	resp, err := api.client.Do(ctx, req)
	if err != nil {
		log.ErrorC("Failed to call elastic", err, logData)
		return nil, 0, err
	}
	defer resp.Body.Close()

	logData["httpCode"] = resp.StatusCode
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= 300 {
		return nil, resp.StatusCode, ErrorUnexpectedStatusCode
	}

	jsonBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.ErrorC("failed to read response body from call to elastic", err, logData)
		return nil, resp.StatusCode, err
	}

	return jsonBody, resp.StatusCode, nil
}
