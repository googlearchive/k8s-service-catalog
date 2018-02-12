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
	"time"
)

const (
	DeploymentManagerAPI = "deploymentmanager.googleapis.com"
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
		return nil, fmt.Errorf("failed to retrieve enabled GCP APIs : %v", err)
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
	_, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to enable API %s : %v", api, err)
	}

	return nil
}

func CreateServiceAccount(name, displayName string) error {
	cmd := exec.Command("gcloud", "iam", "service-accounts", "create",
		name,
		"--display-name", displayName,
		"--format", "json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create service account : %v %s", err, string(output))
	}

	return err
}

func GetServiceAccount(email string) (*ServiceAccount, error) {
	cmd := exec.Command("gcloud", "iam", "service-accounts", "describe", email, "--format", "json")
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

func AddServiceAccountPerms(projectID, email, roles string) error {
	cmd := exec.Command("gcloud", "projects", "add-iam-policy-binding", projectID, "--member", "serviceAccount:"+email, "--role", roles, "--format", "json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to add service account permissions: %v %s", string(output), err)
	}
	return nil
}

func RemoveServiceAccountPerms(projectID, email, roles string) error {
	cmd := exec.Command("gcloud", "projects", "remove-iam-policy-binding", projectID, "--member", "serviceAccount:"+email, "--role", roles, "--format", "json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to remove service account permissions: %v %s", string(output), err)
	}
	return nil
}

type ServiceAccount struct {
	Email       string `json:"email"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

func CreateServiceAccountKey(email, keyFilepath string) error {
	cmd := exec.Command("gcloud", "iam", "service-accounts", "keys", "create", "--iam-account", email, keyFilepath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create service account key: %s : %v", string(out), err)
	}
	return nil
}

type saKey struct {
	Algorithm       string `json:"keyAlgorithm"`
	Name            string `json:"name"`
	ValidAfterTime  string `json:"validAfterTime"`
	ValidBeforeTime string `json:"validBeforeTime"`
}

func RemoveServiceAccountKeys(email string) error {
	cmd := exec.Command("gcloud", "iam", "service-accounts", "keys", "list", "--iam-account", email, "--format=json")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to list service account keys: %s : %v", string(out), err)
	}

	var keys []saKey
	err = json.Unmarshal(out, &keys)
	if err != nil {
		return fmt.Errorf("failed to unmarshal service account keys: %s : %v", string(out), err)
	}

	for _, k := range keys {
		// Check the life ("ValidBeforeTime" - "ValidAfterTime") of it because we only need to delete the keys generated
		// by the user. Those keys should be living for 3650 days. Here we check whether the life is more than a year.
		// The service accounts also have some robot keys, but those keys are only alive for a couple of days.
		bt, err := time.Parse(time.RFC3339, k.ValidAfterTime)
		if err != nil {
			return fmt.Errorf("failed to parse the timestamp of the service account key (%+v): %v", k, err)
		}

		et, err := time.Parse(time.RFC3339, k.ValidBeforeTime)
		if err != nil {
			return fmt.Errorf("failed to parse the timestamp of the service account key (%+v): %v", k, err)
		}

		life := et.Sub(bt)
		if life > 365*24*time.Hour {
			cmd := exec.Command("gcloud", "iam", "service-accounts", "keys", "delete", k.Name, "--iam-account", email, "--quiet" /*disable interactive mode*/)
			out, err := cmd.CombinedOutput()
			if err != nil {
				fmt.Printf("failed to delete service account key: %s : %v\n", string(out), err)
				fmt.Printf("WARNING: Please clean up the key from service account %s", email)
			}
		}
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

// GetConfigMap returns all the gcloud config in a JSON struct.
func GetConfigMap() (map[string]interface{}, error) {
	cmd := exec.Command("gcloud", "config", "list", "--format=json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list gcloud config : %v", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(output, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal gcloud config: %s : %v", string(output), err)
	}

	return result, nil
}
