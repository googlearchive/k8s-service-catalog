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
//
// Package adapter contains the interface to create/delete/list resources as
// well as various implementations using different protocols.
package adapter

import (
	"fmt"
	"strconv"

	"github.com/GoogleCloudPlatform/k8s-service-catalog/broker-cli/client/osb"
)

// Adapter is the interface to connect to service brokers and the admin service.
type Adapter interface {
	// Admin methods.
	CreateBroker(params *CreateBrokerParams) (*osb.Broker, error)
	DeleteBroker(params *DeleteBrokerParams) error
	ListInstances(params *ListInstancesParams) (*ListInstancesResult, error)
	ListBindings(params *ListBindingsParams) (*ListBindingsResult, error)
	ListBrokers(params *ListBrokersParams) (*ListBrokersResult, error)

	// OSB APIs.
	GetCatalog(params *GetCatalogParams) (*GetCatalogResult, error)
	CreateInstance(params *CreateInstanceParams) (*CreateInstanceResult, error)
	DeleteInstance(params *DeleteInstanceParams) (*DeleteInstanceResult, error)
	UpdateInstance(params *UpdateInstanceParams) (*UpdateInstanceResult, error)
	InstanceLastOperation(params *InstanceLastOperationParams) (*Operation, error)
	CreateBinding(params *CreateBindingParams) (*CreateBindingResult, error)
	DeleteBinding(params *DeleteBindingParams) (*DeleteBindingResult, error)
	BindingLastOperation(params *BindingLastOperationParams) (*Operation, error)
}

// CreateBrokerParams is used as input to CreateBroker.
type CreateBrokerParams struct {
	Host    string
	Project string
	Name    string
	Title   string
}

// DeleteBrokerParams is used as input to DeleteBroker.
type DeleteBrokerParams struct {
	BrokerURL string
}

// ListBrokersParams is used as input to ListBrokers.
type ListBrokersParams struct {
	Host    string
	Project string
}

// ListBrokersResult is the output from ListBrokers.
type ListBrokersResult struct {
	Brokers []osb.Broker `json:"brokers"`
}

// OperationType is the enum to represent all the types of OSB operations.
type OperationType int

const (
	OperationCreate OperationType = iota
	OperationUpdate
	OperationDelete
	OperationUnknown
)

// A list of constants to represent all the states of OSB operations.
const (
	OperationSucceeded  = "succeeded"
	OperationFailed     = "failed"
	OperationInProgress = "in progress"
)

// GetCatalogParams is used as input to GetCatalog.
type GetCatalogParams struct {
	// Server is URL of the broker.
	Server string
	// APIVersion is the header value associated with the version of the Open Service Broker API used
	// by the request.
	APIVersion string
}

// GetCatalogResult is output of successful GetCatalog request.
type GetCatalogResult struct {
	Services []osb.Service
}

// CreateInstanceParams stores the parameters used to create an instance.
type CreateInstanceParams struct {
	// Server is the URL for the broker.
	Server string
	// APIVersion is the header value associated with the version of the Open Service Broker API used
	// by the request.
	APIVersion string
	// AcceptsIncomplete indicates whether the client can accept asynchronous provisioning.  If the
	// broker does not support synchronous/asynchronous provisioning of a service, it will reject a
	// request with this field set to false/true respectively.
	AcceptsIncomplete bool
	// InstanceID is the ID of the service instance. The Open Service Broker API specification
	// recommends using a GUID for this field.
	InstanceID string
	// ServiceID is the ID of the service to use for the service instance.
	ServiceID string
	// PlanID is the ID of the plan to use for the service instance.
	PlanID string
	// Context is platform-specific contextual information under which the service instance is to be
	// provisioned. Context was added in version 2.12 of the OSB API and is only sent for versions
	// 2.12 or later. Optional.
	Context map[string]interface{}
	// OrganizationGUID is the platform GUID for the organization under which the service plan is to
	// be provisioned. CF-specific. Optional.
	OrganizationGUID string
	// SpaceGUID is the identifier for the project space within the platform organization.
	// CF-specific. Optional.
	SpaceGUID string
	// Parameters is a set of configuration options for the service instance. Optional.
	Parameters map[string]interface{}
}

