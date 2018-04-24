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
	"strconv"
)

// Broker represents a Service Broker.
type Broker struct {
	Name     string   `json:"name"`
	Title    string   `json:"title,omitempty"`
	Catalogs []string `json:"catalogs"`
	URL      *string  `json:"url,omitempty"`
}

// CreateBrokerParams is used as input to CreateBroker.
type CreateBrokerParams struct {
	URL     string
	Project string
	Name    string
	Title   string
}

// GetCatalogParams is used as input to GetCatalog.
type GetCatalogParams struct {
	URL     string
	Project string
	Name    string
}

// GetCatalogResult is output of successful GetCatalog request.
type GetCatalogResult struct {
	Services []Service `json:"services"`
}

// Service corresponds to the Service Object in the Open Service Broker API.
type Service struct {
	ID string `json:"id"`
}

// BrokerError is the result of a failed broker request.
type BrokerError struct {
	// StatusCode is the HTTP status code returned by the broker.
	StatusCode int
	// ErrorDescription describes the failed result of the request.
	ErrorDescription string
	// ErrorBody is the response body returned by the broker.
	ErrorBody string
}

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

// CreateBroker creates a broker in project with the given name, title, and catalogs using the URL.
func (adapter *HttpAdapter) CreateBroker(params *CreateBrokerParams) (int, []byte, error) {
	url := fmt.Sprintf("%s/v1beta1/projects/%s/brokers", params.URL, params.Project)

	broker := Broker{
		Name:  fmt.Sprintf("projects/%s/brokers/%s", params.Project, params.Name),
		Title: params.Title,
	}
	postBody, err := json.Marshal(broker)
	if err != nil {
		return 0, nil, fmt.Errorf("error marshalling the broker %v for body: %v", broker, err)
	}

	return adapter.doRequest(url, http.MethodPost, bytes.NewReader(postBody))
}

func (adapter *HttpAdapter) GetCatalog(params *GetCatalogParams) (*GetCatalogResult, error) {
	url := fmt.Sprintf("%s/v1beta1/projects/%s/brokers/%s/v2/catalog", params.URL, params.Project, params.Name)

	statusCode, body, err := adapter.doRequest(url, http.MethodGet, nil)
	if err != nil {
		return nil, err
	}

	if statusCode != http.StatusOK {
		return nil, validateGCPFailureResponse(body, statusCode, "error fetching catalog")
	}

	res := &GetCatalogResult{}
	err = json.Unmarshal(body, res)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling response body: %s\nerror: %v", string(body), err)
	}
	return res, nil
}

// doRequst is a helper function that performs the request, reads the body, checks the status code,
// and returns an appropriate error message if anything goes wrong.
func (adapter *HttpAdapter) doRequest(url, method string, reqBody io.Reader) (int, []byte, error) {
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return 0, nil, fmt.Errorf("error creating request: %v", err)
	}
	resp, err := adapter.client.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("error executing request: %v; error: %v", req, err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, fmt.Errorf("error reading response body: %v", err)
	}

	if resp.StatusCode >= 300 || resp.StatusCode < 200 {
		return resp.StatusCode, nil, fmt.Errorf("request was not successful: %v", string(body))
	}

	return resp.StatusCode, body, nil
}

func validateGCPFailureResponse(body []byte, code int, desc string) error {
	rb := &gcpBrokerFailureResponseBody{}
	err := json.Unmarshal(body, rb)
	if err != nil {
		return &BrokerError{StatusCode: code, ErrorDescription: "error unmarshalling failure response body", ErrorBody: string(body)}
	}

	return &BrokerError{
		StatusCode:       code,
		ErrorDescription: desc,
		ErrorBody:        string(body),
	}
}

type gcpBrokerFailureResponseBody struct {
	Error *gcpBrokerError `json:"error,omitempty"`
}

type gcpBrokerError struct {
	Code    int            `json:"code,omitempty"`
	Message string         `json:"message,omitempty"`
	Status  string         `json:"status,omitempty"`
	Details []*errorDetail `json:"details,omitempty"`
}

type errorDetail struct {
	Detail *string `json:"detail,omitempty"`
}

// Error is the method inherited from "error" interface to print the error.
func (e *BrokerError) Error() string {
	code := "<nil>"
	description := "<nil>"

	if e.StatusCode != 0 {
		code = strconv.Itoa(e.StatusCode)
	}
	if e.ErrorDescription != "" {
		description = e.ErrorDescription
	}

	return fmt.Sprintf("StatusCode: %s\n Description: %s\n Details: %s", code, description, e.ErrorBody)
}
