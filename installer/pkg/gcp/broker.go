/*
Copyright 2017 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package gcp

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

const (
	DeploymentManagerAPI = "deploymentmanager.googleapis.com"
	ServiceRegistryAPI   = "serviceregistry.googleapis.com"
	ServiceBrokerAPI     = "servicebroker.googleapis.com"
)

// EnableAPIs enables given APIs in user's GCP project.
func EnableAPIs(apis []string) error {
	existingAPIs, err := enabledAPIs()
	if err != nil {
		return err
	}

	for _, api := range apis {
		if _, found := existingAPIs[api]; !found {
			err = enableAPI(api)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// enabledAPIs returned set of enabled GCP APIs.
func enabledAPIs() (map[string]bool, error) {
	cmd := exec.Command("gcloud", "service-management", "list", "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to retrived enabled GCP APIs : %v", err)
	}

	var apis []gcpAPI
	err = json.Unmarshal(output, &apis)
	if err != nil {
		return nil, fmt.Errorf("failed to parse enabled API response : %v", err)
	}

	m := make(map[string]bool, len(apis))
	for _, api := range apis {
		m[api.ServiceName] = true
	}

	return m, nil
}

type gcpAPI struct {
	ServiceName string `json:"serviceName"`
}

// enableAPI enables a GCP API.
func enableAPI(api string) error {
	cmd := exec.Command("gcloud", "service-management", "enable", api)
	_, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to enable API %s : %v", api, err)
	}

	return nil
}

func CreateServiceAccount(name, displayName string) error {
	cmd := exec.Command("gcloud", "beta", "iam", "service-accounts", "create",
		name,
		"--display-name", displayName,
		"--format", "json")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to create service account : %v %s", err, string(output))
	}

	return err
}

func GetServiceAccount(email string) (*ServiceAccount, error) {
	cmd := exec.Command("gcloud", "beta", "iam", "service-accounts", "describe", email, "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve service account : %v:%v", err, string(output))
	}

	var sa ServiceAccount
	err = json.Unmarshal(output, &sa)
	if err != nil {
		return nil, fmt.Errorf("failed to parse service account API response : %v", err)
	}

	return &sa, nil
}

func UpdateServiceAccountPerms(projectID, email, roles string) error {
	cmd := exec.Command("gcloud", "projects", "add-iam-policy-binding", projectID, "--member", "serviceAccount:"+email, "--role", roles, "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to update service account permissions: %v %s", string(output), err)
	}
	return nil
}

type ServiceAccount struct {
	Email       string `json:"email"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

func CreateServiceAccountKey(email, keyFilepath string) error {
	cmd := exec.Command("gcloud", "beta", "iam", "service-accounts", "keys", "create", "--iam-account", email, keyFilepath)
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to create service account key: %s : %v", string(out), err)
	}
	return nil
}

// GetConfigValue returns a property value from given section of gcloud's
// default config.
func GetConfigValue(section, property string) (string, error) {
	cmd := exec.Command("gcloud", "config", "get-value", section+"/"+property)
	value, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to retrieve config-value : %v", err)
	}
	return strings.Trim(string(value), "\n"), nil
}
