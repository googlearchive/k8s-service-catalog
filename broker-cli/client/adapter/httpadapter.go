// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
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

	"github.com/GoogleCloudPlatform/k8s-service-catalog/broker-cli/client/osb"
)

const (
	acceptsIncompleteKey = "accepts_incomplete"
	serviceIDKey         = "service_id"
	planIDKey            = "plan_id"
	operationKey         = "operation"
	instanceKey          = "instance"
	bindingKey           = "binding"
	apiVersionHeader     = "X-Broker-API-Version"
)

type DoClient interface {
	Do(*http.Request) (*http.Response, error)
}

// httpAdapter is an implementation of Adapter that uses HTTP.
type httpAdapter struct {
	client DoClient
}

// NewHttpAdapter returns an Adapter which uses the given client.
func NewHttpAdapter(client DoClient) Adapter {
	return &httpAdapter{
		client: client,
	}
}

// CreateBroker creates a broker in project with the given name, title, and catalogs using the request information.
func (adapter *httpAdapter) CreateBroker(params *CreateBrokerParams) (*osb.Broker, error) {
	url := fmt.Sprintf("%s/v1beta1/projects/%s/brokers", params.Host, params.Project)

	broker := osb.Broker{
		Name:  fmt.Sprintf("projects/%s/brokers/%s", params.Project, params.Name),
		Title: params.Title,
	}
	postBody, err := json.Marshal(broker)
	if err != nil {
		return nil, fmt.Errorf("error marshalling the broker %v for body: %v", broker, err)
	}

	ret := &osb.Broker{}
	err = adapter.doRequest(url, http.MethodPost, bytes.NewReader(postBody), ret)
	return ret, err
}

// DeleteBroker deletes the broker with the given name from the project using the request information.
func (adapter *httpAdapter) DeleteBroker(params *DeleteBrokerParams) error {
	return adapter.doRequest(params.BrokerURL, http.MethodDelete, nil, nil)
}

// ListBrokers lists the brokers in a given project using the request information.
func (adapter *httpAdapter) ListBrokers(params *ListBrokersParams) (*ListBrokersResult, error) {
	url := fmt.Sprintf("%s/v1beta1/projects/%s/brokers", params.Host, params.Project)
	ret := &ListBrokersResult{}
	err := adapter.doRequest(url, http.MethodGet, nil, ret)
	return ret, err
}