// CreateInstanceResult is the result of a successful instance creation request.
type CreateInstanceResult struct {
	// Async indicates whether the broker is handling the provision request asynchronously.
	Async bool
	// DashboardURL is the URL of a web-based management user interface for the service instance.
	DashboardURL string
	// OperationID is an extra identifier supplied by the broker to identify asynchronous operations.
	// It should be an empty string if the request is completed synchronously.
	OperationID string
}

// ListInstancesParams stores the parameters used to list instances in a broker.
type ListInstancesParams struct {
	// Server is the URL for the broker.
	Server string
}

// ListInstancesResult is the output from ListInstances.
type ListInstancesResult struct {
	Instances []*osb.Instance `json:"instances"`
}

// ListBindingsParams stores the parameters used to list bindings to an instance.
type ListBindingsParams struct {
	// Server is the URL for the broker.
	Server string
	// UUID representing the instance.
	InstanceID string
}

// ListBindingsResult is the output from ListBindings.
type ListBindingsResult struct {
	Bindings []*osb.Binding `json:"bindings"`
}

// DeleteInstanceParams stores the parameters used to delete an instance.
type DeleteInstanceParams struct {
	// Server is the URL for the broker.
	Server string
	// APIVersion is the header value associated with the version of the Open Service Broker API used
	// by the request.
	APIVersion string
	// AcceptsIncomplete indicates whether the client can accept asynchronous provisioning.  If the
	// broker does not support synchronous/asynchronous provisioning of a service, it will reject a
	// request with this field set to false/true respectively.
	AcceptsIncomplete bool
	// InstanceID is the ID of the service instance.  The Open Service Broker API specification
	// recommends using a GUID for this field.
	InstanceID string
	// ServiceID is the ID of the service to use for the service instance.
	ServiceID string
	// PlanID is the ID of the plan to use for the service instance.
	PlanID string
}

// DeleteInstanceResult is the result of a successful instance deletion request.
type DeleteInstanceResult struct {
	// Async indicates whether the broker is handling the deprovision request asynchronously.
	Async bool
	// OperationID is an extra identifier supplied by the broker to identify asynchronous operations.
	// It should be an empty string if the request is completed synchronously.
	OperationID string
}

// UpdateInstanceParams stores the parameters used to update an instance.
type UpdateInstanceParams struct {
	// Server is the URL for the broker.
	Server string
	// APIVersion is the header value associated with the version of the Open Service Broker API used
	// by the request.
	APIVersion string
	// AcceptsIncomplete indicates whether the client can accept asynchronous update. If the broker
	// does not support synchronous/asynchronous update of an instance, it will reject a request with
	// this field set to false/true respectively.
	AcceptsIncomplete bool
	// InstanceID is the ID of a previously provisioned service instance.
	InstanceID string
	// ServiceID is the ID of the service provisioned by the service instance.
	ServiceID string
	// PlanID is the ID of the plan to use for the service instance.
	PlanID string
	// Context is the contextual data under which the service instance is created.
	Context map[string]interface{}
	// Parameters is the configuration options for the service instance.
	Parameters map[string]interface{}
	// PreviousServiceID is the ID of the service provisioned by the service instance.
	PreviousServiceID string
	// PreviousPlanID is the ID of the plan prior to the update.
	PreviousPlanID string
	// PreviousOrganizationID is the ID of the organization specified for the service instance.
	PreviousOrganizationID string
	// PreviousSpaceID is the ID of the space specified for the service instance.
	PreviousSpaceID string
}

// UpdateInstanceResult is the result of a successful instance update request.
type UpdateInstanceResult struct {
	// Async indicates whether the broker is handling the update request asynchronously.
	Async bool
	// OperationID is an extra identifier supplied by the broker to identify asynchronous operations.
	// It should be an empty string if the request is completed synchronously.
	OperationID string
}

// CreateBindingParams stores the parameters used to create a binding.
type CreateBindingParams struct {
	// Server is the URL for the broker.
	Server string
	// APIVersion is the header value associated with the version of the Open Service Broker API used
	// by the request.
	APIVersion string
	// AcceptsIncomplete indicates whether the client can accept asynchronous provisioning.  If the
	// broker does not support synchronous/asynchronous provisioning of a service, it will reject a
	// request with this field set to false/true respectively.
	AcceptsIncomplete bool
	// InstanceID is the ID of the service instance to bind to.
	InstanceID string
	// BindingID is the ID of the service binding to be created. The Open Service Broker API
	// specification recommends using a GUID for this field.
	BindingID string
	// ServiceID is the ID of the service to use for the service binding.
	ServiceID string
	// PlanID is the ID of the plan to use for the service binding.
	PlanID string
	// Context is the contextual information under which the service binding is to be created.
	Context map[string]interface{}
	// AppGUID is the GUID of an application associated with the binding to be created. Optional.
	AppGUID string
	// BindResource holds extra information about platform resources associated with the binding to
	// be created. CF-specific. Optional.
	BindResource map[string]interface{}
	// Parameters is a set of configuration options for the service binding. Optional.
	Parameters map[string]interface{}
}

