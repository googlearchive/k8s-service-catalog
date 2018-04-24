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
	oldBrokerSAName                = "service-catalog-gcp"
	brokerSANamePrefix             = "scg-"
	brokerSARole                   = "roles/servicebroker.operator"
	gcpBrokerTemplateDir           = "templates/gcp/"
	gcpBrokerDeprecatedTemplateDir = "templates/gcp-deprecated/"
)

var (
	gcpBrokerDeprecatedFileNames = []string{"google-oauth-deployment", "service-account-secret"}
	gcpBrokerFileNames           = []string{"namespace", "gcp-broker", "google-oauth-deployment", "service-account-secret", "google-oauth-rbac", "google-oauth-service-account"}

	requiredAPIs = []string{
		"servicebroker.googleapis.com",
		// In the future, the APIs below will be enabled on-demand.
		"bigquery-json.googleapis.com",
		"bigtableadmin.googleapis.com",
		"ml.googleapis.com",
		"pubsub.googleapis.com",
		"spanner.googleapis.com",
		"sqladmin.googleapis.com",
		"storage-api.googleapis.com",
	}

	eapServiceGUIDS = map[string]bool{
		"e8c2ab3e-b96d-4140-8ff3-da682088737c": true,
		"33b61a24-c3ec-4721-a9e6-35648d7c441a": true,
		"31e5a956-cf67-4548-b266-20414d0ecfbb": true,
		"d730b49e-f865-40c4-8d22-1c62f2d37a83": true,
		"ab5b24a3-2962-42e8-9746-95242b3fe732": true,
		"483e1331-919f-403f-8618-dcea51ca88f7": true,
		"997146f4-bb5b-49c9-9ba6-f38931895825": true,
		"cbf73cfc-473a-4d42-874f-9ec9bd349963": true,
		"70eb86dc-7d68-4b72-8e3a-a42b40219b6d": true,
		"7cdbd1ef-f691-411e-8a91-2d8968387a33": true,
		"62153072-d076-4167-8014-596d1d2f7d28": true,
		"6c5b9ef4-ec57-4a39-ae4b-0b91f432b9c2": true,
		"42e243aa-b16a-4162-b8e5-3de2b461b119": true,
		"3007c0d2-91a7-4678-80f5-89877a9e08ad": true,
		"cac55a0b-9359-4ba0-accf-dac9b9c2c8bb": true,
		"d1d8c675-94d4-470d-9a71-a63d9e3c3915": true,
		"312e1044-ac87-4568-8155-9ad7765594fe": true,
		"61f859d6-86d4-4d62-a323-1a94c4d15bf3": true,
		"94e9d121-4936-4cb0-9928-b3540f92363c": true,
		"e06f6df6-2889-441e-a493-6184bf80b82d": true,
		"8ed5ce7b-849f-4c28-bc0a-383b336db407": true,
		"d5e45fc8-345b-490d-9f1e-3765a22d9b77": true,
		"6bd33df9-d45e-40be-86ce-cb2ff08875db": true,
		"2df32c48-0e0d-4080-9848-406ab20df26d": true,
		"3a25027c-0a83-48db-a31b-808632541b30": true,
		"db341b05-ae8a-4d0e-bb50-246b9714b263": true,
		"59d23ff0-f756-44e8-ba2b-24c6d846dc94": true,
		"44d078ba-8789-4222-bb11-0b8944fc5309": true,
		"ada963b1-7b7c-4f10-8bc5-82b163c5c25a": true,
	}
)

func NewAddGCPBrokerCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add-gcp-broker",
		Short: "Adds the Service Broker",
		Long:  `Adds Google Cloud Platfrom Service Broker to Service Catalog`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := addGCPBroker(); err != nil {
				fmt.Println("Failed to configure the Service Broker")
				return err
			}
			fmt.Println("The Service Broker has been added successfully.")
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

	if err := enableRequiredAPIs(projectID); err != nil {
		return err
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
		return fmt.Errorf("error creating service account key: %v", err)
	}
	fmt.Println("generated the key at: ", keyFile)

	key, err := base64FileContent(keyFile)
	if err != nil {
		return fmt.Errorf("error reading content of the key file : %v", err)
	}

	vb, err := getOrCreateVirtualBroker(projectID, "default", "Default Broker")
	if err != nil {
		// Clean up the newly generated key if the command failed.
		cleanupNewKey(brokerSAEmail, key)
		return fmt.Errorf("error retrieving or creating default broker: %v", err)
	}

	data := map[string]interface{}{
		"SvcAccountKey": key,
		"GCPBrokerURL":  vb.URL,
	}

	// generate config files and deploy the GCP broker resources
	err = generateConfigs(dir, gcpBrokerTemplateDir, gcpBrokerFileNames, data)
	if err != nil {
		// Clean up the newly generated key if the command failed.
		cleanupNewKey(brokerSAEmail, key)
		return fmt.Errorf("error generating configs for the Service Broker: %v", err)
	}

	err = deployConfigs(dir, gcpBrokerFileNames)
	if err != nil {
		// Clean up the newly generated key if the command failed.
		cleanupNewKey(brokerSAEmail, key)
		return fmt.Errorf("error deploying the Service Broker configs: %v", err)
	}

	return err
}

