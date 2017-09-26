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

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/GoogleCloudPlatform/k8s-service-catalog/installer/pkg/gcp"
)

var (
	gcpBrokerFileNames = []string{"gcp-broker", "service-account-secret"}
)

func addGCPBroker() error {
	projectID, err := gcp.GetConfigValue("core", "project")
	if err != nil {
		fmt.Errorf("error getting configured project value : %v", err)
	}

	fmt.Println("using project: ", projectID)

	requiredAPIs := []string{
		gcp.DeploymentManagerAPI,
		gcp.ServiceBrokerAPI,
		gcp.ServiceRegistryAPI,
	}
	err = gcp.EnableAPIs(requiredAPIs)

	fmt.Println("enabled required APIs ", requiredAPIs)

	brokerSAName := "service-catalog-gcp"
	brokerSAEmail := fmt.Sprintf("%s@%s.iam.gserviceaccount.com", brokerSAName, projectID)

	_, err = gcp.GetServiceAccount(brokerSAEmail)
	if err != nil {
		fmt.Printf("error fetching service account :%v", err)
		// TODO(droot): distinguish between real error and NOT_FOUND error
		err = gcp.CreateServiceAccount(brokerSAName, "Service Catalog GCP Broker Service Account")
		if err != nil {
			return err
		}
	}

	err = gcp.UpdateServiceAccountPerms(projectID, brokerSAEmail, "roles/servicebroker.operator")
	if err != nil {
		return err
	}

	// create temporary directory for k8s artifacts and other temporary files
	dir, err := ioutil.TempDir("/tmp", "service-catalog-gcp")
	if err != nil {
		return fmt.Errorf("error creating temporary dir: %v", err)
	}

	keyFile := filepath.Join(dir, "key.json")
	err = gcp.CreateServiceAccountKey(brokerSAEmail, keyFile)
	if err != nil {
		return fmt.Errorf("error creating service account key :%v", err)
	}
	fmt.Println("generated the key at :", keyFile)

	key, err := base64FileContent(keyFile)
	if err != nil {
		return fmt.Errorf("error reading content of the key file : %v", err)
	}

	data := map[string]string{
		"SvcAccountKey": key,
		// TODO(droot): replace the URL below with virtual broker URL for this project
		"GCPBrokerURL": "https://staging-servicebroker.sandbox.googleapis.com/v1alpha1/projects/seans-walkthrough/brokers/gcp-broker",
	}

	err = generateGCPBrokerConfigs(dir, data)
	if err != nil {
		return fmt.Errorf("error generating configs for GCP :: %v", err)
	}

	err = deployGCPBrokerConfigs(dir)
	if err != nil {
		return fmt.Errorf("error deploying GCP broker configs :%v", err)
	}

	return err
}

func generateGCPBrokerConfigs(dir string, data map[string]string) error {
	for _, f := range gcpBrokerFileNames {
		err := generateFileFromTmpl(filepath.Join(dir, f+".yaml"), "templates/gcp/"+f+".yaml.tmpl", data)
		if err != nil {
			return fmt.Errorf("error generating config file: %s :%v", f, err)
		}
	}
	return nil
}

func deployGCPBrokerConfigs(dir string) error {
	for _, f := range gcpBrokerFileNames {
		output, err := exec.Command("kubectl", "create", "-f", filepath.Join(dir, f+".yaml")).CombinedOutput()
		// TODO(droot): cleanup
		if err != nil {
			return fmt.Errorf("deploy failed with output: %s :%v", err, string(output))
		}
	}
	return nil
}

func removeGCPBroker() error {
	// create temporary directory for k8s artifacts and other temporary files
	dir, err := ioutil.TempDir("/tmp", "service-catalog-gcp")
	if err != nil {
		return fmt.Errorf("error creating temporary dir: %v", err)
	}

	defer os.RemoveAll(dir)

	err = generateGCPBrokerConfigs(dir, nil)
	if err != nil {
		return fmt.Errorf("error generating configs for GCP :: %v", err)
	}

	err = removeGCPBrokerConfigs(dir)
	if err != nil {
		return fmt.Errorf("error deleting broker resources : %v", err)
	}

	return nil
}

func removeGCPBrokerConfigs(dir string) error {
	for _, f := range gcpBrokerFileNames {
		output, err := exec.Command("kubectl", "delete", "-f", filepath.Join(dir, f+".yaml")).CombinedOutput()
		// TODO(droot): cleanup
		if err != nil {
			return fmt.Errorf("failed to delete broker resources output: %s :%v", err, string(output))
		}
	}
	return nil
}
