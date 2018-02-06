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

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/GoogleCloudPlatform/k8s-service-catalog/installer/pkg/broker-cli/auth"
	"github.com/GoogleCloudPlatform/k8s-service-catalog/installer/pkg/broker-cli/client/adapter"
	"github.com/GoogleCloudPlatform/k8s-service-catalog/installer/pkg/gcp"
	"github.com/spf13/cobra"
)

var (
	gcpBrokerFileNames = []string{"gcp-broker", "google-oauth-deployment", "service-account-secret"}
)

func NewAddGCPBrokerCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add-gcp-broker",
		Short: "Adds GCP broker",
		Long:  `Adds a GCP broker to Service Catalog`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := addGCPBroker(); err != nil {
				fmt.Println("failed to configure GCP broker")
				return err
			}
			fmt.Println("GCP broker added successfully.")
			return nil
		},
	}
}

func addGCPBroker() error {
	projectID, err := gcp.GetConfigValue("core", "project")
	if err != nil {
		fmt.Errorf("error getting configured project value : %v", err)
	}

	fmt.Println("using project: ", projectID)

	requiredAPIs := []string{
		gcp.DeploymentManagerAPI,
		gcp.ServiceBrokerAPI,
	}
	err = gcp.EnableAPIs(requiredAPIs)

	fmt.Println("enabled required APIs ", requiredAPIs)

	brokerSAName := "service-catalog-gcp"
	brokerSAEmail := fmt.Sprintf("%s@%s.iam.gserviceaccount.com", brokerSAName, projectID)

	err = getOrCreateGCPServiceAccount(brokerSAName, brokerSAEmail)
	if err != nil {
		return err
	}
	// brokerSARole := "roles/editor"
	brokerSARole := "roles/servicebroker.operator"
	err = gcp.UpdateServiceAccountPerms(projectID, brokerSAEmail, brokerSARole)
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

	vb, err := getOrCreateVirtualBroker(projectID, "default", "Default Broker")
	if err != nil {
		return fmt.Errorf("error retrieving or creating default broker : %v", err)
	}

	data := map[string]interface{}{
		"SvcAccountKey": key,
		"GCPBrokerURL":  vb.URL,
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

func getOrCreateGCPServiceAccount(name, email string) error {
	_, err := gcp.GetServiceAccount(email)
	if err != nil {
		fmt.Printf("error fetching service account :%v", err)
		// TODO(droot): distinguish between real error and NOT_FOUND error
		err = gcp.CreateServiceAccount(name, "Service Catalog GCP Broker Service Account")
		if err != nil {
			return err
		}
	}
	return nil
}

func getOrCreateVirtualBroker(projectID, brokerName, brokerTitle string) (*virtualBroker, error) {
	// use the application default credentials
	brokerClient, err := httpAdapterFromAuthKey("")
	if err != nil {
		return nil, fmt.Errorf("failed to create broker client. You might want to run 'gcloud auth application-default login'")
	}

	brokerURL := "https://servicebroker.googleapis.com"
	errCode, respBody, err := brokerClient.CreateBroker(&adapter.CreateBrokerParams{
		URL:      brokerURL,
		Project:  projectID,
		Name:     brokerName,
		Title:    brokerTitle,
		Catalogs: []string{"projects/gcp-services/catalogs/gcp-catalog"},
	})
	if errCode == 409 {
		return &virtualBroker{
			Name:     brokerName,
			URL:      fmt.Sprintf("%s/v1beta1/projects/%s/brokers/%s", brokerURL, projectID, brokerName),
			Title:    brokerTitle,
			Catalogs: []string{"projects/gcp-services/catalogs/gcp-catalog"},
		}, nil
	}

	if err != nil {
		return nil, err
	}

	var vb virtualBroker
	err = json.Unmarshal(respBody, &vb)
	return &vb, err
}

// virtualBroker represents a GCP virtual broker.
type virtualBroker struct {
	Name     string   `json:"name"`
	Title    string   `json:"title"`
	Catalogs []string `json:"catalogs"`
	URL      string   `json:"url"`
}

func generateGCPBrokerConfigs(dir string, data map[string]interface{}) error {
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

// getContext returns a context using information from flags.
func getContext() context.Context {
	// TODO(richardfung): add flags so users can control this?
	return context.Background()
}

// httpAdapterFromAuthKey returns an http adapter with credentials to gcloud if
// keyFile is not set and to a service account if it is set.
func httpAdapterFromAuthKey(keyFile string) (*adapter.HttpAdapter, error) {
	var client *http.Client
	var err error
	if keyFile != "" {
		client, err = auth.HttpClientFromFile(getContext(), keyFile)
		if err != nil {
			return nil, fmt.Errorf("error creating http client from service account file %s: %v", keyFile, err)
		}
	} else {
		client, err = auth.HttpClientWithDefaultCredentials(getContext())
		if err != nil {
			return nil, fmt.Errorf("Error creating http client using default gcloud credentials: %v", err)
		}
	}
	return adapter.NewHttpAdapter(client), nil
}

func NewRemoveGCPBrokerCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove-gcp-broker",
		Short: "Remove GCP broker",
		Long:  `Removes a GCP broker from service catalog`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := removeGCPBroker(); err != nil {
				fmt.Println("failed to remove GCP broker")
				return err
			}
			fmt.Println("GCP broker removed successfully.")
			return nil
		},
	}
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