// CreateBindingResult is the result of a successful binding creation request.
type CreateBindingResult struct {
	// Async indicates whether the broker is handling the bind request asynchronously.
	Async bool
	// Credentials is a free-form hash of credentials that can be used by applications or users to
	// access the service.
	Credentials map[string]interface{}
	// SyslogDrainURl is a URL to which logs must be streamed. CF-specific. May only be supplied by a
	// service that declares a requirement for the 'syslog_drain' permission.
	SyslogDrainURL *string
	// RouteServiceURL is a URL to which the platform must proxy requests to the application the
	// binding is for. CF-specific. May only be supplied by a service that declares a requirement for
	// the 'route_service' permission.
	RouteServiceURL *string
	// VolumeMounts is an array of configuration string for mounting volumes. CF-specific. May only be
	// supplied by a service that declares a requirement for the 'volume_mount' permission.
	VolumeMounts []interface{}
	// OperationID is an extra identifier supplied by the broker to identify asynchronous operations.
	// It should be an empty string if the request is completed synchronously.
	OperationID string
}

// DeleteBindingParams stores the parameters used to delete a binding.
type DeleteBindingParams struct {
	// Server is the URL for the broker.
	Server string
	// APIVersion is the header value associated with the version of the Open Service Broker API used
	// by the request.
	APIVersion string
	// AcceptsIncomplete indicates whether the client can accept asynchronous provisioning.  If the
	// broker does not support synchronous/asynchronous provisioning of a service, it will reject a
	// request with this field set to false/true respectively.
	AcceptsIncomplete bool
	// InstanceID is the ID of the service instance to bind to.
	InstanceID string
	// BindingID is the ID of the service binding to be created. The Open Service Broker API
	// specification recommends using a GUID for this field.
	BindingID string
	// ServiceID is the ID of the service to use for the service binding.
	ServiceID string
	// PlanID is the ID of the plan to use for the service binding.
	PlanID string
}

// DeleteBindingResult is the result of a successful binding deletion request.
type DeleteBindingResult struct {
	// Async indicates whether the broker is handling the bind request asynchronously.
	Async bool
	// OperationID is an extra identifier supplied by the broker to identify asynchronous operations.
	// It should be an empty string if the request is completed synchronously.
	OperationID string
}

// LastOperationParams contains the common params used to poll last operation of the resource.
type LastOperationParams struct {
	// APIVersion is the header value associated with the version of the Open Service Broker API used
	// by the request.
	APIVersion string
	// ServiceID is the ID of the service that the resource is created from. Optional, but recommended.
	ServiceID string
	// PlanID is the ID of the plan that the resource is created from. Optional, but recommended.
	PlanID string
	// OperationID is the operation ID provided by the broker in the response to the initial request.
	// Optional, but must be sent if supplied in the response to the original request.
	OperationID string
	// OperationType is one of Create, Update or Delete.
	OperationType OperationType
}

// InstanceLastOperation stores the parameters used to poll the last operation of a service instance.
type InstanceLastOperationParams struct {
	// Server is the URL for the broker.
	Server string
	// InstanceID is the instance of the service to query the last operation for.
	InstanceID string
	// LastOperationParams contains the common params used to poll last operation of the resource.
	*LastOperationParams
}

// BindingLastOperation stores the parameters used to poll the last operation of a service binding.
type BindingLastOperationParams struct {
	// Server is the URL for the broker.
	Server string
	// InstanceID is the instance that the binding binds to.
	InstanceID string
	// BindingID is the service binding to query the last operation for.
	BindingID string
	// LastOperationParams contains the common params used to poll last operation of the resource.
	*LastOperationParams
}

// Operation is the result of a successful operation polling request.
type Operation struct {
	// State is the state of the queried operation.
	State string
	// Description is a message from the broker describing the current state of the operation.
	Description string
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
