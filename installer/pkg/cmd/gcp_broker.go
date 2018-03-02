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
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base32"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/k8s-service-catalog/installer/pkg/broker-cli/auth"
	"github.com/GoogleCloudPlatform/k8s-service-catalog/installer/pkg/broker-cli/client/adapter"
	"github.com/GoogleCloudPlatform/k8s-service-catalog/installer/pkg/gcp"
	"github.com/spf13/cobra"
)

const (
	oldBrokerSAName    = "service-catalog-gcp"
	brokerSANamePrefix = "scg-"
	brokerSARole       = "roles/servicebroker.operator"
)

var (
	gcpBrokerFileNames = []string{"gcp-broker", "google-oauth-deployment", "service-account-secret"}

	requiredAPIs = []string{
		"deploymentmanager.googleapis.com",
		"servicebroker.googleapis.com",
		// In the future, the APIs below will be enabled on-demand.
		"iam.googleapis.com",
		"bigtableadmin.googleapis.com",
		"cloudkms.googleapis.com",
		"cloudresourcemanager.googleapis.com",
		"ml.googleapis.com",
		"spanner.googleapis.com",
		"sqladmin.googleapis.com",
	}
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
		return fmt.Errorf("error getting configured project value : %v", err)
	}

	fmt.Println("using project: ", projectID)

	err = gcp.EnableAPIs(requiredAPIs)
	if err != nil {
		var b bytes.Buffer
		fmt.Fprintln(&b, "error enabling APIs. To make sure all APIs are correctly enabled, use links below:")
		for _, a := range requiredAPIs {
			fmt.Fprintf(&b, "   %s: https://console.cloud.google.com/apis/library/%s/?project=%s\n", a, a, projectID)
		}
		return errors.New(b.String())
	}

	fmt.Println("enabled required APIs:")
	for _, a := range requiredAPIs {
		fmt.Printf("  %s\n", a)
	}

	brokerSAName, err := constructSAName()
	if err != nil {
		return fmt.Errorf("error constructing service account name: %v", err)
	}

	brokerSAEmail := fmt.Sprintf("%s@%s.iam.gserviceaccount.com", brokerSAName, projectID)
	err = getOrCreateGCPServiceAccount(brokerSAName, brokerSAEmail)
	if err != nil {
		return err
	}

	err = gcp.AddServiceAccountPerms(projectID, brokerSAEmail, brokerSARole)
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
		// Clean up the newly generated key if the command failed.
		cleanupNewKey(brokerSAEmail, key)
		return fmt.Errorf("error retrieving or creating default broker : %v", err)
	}

	data := map[string]interface{}{
		"SvcAccountKey": key,
		"GCPBrokerURL":  vb.URL,
	}

	err = generateGCPBrokerConfigs(dir, data)
	if err != nil {
		// Clean up the newly generated key if the command failed.
		cleanupNewKey(brokerSAEmail, key)
		return fmt.Errorf("error generating configs for GCP :: %v", err)
	}

	err = deployGCPBrokerConfigs(dir)
	if err != nil {
		// Clean up the newly generated key if the command failed.
		cleanupNewKey(brokerSAEmail, key)
		return fmt.Errorf("error deploying GCP broker configs :%v", err)
	}

	return err
}

func constructSAName() (string, error) {
	bout, err := exec.Command("kubectl", "config", "view", "--output", "json").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error retriving kubernetes config: %s : %v", string(bout), err)
	}

	jout := make(map[string]interface{})
	err = json.Unmarshal(bout, &jout)
	if err != nil {
		return "", fmt.Errorf("error unmarshalling kubernetes config: %s : %v", string(bout), err)
	}

	// Get the name of the current cluster.
	fcn := jout["current-context"].(string)

	// Since there are limitations on the GCP service account name, we need to process the cluster name
	// before constructing the SA name with it.
	//
	// Hash the cluster name using MD5 algorithm.
	// This is because SA name only allows a maximum of 30 characters, so we need to reduce the length
	// of the cluster name.
	md5res := md5.Sum([]byte(fcn))
	var md5bytes []byte = md5res[:]

	// Use base32 to encode the MD5 hash result.
	// The result of MD5 hash will be a 32 digit hexadecimal number, so we need to further reduce the
	// length.
	res := base32.StdEncoding.EncodeToString(md5bytes)

	// Remove the last six "="s.
	// The raw result of MD5 hash is 16 bytes, so base32 encoding result will alway have a padding
	// "======".
	// This step can be replaced by base32.StdEncoding.WithPadding(base32.NoPadding) in Golang 1.9.
	res = strings.Trim(res, "=")

	// Add the prefix.
	res = fmt.Sprintf("%s%s", brokerSANamePrefix, res)

	// Convert the result to lowercase.
	// The result of base32 encoding contains uppercase letters but service account only allows
	// lowercase letters in the name.
	return strings.ToLower(res), nil
}

func getOrCreateGCPServiceAccount(name, email string) error {
	_, err := gcp.GetServiceAccount(email)
	if err != nil {
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
		output, err := exec.Command("kubectl", "apply", "-f", filepath.Join(dir, f+".yaml")).CombinedOutput()
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

// cleanupNewKey removes the newly generated service account key.
func cleanupNewKey(email, key string) {
	keyBytes, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		// Silently return if there is an error decoding the newly generated key.
		return
	}

	keyJson := make(map[string]interface{})
	err = json.Unmarshal(keyBytes, &keyJson)
	if err != nil {
		// Silently return if there is an error unmarshalling the key.
		return
	}

	keyID := keyJson["private_key_id"].(string)
	gcp.RemoveServiceAccountKey(email, keyID)
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
	// Create temporary directory for k8s artifacts and other temporary files.
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

	projectID, err := gcp.GetConfigValue("core", "project")
	if err != nil {
		return fmt.Errorf("error getting configured project value : %v", err)
	}

	brokerSAName, err := constructSAName()
	if err != nil {
		return fmt.Errorf("error constructing service account name: %v", err)
	}

	brokerSAEmail := fmt.Sprintf("%s@%s.iam.gserviceaccount.com", brokerSAName, projectID)
	_, err = gcp.GetServiceAccount(brokerSAEmail)
	if err != nil {
		// TODO(maqiuyujoyce): distinguish between real error and NOT_FOUND error
		oldBrokerSAEmail := fmt.Sprintf("%s@%s.iam.gserviceaccount.com", oldBrokerSAName, projectID)
		_, err := gcp.GetServiceAccount(oldBrokerSAEmail)
		if err == nil {
			fmt.Printf("WARNING: Service account %s is deprecated now. Please clean it up from your GCP project.\n", oldBrokerSAEmail)
		}

		// If we can't retrieve any of the service accounts, then it means either there is
		// something wrong with IAM server, or both accounts are invalid/nonexistent. And
		// we should safely assume the remove process is done.
		return nil
	}

	// Remove the Service Broker Operator role.
	err = gcp.RemoveServiceAccountPerms(projectID, brokerSAEmail, brokerSARole)
	if err != nil {
		return err
	}

	// Clean up all the associated keys.
	err = gcp.RemoveAllServiceAccountKeys(brokerSAEmail)
	if err != nil {
		return err
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
