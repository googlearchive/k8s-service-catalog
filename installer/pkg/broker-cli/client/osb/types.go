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
// Pakcage osb contains the type definitions of OSB abstractions.
package osb

type Broker struct {
	Name     string   `json:"name"`
	Title    string   `json:"title,omitempty"`
	Catalogs []string `json:"catalogs"`
	URL      *string  `json:"url,omitempty"`
}

// ProvisionRequestBody contains the serialized request body to provision a service instance.
type ProvisionRequestBody struct {
	// ServiceID is the ID of the service to use for the service instance.
	ServiceID string `json:"service_id"`
	// PlanID is the ID of the plan to use for the service instance.
	PlanID string `json:"plan_id"`
	// Context is platform-specific contextual information under which the service instance is to be
	// provisioned. Context was added in version 2.12 of the OSB API and is only sent for versions
	// 2.12 or later. Optional.
	Context map[string]interface{} `json:"context,omitempty"`
	// OrganizationGUID is the platform GUID for the organization under which the service plan is to
	// be provisioned. CF-specific. Optional.
	OrganizationGUID string `json:"organization_guid,omitempty"`
	// SpaceGUID is the identifier for the project space within the platform organization.
	// CF-specific. Optional.
	SpaceGUID string `json:"space_guid,omitempty"`
	// Parameters is a set of configuration options for the service instance. Optional.
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// ProvisionResponseBody is the response body for a successful provision request.
type ProvisionResponseBody struct {
	// DashboardURL is the URL of a web-based management user interface for the service instance.
	DashboardURL string `json:"dashboard_url,omitempty"`
	// Operation is an extra identifier supplied by the broker to identify asynchronous operations.
	Operation string `json:"operation,omitempty"`
}