func enableRequiredAPIs(projectID string) error {
	if err := gcp.EnableAPIs(requiredAPIs); err != nil {
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

	return nil
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
		err = gcp.CreateServiceAccount(name, "Google Cloud Platform Service Broker")
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

	host := "https://servicebroker.googleapis.com"
	errCode, respBody, err := brokerClient.CreateBroker(&adapter.CreateBrokerParams{
		URL:     host,
		Project: projectID,
		Name:    brokerName,
		Title:   brokerTitle,
	})
	if errCode == http.StatusConflict {
		return handleExistingBroker(host, projectID, brokerName, brokerTitle, brokerClient)
	}

	if err != nil {
		return nil, err
	}

	var vb virtualBroker
	err = json.Unmarshal(respBody, &vb)
	return &vb, err
}

func handleExistingBroker(host, projectID, brokerName, brokerTitle string, brokerClient *adapter.HttpAdapter) (*virtualBroker, error) {
	fmt.Printf("Broker %q, already exists\n", brokerName)
	brokerURL := fmt.Sprintf("%s/v1beta1/projects/%s/brokers/%s", host, projectID, brokerName)
	res, err := brokerClient.GetCatalog(&adapter.GetCatalogParams{
		URL:     host,
		Project: projectID,
		Name:    brokerName,
	})
	if err != nil {
		return nil, fmt.Errorf("Invalid broker %q, error getting catalog for broker: %v\n", brokerURL, err)
	}

	isEAPBroker := false
	for _, svc := range res.Services {
		if _, ok := eapServiceGUIDS[svc.ID]; ok {
			isEAPBroker = true
			break
		}
	}

	if isEAPBroker {
		return nil, fmt.Errorf("Your existing broker is an early version of the broker. Please delete your broker using broker-cli and re-run \"sc add-gcp-broker\".\n")
	}

	return &virtualBroker{
		Name:     brokerName,
		URL:      brokerURL,
		Title:    brokerTitle,
		Existing: true,
	}, nil
}

// virtualBroker represents a GCP virtual broker.
type virtualBroker struct {
	Name     string `json:"name"`
	Title    string `json:"title"`
	URL      string `json:"url"`
	Existing bool   `json:"-"`
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
		Short: "Remove the Service Broker",
		Long:  `Removes Google Cloud Platform Service Broker from service catalog`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := removeGCPBroker(); err != nil {
				fmt.Println("Failed to remove the Service Broker")
				return err
			}
			fmt.Println("The Service Broker removed successfully.")
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

	// remove GCP Broker k8s resources
	err = generateConfigs(dir, gcpBrokerTemplateDir, gcpBrokerFileNames, nil)
	if err != nil {
		return fmt.Errorf("error generating configs for the Service Broker: %v", err)
	}

	err = removeConfigs(dir, gcpBrokerFileNames)
	if err != nil {
		return fmt.Errorf("error deleting broker resources: %v", err)
	}

	// due to moving the google-oauth resources to a separate namespace, we
	// must also remove deprecated Service Broker k8s resources for backwards
	// compatibility
	err = removeDeprecatedGCPBrokerResources()
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

// removeDeprecatedGCPBrokerResources removes GCP broker-related k8s resources
// that were created by a previous version of this tool in a different namespace,
// for backwards compatibility
func removeDeprecatedGCPBrokerResources() error {
	dir, err := ioutil.TempDir("/tmp", "service-catalog-gcp-deprecated")
	if err != nil {
		return fmt.Errorf("error creating temporary dir: %v", err)
	}

	defer os.RemoveAll(dir)

	err = generateConfigs(dir, gcpBrokerDeprecatedTemplateDir, gcpBrokerDeprecatedFileNames, nil)
	if err != nil {
		return fmt.Errorf("error generating configs: %v", err)
	}

	err = removeConfigs(dir, gcpBrokerDeprecatedFileNames)
	if err != nil {
		return err
	}

	return nil
}

func generateConfigs(genDir, templateDir string, filenames []string, data map[string]interface{}) error {
	for _, f := range filenames {
		if err := generateFileFromTmpl(filepath.Join(genDir, f+".yaml"), templateDir+f+".yaml.tmpl", data); err != nil {
			return err
		}
	}
	return nil
}

func deployConfigs(dir string, filenames []string) error {
	for _, f := range filenames {
		output, err := exec.Command("kubectl", "apply", "-f", filepath.Join(dir, f+".yaml")).CombinedOutput()
		// TODO: cleanup
		if err != nil {
			return fmt.Errorf("deploy failed with output: %s: %v", err, string(output))
		}
	}
	return nil
}

func removeConfigs(dir string, filenames []string) error {
	for _, f := range filenames {
		output, err := exec.Command("kubectl", "delete", "-f", filepath.Join(dir, f+".yaml"), "--ignore-not-found").CombinedOutput()
		// TODO: cleanup
		if err != nil {
			return fmt.Errorf("failed to delete resources output: %s: %v", err, string(output))
		}
	}
	return nil
}

type createBrokerConfig struct {
	name  string // name of the broker
	title string // title of the broker
}

// NewCreateGCPBrokerCmd returns a cobra command which creates a new GCP service
// broker without adding it to the existing Kubernetes cluster.
func NewCreateGCPBrokerCmd() *cobra.Command {
	cfg := &createBrokerConfig{}
	cmd := &cobra.Command{
		Use:   "create-gcp-broker",
		Short: "Create the Service Broker",
		Long:  "Creates Google Cloud Platform Service Broker without adding it to an existing cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			return createGCPBroker(cfg)
		},
	}
	cmd.Flags().StringVar(&cfg.name, "name", "default", "Broker name, lowercase, hyphens allowed")
	cmd.Flags().StringVar(&cfg.title, "title", "Default Broker", "A title of the broker for display")
	return cmd
}

func createGCPBroker(cfg *createBrokerConfig) error {
	projectID, err := gcp.GetConfigValue("core", "project")
	if err != nil {
		return fmt.Errorf("error getting configured project value : %v", err)
	}

	fmt.Println("using project: ", projectID)

	if err := enableRequiredAPIs(projectID); err != nil {
		return err
	}

	vb, err := getOrCreateVirtualBroker(projectID, cfg.name, cfg.title)
	if err != nil {
		return fmt.Errorf("error retrieving or creating default broker : %v", err)
	}

	msg := "Created a new"
	if vb.Existing {
		msg = "Reused an existing"
	}

	fmt.Printf(`%s Service Broker:
    Name:  %s
    Title: %s
    URL:   %s
`, msg, vb.Name, vb.Title, vb.URL)

	return err
}
