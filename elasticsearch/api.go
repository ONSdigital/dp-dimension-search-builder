package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"

	rchttp "github.com/ONSdigital/dp-rchttp"
	"github.com/ONSdigital/dp-search-builder/models"
	"github.com/ONSdigital/log.go/log"
	awsauth "github.com/smartystreets/go-aws-auth"
)

// ErrorUnexpectedStatusCode represents the error message to be returned when
// the status received from elastic is not as expected
var ErrorUnexpectedStatusCode = errors.New("unexpected status code from api")

// API aggregates a client and URL and other common data for accessing the API
type API struct {
	clienter     rchttp.Clienter
	url          string
	signRequests bool
}

// NewElasticSearchAPI creates an ElasticSearchAPI object
func NewElasticSearchAPI(clienter rchttp.Clienter, elasticSearchAPIURL string, signRequests bool) *API {

	return &API{
		clienter:     clienter,
		url:          elasticSearchAPIURL,
		signRequests: signRequests,
	}
}

// CreateSearchIndex creates a new index in elastic search
func (api *API) CreateSearchIndex(ctx context.Context, instanceID, dimension string) (int, error) {
	path := api.url + "/" + instanceID + "_" + dimension

	indexMappings, err := Asset("mappings.json")
	if err != nil {
		return 0, err
	}

	_, status, err := api.CallElastic(ctx, path, "PUT", indexMappings)
	if err != nil {
		return status, err
	}

	return status, nil
}

// DeleteSearchIndex removes an index from elastic search
func (api *API) DeleteSearchIndex(ctx context.Context, instanceID, dimension string) (int, error) {
	path := api.url + "/" + instanceID + "_" + dimension

	_, status, err := api.CallElastic(ctx, path, "DELETE", nil)
	if err != nil {
		return status, err
	}

	return status, nil
}

// AddDimensionOption adds a document to an elastic search index
func (api *API) AddDimensionOption(ctx context.Context, instanceID, dimension string, dimensionOption models.DimensionOption) (int, error) {
	log.Event(ctx, "adding dimension option", log.INFO, log.Data{"dimension_option": dimensionOption})
	if dimensionOption.Code == "" {
		return 0, errors.New("missing dimension option code")
	}

	path := api.url + "/" + instanceID + "_" + dimension + "/dimension_option/" + dimensionOption.Code

	bytes, err := json.Marshal(dimensionOption)
	if err != nil {
		return 0, err
	}

	_, status, err := api.CallElastic(ctx, path, "PUT", bytes)
	if err != nil {
		return status, err
	}

	return status, nil
}

// CallElastic builds a request to elastic search based on the method, path and payload
func (api *API) CallElastic(ctx context.Context, path, method string, payload interface{}) ([]byte, int, error) {
	logData := log.Data{"url": path, "method": method}

	URL, err := url.Parse(path)
	if err != nil {
		log.Event(ctx, "failed to create url for elastic call", log.ERROR, log.Error(err), logData)
		return nil, 0, err
	}
	path = URL.String()
	logData["url"] = path

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
		log.Event(ctx, "failed to create request for call to elastic", log.ERROR, log.Error(err), logData)
		return nil, 0, err
	}

	if api.signRequests {
		awsauth.Sign(req)
	}

	resp, err := api.clienter.Do(ctx, req)
	if err != nil {
		log.Event(ctx, "failed to call elastic", log.ERROR, log.Error(err), logData)
		return nil, 0, err
	}
	defer resp.Body.Close()

	logData["http_code"] = resp.StatusCode

	jsonBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Event(ctx, "failed to read response body from call to elastic", log.ERROR, log.Error(err), logData)
		return nil, resp.StatusCode, err
	}
	logData["json_body"] = string(jsonBody)
	logData["status_code"] = resp.StatusCode

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= 300 {
		log.Event(ctx, "failed", log.ERROR, log.Error(ErrorUnexpectedStatusCode), logData)
		return nil, resp.StatusCode, ErrorUnexpectedStatusCode
	}

	return jsonBody, resp.StatusCode, nil
}
