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
//
// Package adapter contains the interface to create/delete/list resources as
// well as various implementations using different protocols.
package adapter

// Adapter is the interface to connect to service brokers and the admin service.
type Adapter interface {
	// Admin methods.
	CreateBroker(params *CreateBrokerParams) ([]byte, error)
	DeleteBroker(params *DeleteBrokerParams) ([]byte, error)
	GetBroker(params *GetBrokerParams) ([]byte, error)
	ListBrokers(params *ListBrokersParams) ([]byte, error)

	// OSB APIs.
	CreateInstance(params *CreateInstanceParams) (*CreateInstanceResult, error)
}

// CreateBrokerParams is used as input to CreateBroker.
type CreateBrokerParams struct {
	RegistryURL string
	Project     string
	Name        string
	Title       string
	Catalogs    []string
}

// DeleteBrokerParams is used as input to DeleteBroker.
type DeleteBrokerParams struct {
	RegistryURL string
	Project     string
	Name        string
}

// GetBrokerParams is used as input to GetBroker.
type GetBrokerParams struct {
	RegistryURL string
	Project     string
	Name        string
}

// ListBrokersParams is used as input to ListBrokers.
type ListBrokersParams struct {
	RegistryURL string
	Project     string
}

// CreateInstanceParams stores the parameters used to create an instance.
type CreateInstanceParams struct {
	// Server is the URL for the broker.
	Server string
	// AcceptsIncomplete indicates whether the client can accept asynchronous provisioning.  If the
	// broker does not support synchronous/asynchronous provisioning of a service, it will reject a
	// request with this field set to false/true.
	AcceptsIncomplete bool
	// InstanceID is the ID of the service instance.  The Open Service Broker API specification
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
	OperationID string
}
