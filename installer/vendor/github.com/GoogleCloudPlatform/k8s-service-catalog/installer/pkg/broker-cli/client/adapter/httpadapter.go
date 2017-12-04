// Copyright © 2017 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package adapter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"

	"github.com/GoogleCloudPlatform/k8s-service-catalog/installer/pkg/broker-cli/client/osb"
)

const acceptsIncompleteKey = "accepts_incomplete"

type DoClient interface {
	Do(*http.Request) (*http.Response, error)
}

// HttpAdapter is an implementation of Adapter that uses HTTP.
type HttpAdapter struct {
	client DoClient
}

// NewHttpAdapter returns an *HttpAdapter which uses the given client.
func NewHttpAdapter(client DoClient) *HttpAdapter {
	return &HttpAdapter{
		client: client,
	}
}

// CreateBroker creates a broker in project with the given name, title, and catalogs using the registryURL.
func (adapter *HttpAdapter) CreateBroker(params *CreateBrokerParams) ([]byte, error) {
	url := fmt.Sprintf("%s/v1alpha1/projects/%s/brokers", params.RegistryURL, params.Project)

	broker := osb.Broker{
		Name:     fmt.Sprintf("projects/%s/brokers/%s", params.Project, params.Name),
		Title:    params.Title,
		Catalogs: params.Catalogs,
	}
	postBody, err := json.Marshal(broker)
	if err != nil {
		return nil, fmt.Errorf("error marshalling the broker %v for body: %v", broker, err)
	}

	return adapter.doRequest(url, http.MethodPost, bytes.NewReader(postBody))
}

// DeleteBroker deletes the broker with the given name from the project using registryURL.
func (adapter *HttpAdapter) DeleteBroker(params *DeleteBrokerParams) ([]byte, error) {
	url := fmt.Sprintf("%s/v1alpha1/projects/%s/brokers/%s", params.RegistryURL, params.Project, params.Name)
	return adapter.doRequest(url, http.MethodDelete, nil)
}

// GetBroker retrieves the broker with the given name and project using the registryURL.
func (adapter *HttpAdapter) GetBroker(params *GetBrokerParams) ([]byte, error) {
	url := fmt.Sprintf("%s/v1alpha1/projects/%s/brokers/%s", params.RegistryURL, params.Project, params.Name)
	return adapter.doRequest(url, http.MethodGet, nil)
}

// ListBrokers lists the brokers of the given project using the registryURL.
func (adapter *HttpAdapter) ListBrokers(params *ListBrokersParams) ([]byte, error) {
	url := fmt.Sprintf("%s/v1alpha1/projects/%s/brokers", params.RegistryURL, params.Project)
	return adapter.doRequest(url, http.MethodGet, nil)
}

// CreateInstance calls the given server to provision a service instance using the request information.
func (adapter *HttpAdapter) CreateInstance(params *CreateInstanceParams) (*CreateInstanceResult, error) {
	instanceURL := fmt.Sprintf("%s/v2/service_instances/%s", params.Server, params.InstanceID)

	putBody := &osb.ProvisionRequestBody{
		ServiceID:        params.ServiceID,
		PlanID:           params.PlanID,
		Context:          params.Context,
		OrganizationGUID: params.OrganizationGUID,
		SpaceGUID:        params.SpaceGUID,
		Parameters:       params.Parameters,
	}

	putParams := url.Values{}
	putParams.Set(acceptsIncompleteKey, strconv.FormatBool(params.AcceptsIncomplete))

	respCode, respBody, err := adapter.doOSBRequest(instanceURL, http.MethodPut, putBody, putParams)
	if err != nil {
		return nil, err
	}

	switch respCode {
	case http.StatusOK, http.StatusCreated:
		// Identical service instance already exists, or service instance is provisioned synchronously.
		rb := &osb.ProvisionResponseBody{}
		err = json.Unmarshal(respBody, rb)
		if err != nil {
			return nil, fmt.Errorf("error unmarshalling response body: %s\nerror: %v", string(respBody), err)
		}
		return &CreateInstanceResult{Async: false, DashboardURL: rb.DashboardURL, OperationID: rb.Operation}, nil
	case http.StatusAccepted:
		// Service instance is being provisioned asynchronously.
		if !params.AcceptsIncomplete {
			return nil, fmt.Errorf("request shouldn't be handled asynchronously: %s", string(respBody))
		}

		rb := &osb.ProvisionResponseBody{}
		err = json.Unmarshal(respBody, rb)
		if err != nil {
			return nil, fmt.Errorf("error unmarshalling response body: %s\nerror: %v", string(respBody), err)
		}
		return &CreateInstanceResult{Async: true, DashboardURL: rb.DashboardURL, OperationID: rb.Operation}, nil
	case http.StatusBadRequest:
		return nil, fmt.Errorf("request was malformed or missing mandatory data: %s", string(respBody))
	case http.StatusConflict:
		return nil, fmt.Errorf("service instance with the same id and different attributes already exists: %s", string(respBody))
	case http.StatusUnprocessableEntity:
		return nil, fmt.Errorf("the broker only supports asynchronous requests: %s", string(respBody))
	default:
		return nil, fmt.Errorf("request was not successful: %v", string(respBody))
	}
}

// doRequst is a helper function that performs the request, reads the body, checks the status code,
// and returns an appropriate error message if anything goes wrong.
func (adapter *HttpAdapter) doRequest(url, method string, reqBody io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}
	resp, err := adapter.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %v; error: %v", req, err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	if resp.StatusCode >= 300 || resp.StatusCode < 200 {
		return nil, fmt.Errorf("request was not successful: %v", string(body))
	}

	return body, nil
}

// TODO(maqiuyu): Merge doRequest and doOSBRequest.
// doOSBRequst is a helper function that performs the OSB request, reads the response body, and
// returns the response status code and response body.
func (adapter *HttpAdapter) doOSBRequest(url, method string, reqBody interface{}, reqParams url.Values) (int, []byte, error) {
	serializedReqBody, err := json.Marshal(reqBody)
	if err != nil {
		return 0, nil, fmt.Errorf("error marshalling the request body %+v: %v", reqBody, err)
	}

	req, err := http.NewRequest(method, url, bytes.NewReader(serializedReqBody))
	if err != nil {
		return 0, nil, fmt.Errorf("error creating request: %v", err)
	}
	req.URL.RawQuery = reqParams.Encode()

	resp, err := adapter.client.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("error executing request: %v; error: %v", req, err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, fmt.Errorf("error reading response body: %v", err)
	}

	return resp.StatusCode, body, nil
}