func (adapter *httpAdapter) GetCatalog(params *GetCatalogParams) (*GetCatalogResult, error) {
	url := fmt.Sprintf("%s/v2/catalog", params.Server)

	statusCode, body, err := adapter.doOSBRequest(url, http.MethodGet, params.APIVersion, nil, nil)
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

// CreateInstance calls the given server to provision a service instance using the request information.
func (adapter *httpAdapter) CreateInstance(params *CreateInstanceParams) (*CreateInstanceResult, error) {
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

	respCode, respBody, err := adapter.doOSBRequest(instanceURL, http.MethodPut, params.APIVersion, putBody, putParams)
	if err != nil {
		return nil, err
	}

	unmarshalSuccessResponse := func(body []byte, isAsync bool) (*CreateInstanceResult, error) {
		rb := &osb.ProvisionResponseBody{}
		err := json.Unmarshal(body, rb)
		if err != nil {
			return nil, fmt.Errorf("error unmarshalling response body: %s\nerror: %v", string(body), err)
		}
		return &CreateInstanceResult{Async: isAsync, DashboardURL: rb.DashboardURL, OperationID: rb.Operation}, nil
	}

	switch respCode {
	case http.StatusOK, http.StatusCreated:
		// Identical service instance already exists, or service instance is provisioned synchronously.
		return unmarshalSuccessResponse(respBody, false)
	case http.StatusAccepted:
		// Service instance is being provisioned asynchronously.
		if !params.AcceptsIncomplete {
			return nil, fmt.Errorf("request shouldn't be handled asynchronously: %s", string(respBody))
		}

		return unmarshalSuccessResponse(respBody, true)
	case http.StatusBadRequest:
		return nil, validateGCPFailureResponse(respBody, respCode, "request was malformed or missing mandatory data")
	case http.StatusConflict:
		return nil, validateGCPFailureResponse(respBody, respCode, "instance with the same id but different attributes already exists")
	case http.StatusUnprocessableEntity:
		return nil, validateGCPFailureResponse(respBody, respCode, "the broker only supports asynchronous requests")
	default:
		return nil, validateGCPFailureResponse(respBody, respCode, "request was not successful")
	}
}

// ListInstances lists instances in a given project and broker using the request information.
func (adapter *httpAdapter) ListInstances(params *ListInstancesParams) (*ListInstancesResult, error) {
	URL := fmt.Sprintf("%s/instances", params.Server)
	lir := &ListInstancesResult{}
	if err := adapter.doRequest(URL, http.MethodGet, nil, lir); err != nil {
		return nil, err
	}
	return lir, nil
}

// ListBindings lists bindings to an instance using the request information.
func (adapter *httpAdapter) ListBindings(params *ListBindingsParams) (*ListBindingsResult, error) {
	URL := fmt.Sprintf("%s/instances/%s/bindings", params.Server, params.InstanceID)
	ret := &ListBindingsResult{}
	if err := adapter.doRequest(URL, http.MethodGet, nil, ret); err != nil {
		return nil, err
	}
	return ret, nil
}

// DeleteInstance calls the given server to deprovision a service instance using the request information.
func (adapter *httpAdapter) DeleteInstance(params *DeleteInstanceParams) (*DeleteInstanceResult, error) {
	instanceURL := fmt.Sprintf("%s/v2/service_instances/%s", params.Server, params.InstanceID)

	deleteParams := url.Values{}
	deleteParams.Set(serviceIDKey, params.ServiceID)
	deleteParams.Set(planIDKey, params.PlanID)
	deleteParams.Set(acceptsIncompleteKey, strconv.FormatBool(params.AcceptsIncomplete))

	respCode, respBody, err := adapter.doOSBRequest(instanceURL, http.MethodDelete, params.APIVersion, nil, deleteParams)
	if err != nil {
		return nil, err
	}

	switch respCode {
	case http.StatusOK, http.StatusGone:
		// The service instance is deprovisioned synchronously, or it doesn't exist.
		// The response should be an empty JSON object ("{}") with no data.
		return &DeleteInstanceResult{Async: false}, nil
	case http.StatusAccepted:
		// The service instance is being deprovisioned asynchronously.
		if !params.AcceptsIncomplete {
			return nil, fmt.Errorf("request shouldn't be handled asynchronously: %s", string(respBody))
		}

		rb := &osb.DeprovisionResponseBody{}
		err := json.Unmarshal(respBody, rb)
		if err != nil {
			return nil, fmt.Errorf("error unmarshalling response body: %s\nerror: %v", string(respBody), err)
		}
		return &DeleteInstanceResult{Async: true, OperationID: rb.Operation}, nil
	case http.StatusBadRequest:
		return nil, validateGCPFailureResponse(respBody, respCode, "request was malformed or missing mandatory data")
	case http.StatusUnprocessableEntity:
		return nil, validateGCPFailureResponse(respBody, respCode, "the broker only supports asynchronous requests")
	default:
		return nil, validateGCPFailureResponse(respBody, respCode, "request was not successful")
	}
}

// UpdateInstance calls the given server to update a service instance using the request information.
func (adapter *httpAdapter) UpdateInstance(params *UpdateInstanceParams) (*UpdateInstanceResult, error) {
	instanceURL := fmt.Sprintf("%s/v2/service_instances/%s", params.Server, params.InstanceID)

	patchBody := &osb.UpdateInstanceRequestBody{
		ServiceID:  params.ServiceID,
		PlanID:     params.PlanID,
		Context:    params.Context,
		Parameters: params.Parameters,
		PreviousValues: &osb.UpdateInstancePreviousValues{
			ServiceID:      params.PreviousServiceID,
			PlanID:         params.PreviousPlanID,
			OrganizationID: params.PreviousOrganizationID,
			SpaceID:        params.PreviousSpaceID,
		},
	}

	patchParams := url.Values{}
	patchParams.Set(acceptsIncompleteKey, strconv.FormatBool(params.AcceptsIncomplete))

	respCode, respBody, err := adapter.doOSBRequest(instanceURL, http.MethodPatch, params.APIVersion, patchBody, patchParams)
	if err != nil {
		return nil, err
	}

	unmarshalSuccessResponse := func(body []byte, isAsync bool) (*UpdateInstanceResult, error) {
		rb := &osb.UpdateInstanceResponseBody{}
		err := json.Unmarshal(body, rb)
		if err != nil {
			return nil, fmt.Errorf("error unmarshalling response body: %s\nerror: %v", string(body), err)
		}
		return &UpdateInstanceResult{Async: isAsync, OperationID: rb.Operation}, nil
	}

	switch respCode {
	case http.StatusOK:
		// Requested changes have been applied.
		return unmarshalSuccessResponse(respBody, false)
	case http.StatusAccepted:
		// Service instance is being updated asynchronously.
		if !params.AcceptsIncomplete {
			return nil, fmt.Errorf("request shouldn't be handled asynchronously: %s", string(respBody))
		}

		return unmarshalSuccessResponse(respBody, true)
	case http.StatusBadRequest:
		return nil, validateGCPFailureResponse(respBody, respCode, "request was malformed or missing mandatory data")
	case http.StatusUnprocessableEntity:
		return nil, validateGCPFailureResponse(respBody, respCode, "the broker only supports asynchronous requests")
	default:
		return nil, validateGCPFailureResponse(respBody, respCode, "request was not successful")
	}
}

// InstanceLastOperation calls the given server to get the state of the last operation for the
// service instance.
func (adapter *httpAdapter) InstanceLastOperation(params *InstanceLastOperationParams) (*Operation, error) {
	operationURL := fmt.Sprintf("%s/v2/service_instances/%s/last_operation", params.Server, params.InstanceID)
	return adapter.lastOperation(instanceKey, operationURL, params.LastOperationParams)
}

// CreateBinding calls the given server to bind to a service instance using the request information.
func (adapter *httpAdapter) CreateBinding(params *CreateBindingParams) (*CreateBindingResult, error) {
	bindingURL := fmt.Sprintf("%s/v2/service_instances/%s/service_bindings/%s", params.Server, params.InstanceID, params.BindingID)

	putBody := &osb.BindRequestBody{
		ServiceID:    params.ServiceID,
		PlanID:       params.PlanID,
		Context:      params.Context,
		AppGUID:      params.AppGUID,
		BindResource: params.BindResource,
		Parameters:   params.Parameters,
	}

	putParams := url.Values{}
	putParams.Set(acceptsIncompleteKey, strconv.FormatBool(params.AcceptsIncomplete))

	respCode, respBody, err := adapter.doOSBRequest(bindingURL, http.MethodPut, params.APIVersion, putBody, putParams)
	if err != nil {
		return nil, err
	}

	marshalSuccessResponse := func(body []byte, isAsync bool) (*CreateBindingResult, error) {
		rb := &osb.BindResponseBody{}
		err := json.Unmarshal(body, rb)
		if err != nil {
			return nil, fmt.Errorf("error unmarshalling response body: %s\nerror: %v", string(body), err)
		}
		return &CreateBindingResult{
			Async:           isAsync,
			Credentials:     rb.Credentials,
			SyslogDrainURL:  rb.SyslogDrainURL,
			RouteServiceURL: rb.RouteServiceURL,
			VolumeMounts:    rb.VolumeMounts,
			OperationID:     rb.Operation,
		}, nil
	}

	switch respCode {
	case http.StatusOK, http.StatusCreated:
		// Identical service binding already exists, or service binding is created synchronously.
		return marshalSuccessResponse(respBody, false)
	case http.StatusAccepted:
		// Service binding is being created asynchronously.
		if !params.AcceptsIncomplete {
			return nil, fmt.Errorf("request shouldn't be handled asynchronously: %s", string(respBody))
		}
		return marshalSuccessResponse(respBody, true)
	case http.StatusBadRequest:
		return nil, validateGCPFailureResponse(respBody, respCode, "request was malformed or missing mandatory data")
	case http.StatusConflict:
		return nil, validateGCPFailureResponse(respBody, respCode, "binding with the same id but different attributes already exists")
	case http.StatusUnprocessableEntity:
		return nil, validateGCPFailureResponse(respBody, respCode, "the broker only supports asynchronous requests")
	default:
		return nil, validateGCPFailureResponse(respBody, respCode, "request was not successful")
	}
}

// DeleteBinding calls the given server to unbind to a service instance using the request information.
func (adapter *httpAdapter) DeleteBinding(params *DeleteBindingParams) (*DeleteBindingResult, error) {
	bindingURL := fmt.Sprintf("%s/v2/service_instances/%s/service_bindings/%s", params.Server, params.InstanceID, params.BindingID)

	deleteParams := url.Values{}
	deleteParams.Set(serviceIDKey, params.ServiceID)
	deleteParams.Set(planIDKey, params.PlanID)
	deleteParams.Set(acceptsIncompleteKey, strconv.FormatBool(params.AcceptsIncomplete))

	respCode, respBody, err := adapter.doOSBRequest(bindingURL, http.MethodDelete, params.APIVersion, nil, deleteParams)
	if err != nil {
		return nil, err
	}

	switch respCode {
	case http.StatusOK, http.StatusGone:
		// The service binding is deleted synchronously, or it doesn't exist.
		// The response should be an empty JSON object ("{}") with no data.
		return &DeleteBindingResult{Async: false}, nil
	case http.StatusAccepted:
		// The service binding is being deleted asynchronously.
		if !params.AcceptsIncomplete {
			return nil, fmt.Errorf("request shouldn't be handled asynchronously: %s", string(respBody))
		}

		rb := &osb.UnbindResponseBody{}
		err := json.Unmarshal(respBody, rb)
		if err != nil {
			return nil, fmt.Errorf("error unmarshalling response body: %s\nerror: %v", string(respBody), err)
		}
		return &DeleteBindingResult{Async: true, OperationID: rb.Operation}, nil
	case http.StatusBadRequest:
		return nil, validateGCPFailureResponse(respBody, respCode, "request was malformed or missing mandatory data")
	case http.StatusUnprocessableEntity:
		return nil, validateGCPFailureResponse(respBody, respCode, "the broker only supports asynchronous requests")
	default:
		return nil, validateGCPFailureResponse(respBody, respCode, "request was not successful")
	}
}

// BindingLastOperation calls the given server to get the state of the last operation for the
// service binding.
func (adapter *httpAdapter) BindingLastOperation(params *BindingLastOperationParams) (*Operation, error) {
	operationURL := fmt.Sprintf("%s/v2/service_instances/%s/service_bindings/%s/last_operation", params.Server, params.InstanceID, params.BindingID)
	return adapter.lastOperation(bindingKey, operationURL, params.LastOperationParams)
}

func (adapter *httpAdapter) lastOperation(resource, operationURL string, params *LastOperationParams) (*Operation, error) {
	getParams := url.Values{}
	if params.ServiceID != "" {
		getParams.Set(serviceIDKey, params.ServiceID)
	}
	if params.PlanID != "" {
		getParams.Set(planIDKey, params.PlanID)
	}
	if params.OperationID != "" {
		getParams.Set(operationKey, params.OperationID)
	}

	respCode, respBody, err := adapter.doOSBRequest(operationURL, http.MethodGet, params.APIVersion, nil, getParams)
	if err != nil {
		return nil, err
	}

	switch respCode {
	case http.StatusOK:
		rb := &osb.OperationResponseBody{}
		err = json.Unmarshal(respBody, rb)
		if err != nil {
			return nil, fmt.Errorf("error unmarshalling response body: %s\nerror: %v", string(respBody), err)
		}
		return &Operation{State: rb.State, Description: rb.Description}, nil
	case http.StatusBadRequest:
		return nil, validateGCPFailureResponse(respBody, respCode, "request was malformed or missing mandatory data")
	case http.StatusGone:
		if params.OperationType == OperationDelete {
			return &Operation{State: OperationSucceeded, Description: fmt.Sprintf("The %s doesn't exist.", resource)}, nil
		}

		return nil, validateGCPFailureResponse(respBody, respCode, fmt.Sprintf("%s doesn't exist", resource))
	default:
		return nil, validateGCPFailureResponse(respBody, respCode, "request was not successful")
	}
}

// doRequst is a helper function that performs the request, reads the body, checks the status code,
// and returns an appropriate error message if anything goes wrong.
// v should be a pointer to the struct that we want to unmarshal the body into.
func (adapter *httpAdapter) doRequest(url, method string, reqBody io.Reader, v interface{}) error {
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}
	resp, err := adapter.client.Do(req)
	if err != nil {
		return fmt.Errorf("error executing request: %v; error: %v", req, err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %v", err)
	}

	if resp.StatusCode >= http.StatusMultipleChoices || resp.StatusCode < http.StatusOK {
		return fmt.Errorf("request was not successful: %v", string(body))
	}

	if v != nil {
		err = json.Unmarshal(body, v)
		if err != nil {
			return fmt.Errorf("error unmarshalling response body into v\nBody: %s\nError: %v", string(body), err)
		}
	}
	return nil
}

// TODO(maqiuyu): Merge doRequest and doOSBRequest.
// doOSBRequst is a helper function that performs the OSB request, reads the response body, and
// returns the response status code and response body.
func (adapter *httpAdapter) doOSBRequest(url, method, apiVersion string, reqBody interface{}, reqParams url.Values) (int, []byte, error) {
	var streamedBody io.Reader
	if reqBody != nil {
		serializedReqBody, err := json.Marshal(reqBody)
		if err != nil {
			return 0, nil, fmt.Errorf("error marshalling the request body %+v: %v", reqBody, err)
		}

		streamedBody = bytes.NewReader(serializedReqBody)
	}

	req, err := http.NewRequest(method, url, streamedBody)
	if err != nil {
		return 0, nil, fmt.Errorf("error creating request: %v", err)
	}
	req.URL.RawQuery = reqParams.Encode()
	req.Header.Add(apiVersionHeader, apiVersion)

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

// TODO(maqiuyu): Add methods to handle non-GCP errors.
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
