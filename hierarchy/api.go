package hierarchy

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/ONSdigital/dp-hierarchy-api/models"
	dphttp "github.com/ONSdigital/dp-net/http"
	"github.com/ONSdigital/log.go/v2/log"
)

// API aggregates a client and URL and other common data for accessing the API
type API struct {
	clienter dphttp.Clienter
	url      string
}

// NewHierarchyAPI creates an HierarchyAPI object
func NewHierarchyAPI(clienter dphttp.Clienter, hierarchyAPIURL string) *API {
	return &API{
		clienter: clienter,
		url:      hierarchyAPIURL,
	}
}

// A list of errors that the dataset package could return
var (
	ErrorUnexpectedStatusCode        = errors.New("unexpected status code from api")
	ErrorInstanceNotFound            = errors.New("Instance not found")
	ErrorRootDimensionOptionNotFound = errors.New("Root dimension not found")
	ErrorDimensionOptionNotFound     = errors.New("Dimension option not found")
)

const method = "GET"

// GetRootDimensionOption queries the Hierarchy API to get the root dimension option for hierarchy
func (api *API) GetRootDimensionOption(ctx context.Context, instanceID, dimension string) (rootDimensionOption *models.Response, err error) {
	path := api.url + "/hierarchies/" + instanceID + "/" + dimension
	logData := log.Data{"func": "GetRootDimensionOption", "url": path, "instance_id": instanceID, "dimension": dimension}

	jsonResult, httpCode, err := api.callHierarchyAPI(ctx, path)
	logData["http_code"] = httpCode
	logData["json_result"] = jsonResult
	if err != nil {
		log.Error(ctx, "failed to get root dimention option", err, logData)
		return nil, handleError(httpCode, err, "root dimension option")
	}

	rootDimensionOption = &models.Response{}
	if err = json.Unmarshal(jsonResult, rootDimensionOption); err != nil {
		log.Error(ctx, "failed to unmarshal root dimension option", err, logData)
		return
	}

	return
}

// GetDimensionOption queries the Hierarchy API to get a dimension option for hierarchy
func (api *API) GetDimensionOption(ctx context.Context, instanceID, dimension, codeID string) (dimensionOption *models.Response, err error) {
	path := api.url + "/hierarchies/" + instanceID + "/" + dimension + "/" + codeID
	logData := log.Data{"func": "GetDimensionOption", "url": path, "instance_id": instanceID, "dimension": dimension, "code_id": codeID}

	jsonResult, httpCode, err := api.callHierarchyAPI(ctx, path)
	logData["http_code"] = httpCode
	logData["json_result"] = jsonResult
	if err != nil {
		log.Error(ctx, "failed to get dimension option", err, logData)
		return nil, handleError(httpCode, err, "dimension option")
	}

	dimensionOption = &models.Response{}
	if err = json.Unmarshal(jsonResult, dimensionOption); err != nil {
		log.Error(ctx, "failed to unmarshal dimension option", err, logData)
		return
	}

	return
}

// callHierarchyAPI contacts the Hierarchy API returns the json body
func (api *API) callHierarchyAPI(ctx context.Context, path string) ([]byte, int, error) {
	logData := log.Data{"url": path, "method": method}

	URL, err := url.Parse(path)
	if err != nil {
		log.Error(ctx, "failed to create url for hierarchy api call", err, logData)
		return nil, 0, err
	}
	path = URL.String()
	logData["url"] = path

	req, err := http.NewRequest(method, path, nil)
	if err != nil {
		log.Error(ctx, "failed to create request for hierarchy api", err, logData)
		return nil, 0, err
	}

	resp, err := api.clienter.Do(ctx, req)
	if err != nil {
		log.Error(ctx, "failed to action hierarchy api", err, logData)
		return nil, 0, err
	}
	defer resp.Body.Close()

	logData["http_code"] = resp.StatusCode
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= 300 {
		return nil, resp.StatusCode, ErrorUnexpectedStatusCode
	}

	jsonBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error(ctx, "failed to read body from dataset api", err, logData)
		return nil, resp.StatusCode, err
	}

	return jsonBody, resp.StatusCode, nil
}

func handleError(httpCode int, err error, typ string) error {
	if err == ErrorUnexpectedStatusCode {
		switch httpCode {
		case http.StatusNotFound:
			if typ == "root dimension option" {
				return ErrorRootDimensionOptionNotFound
			}
			if typ == "dimension option" {
				return ErrorDimensionOptionNotFound
			}
			return ErrorInstanceNotFound
		}
	}

	return err
}
