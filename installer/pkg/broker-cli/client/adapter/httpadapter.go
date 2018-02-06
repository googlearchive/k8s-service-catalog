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
	URL      string
	Project  string
	Name     string
	Title    string
	Catalogs []string
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
		Name:     fmt.Sprintf("projects/%s/brokers/%s", params.Project, params.Name),
		Title:    params.Title,
		Catalogs: params.Catalogs,
	}
	postBody, err := json.Marshal(broker)
	if err != nil {
		return 0, nil, fmt.Errorf("error marshalling the broker %v for body: %v", broker, err)
	}

	return adapter.doRequest(url, http.MethodPost, bytes.NewReader(postBody))
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
