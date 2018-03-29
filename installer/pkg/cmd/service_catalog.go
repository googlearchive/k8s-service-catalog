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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/k8s-service-catalog/installer/pkg/gcp"
	"github.com/GoogleCloudPlatform/k8s-service-catalog/installer/pkg/version"
	"github.com/Masterminds/semver"
	"github.com/spf13/cobra"
)

// Binary names that we depend on.
const (
	GcloudBinaryName    = "gcloud"
	KubectlBinaryName   = "kubectl"
	CfsslBinaryName     = "cfssl"
	CfssljsonBinaryName = "cfssljson"
)

// service catalog resources that will be created as part of deployment.
var (
	svcCatalogFileNames = []k8sResource{
		{name: "namespace"},
		{name: "etcd-operator-rbac"},
		{name: "etcd-operator-service-account"},
		{name: "etcd-operator-rbac-binding"},
		{name: "etcd-operator-deployment"},
		{name: "tls-cert-secret"},
		{name: "api-registration"},
		{name: "service-accounts"},
		{name: "rbac"},
		{name: "service"},
		{name: "apiserver-deployment"},
		{name: "controller-manager-deployment"},
		{name: "etcd-cluster-with-backup", dependsOnAPI: "etcd.database.coreos.com/v1beta2"},
	}
)

type k8sResource struct {
	name string
	// API that this resource depends on
	dependsOnAPI string
}

// InstallConfig contains installation configuration.
type InstallConfig struct {
	// namespace for service catalog
	Namespace string

	// Version of Service Catalog
	Version string

	// APIServerServiceName refers to the API Server's service name
	APIServerServiceName string

	// whether to delete temporary files
	CleanupTempDirOnSuccess bool

	// generate YAML files for deployment, do not deploy them
	DryRun bool

	// CA options (self sign or use kubernetes root CA)

	// storage options
	EtcdClusterSize        int32
	EtcdBackupStorageClass string
}

func NewServiceCatalogInstallCmd() *cobra.Command {
	ic := &InstallConfig{
		Namespace:               "service-catalog",
		APIServerServiceName:    "service-catalog-api",
		CleanupTempDirOnSuccess: false,
		EtcdClusterSize:         3,
		EtcdBackupStorageClass:  "standard",
	}
	c := &cobra.Command{
		Use:   "install",
		Short: "installs Service Catalog in Kubernetes cluster",
		Long: `installs Service Catalog in Kubernetes cluster.
assumes kubectl is configured to connect to the Kubernetes cluster.`,
		// Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := installServiceCatalog(ic); err != nil {
				fmt.Println("Service Catalog could not be installed.")
				return err
			}
			fmt.Println("Service Catalog installed successfully.")
			return nil
		},
	}
	// add install command flags
	c.Flags().Int32Var(&ic.EtcdClusterSize, "etcd-cluster-size", 3, "Etcd cluster size")
	c.Flags().StringVar(&ic.EtcdBackupStorageClass, "etcd-backup-storageclass", "standard", "Etcd Backup StorageClass")
	c.Flags().StringVar(&ic.Version, "version", "0.1.11-gke.0", "Service Catalog version")
	c.Flags().BoolVar(&ic.DryRun, "dryrun", false, "Dryrun")

	return c
}

func installServiceCatalog(ic *InstallConfig) error {
	if err := checkDependencies(); err != nil {
		return err
	}

	backupStorageClassExists, err := storageClassExists(ic.EtcdBackupStorageClass)
	if err != nil {
		return err
	}

	if !backupStorageClassExists {
		return fmt.Errorf("storageclass for etcd backup does not exist. " +
			"Use --etcd-backup-storageclass option to specify an existing storageclass")
	}

	dir, err := generateDeploymentConfigs(ic)
	if err != nil {
		return fmt.Errorf("error generating YAML files: %v", err)
	}

	fmt.Printf("generated service catalog deployment config in dir: %s \n", dir)

	if ic.CleanupTempDirOnSuccess {
		defer os.RemoveAll(dir)
	}

	if ic.DryRun {
		return nil
	}

	err = isAPIServerCompatible()
	if err != nil {
		return err
	}

	err = deployConfig(dir)
	if err != nil {
		if strings.Contains(err.Error(), "\"etcd-operator\" is forbidden: attempt to grant extra privileges") {
			fmt.Println("WARNING: Please run `kubectl create clusterrolebinding cluster-admin-binding --clusterrole=cluster-admin --user=$(gcloud config get-value account)` before `sc install`.")
		}

		return fmt.Errorf("error deploying YAML files: %v", err)
	}

	return nil
}

// generateDeploymentConfigs create configuration files for all the service
// catalog resources in a temporary directory under /tmp. It returns absolute
// path to the temporary directory containing the config.
func generateDeploymentConfigs(ic *InstallConfig) (string, error) {

	// create temporary directory for k8s artifacts and other temporary files
	dir, err := ioutil.TempDir("/tmp", "service-catalog")
	if err != nil {
		return "", fmt.Errorf("error creating temporary dir: %v", err)
	}

	sslArtifacts, err := generateSSLArtifacts(dir, ic)
	if err != nil {
		return dir, fmt.Errorf("error generating SSL artifacts : %v", err)
	}

	ca, err := base64FileContent(sslArtifacts.CAFile)
	if err != nil {
		return dir, err
	}
	apiServerCert, err := base64FileContent(sslArtifacts.APIServerCertFile)
	if err != nil {
		return dir, err
	}
	apiServerPK, err := base64FileContent(sslArtifacts.APIServerPrivateKeyFile)
	if err != nil {
		return dir, err
	}

	// TODO(mkibbe): Hard-code the default version of Service Catalog to a
	// known good one for now. We cannot guarantee that the "latest"-tagged
	// Service Catalog version is compatible with our templates. Later,
	// flesh out the upgrade story to be able to dynamically install the
	// latest version at an explicit versioned tag.
	imageTag := "v0.1.11-gke.0"
	if ic.Version != "" {
		imageTag = "v" + ic.Version
	}
	svcCatalogImage := "gcr.io/gcp-services/service-catalog:" + imageTag

	data := map[string]interface{}{
		"CAPublicKey":            ca,
		"APIServicePublicKey":    apiServerCert,
		"APIServicePrivateKey":   apiServerPK,
		"EtcdClusterSize":        ic.EtcdClusterSize,
		"EtcdBackupStorageClass": ic.EtcdBackupStorageClass,
		"ServiceCatalogImage":    svcCatalogImage,
		"Version":                version.GetVersion(),
	}

	for _, f := range svcCatalogFileNames {
		err = generateFileFromTmpl(filepath.Join(dir, f.name+".yaml"), "templates/sc/"+f.name+".yaml.tmpl", data)
		if err != nil {
			return dir, err
		}
	}
	return dir, nil
}

// this function assumes kubectl executable already exists in PATH.
func deployConfig(dir string) error {
	for _, f := range svcCatalogFileNames {
		if f.dependsOnAPI != "" {
			for {
				available, err := isAPIAvailable(f.dependsOnAPI)
				if err != nil {
					return fmt.Errorf("failed to check API availability : %v", err)
				}
				if available {
					break
				}
				time.Sleep(2 * time.Second)
			}
		}
		output, err := exec.Command("kubectl", "apply", "-f", filepath.Join(dir, f.name+".yaml")).CombinedOutput()
		// TODO(droot): cleanup
		if err != nil {
			return fmt.Errorf("deploy failed with output: %s :%v", err, string(output))
		}
	}
	return nil
}

// sslArtifacts contains SSL artifacts needed
type sslArtifacts struct {
	// CA related SSL files
	CAFile           string
	CAPrivateKeyFile string

	// API Server related SSL files
	APIServerCertFile       string
	APIServerPrivateKeyFile string
}

// generateCertConfig generates config files required for generating CA and
// SSL certificates for API Server.
func generateCertConfig(dir string, ic *InstallConfig) (caCSRFilepath, certConfigFilePath string, err error) {
	host1 := fmt.Sprintf("%s.%s", ic.APIServerServiceName, ic.Namespace)
	host2 := host1 + ".svc"

	data := map[string]interface{}{
		"Host1":          host1,
		"Host2":          host2,
		"APIServiceName": ic.APIServerServiceName,
	}

	caCSRFilepath = filepath.Join(dir, "ca_csr.json")
	err = generateFileFromTmpl(caCSRFilepath, "templates/sc/ca_csr.json.tmpl", data)
	if err != nil {
		return
	}

	certConfigFilePath = filepath.Join(dir, "gencert_config.json")
	err = generateFileFromTmpl(certConfigFilePath, "templates/sc/gencert_config.json.tmpl", data)
	if err != nil {
		return
	}
	return
}

func generateSSLArtifacts(dir string, ic *InstallConfig) (result *sslArtifacts, err error) {
	csrInputJSON, certGenJSON, err := generateCertConfig(dir, ic)
	if err != nil {
		err = fmt.Errorf("error generating cert config :%v", err)
		return
	}

	certConfigFilePath := filepath.Join(dir, "ca_config.json")
	err = generateFile("templates/sc/ca_config.json", certConfigFilePath)
	if err != nil {
		err = fmt.Errorf("error generating ca config: %v", err)
		return
	}

	genKeyCmd := exec.Command("cfssl", "genkey", "--initca", csrInputJSON)

	caFilePath := filepath.Join(dir, "ca")
	cmd2 := exec.Command("cfssljson", "-bare", caFilePath)

	out, outErr, err := Pipeline(genKeyCmd, cmd2)
	if err != nil {
		err = fmt.Errorf("error generating ca: stdout: %v stderr: %v err: %v", string(out), string(outErr), err)
		return
	}

	certGenCmd := exec.Command("cfssl", "gencert",
		"-ca", caFilePath+".pem",
		"-ca-key", caFilePath+"-key.pem",
		"-config", certConfigFilePath, certGenJSON)

	apiServerCertFilePath := filepath.Join(dir, "apiserver")
	certSignCmd := exec.Command("cfssljson", "-bare", apiServerCertFilePath)

	_, _, err = Pipeline(certGenCmd, certSignCmd)
	if err != nil {
		err = fmt.Errorf("error signing api server cert: %v", err)
		return
	}

	result = &sslArtifacts{
		CAFile:                  caFilePath + ".pem",
		CAPrivateKeyFile:        caFilePath + "-key.pem",
		APIServerPrivateKeyFile: apiServerCertFilePath + "-key.pem",
		APIServerCertFile:       apiServerCertFilePath + ".pem",
	}
	return
}

func generateFileFromTmpl(dst, src string, data map[string]interface{}) error {
	b, err := Asset(src)
	if err != nil {
		return err
	}
	tp, err := template.New("").Parse(string(b))
	if err != nil {
		return err
	}

	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()

	err = tp.Execute(f, data)
	if err != nil {
		return err
	}
	return nil
}

func generateFile(src, dst string) error {
	b, err := Asset(src)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(dst, b, 0644)
}

func base64FileContent(filePath string) (encoded string, err error) {
	b, err := ioutil.ReadFile(filePath)
	if err != nil {
		return
	}
	encoded = base64.StdEncoding.EncodeToString(b)
	return
}

func deleteConfig(dir string) error {
	// delete the service catalog artifacts in reverse order
	for i := len(svcCatalogFileNames) - 1; i >= 0; i-- {
		f := svcCatalogFileNames[i]
		output, err := exec.Command("kubectl", "delete", "-f", filepath.Join(dir, f.name+".yaml")).CombinedOutput()
		if err != nil {
			fmt.Errorf("error deleting resources in file: %v :: %v", f.name, string(output))
			// TODO(droot): ignore failures and continue with deleting
			continue
			// return fmt.Errorf("deploy failed with output: %s :%v", err, output)
		}
	}
	return nil
}

func isServiceCatalogInstalled() (bool, error) {
	scAPI := "servicecatalog.k8s.io"

	found := false
	var err error
	if found, err = isAPIAvailable(scAPI); err != nil {
		return false, fmt.Errorf("failed to check if service catalog is installed :%v", err)
	}

	return found, err
}

// isAPIAvailable is a helper function to determine if an API is available in
// given Kubernetes cluster.
func isAPIAvailable(api string) (bool, error) {
	out, err := exec.Command("kubectl", "api-versions").Output()
	if err != nil {
		return false, err
	}

	return strings.Contains(string(out), api), nil
}

func NewServiceCatalogUnInstallCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "uninstall",
		Short: "uninstalls Service Catalog in Kubernetes cluster",
		Long: `uninstalls Service Catalog in Kubernetes cluster.
assumes kubectl is configured to connect to the Kubernetes cluster.`,
		// Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ns := "service-catalog"
			if err := uninstallServiceCatalog(ns); err != nil {
				fmt.Println("Service Catalog could not be installed")
				return err
			}
			return nil
		},
	}
	return c
}

func uninstallServiceCatalog(ns string) error {
	if err := checkDependencies(); err != nil {
		return err
	}

	ic := &InstallConfig{
		Namespace: ns,
		// Following fields are not used during installation, they are needed
		// for generating the DeploymentConfigs.
		EtcdClusterSize:        3,
		EtcdBackupStorageClass: "standard",
	}

	dir, err := generateDeploymentConfigs(ic)
	if err != nil {
		return fmt.Errorf("error generating YAML files: %v", err)
	}

	defer os.RemoveAll(dir)

	// It might take a while to delete the configs, so we want
	fmt.Println("deleting service catalog configs...")
	err = deleteConfig(dir)
	if err != nil {
		return fmt.Errorf("error undeploying YAML files: %v", err)
	}

	// Namespaces are deleted asynchronuously and we need to make sure the
	// deletion is actually done before printing the success message.
	waitOnNSDeletion()

	fmt.Println("Service Catalog uninstalled successfully.")
	return nil
}

// waitOnNSDeletion keeps checking whether namespace "service-catalog" is deleted.
func waitOnNSDeletion() {
	baseDelay := 100 * time.Millisecond
	maxDelay := 6 * time.Second
	retries := 0

	for {
		delay := time.Duration(math.Pow(2, float64(retries)) * float64(baseDelay))
		if delay > maxDelay {
			delay = maxDelay
		}
		time.Sleep(delay)

		if _, err := exec.Command("kubectl", "get", "namespace", "service-catalog" /* Namespace for GCP broker*/).CombinedOutput(); err != nil {
			// TODO(maqiuyujoyce): Check whether the error is a not found error.
			return
		}

		retries++
	}
	return
}

func NewCheckDependenciesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "performs a dependency check",
		Long: `This utility requires cfssl, gcloud, kubectl binaries to be 
present in PATH. This command performs the dependency check.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkDependencies(); err != nil {
				fmt.Println("Dependency check failed")
				return err
			}
			fmt.Println("Dependency check passed. You are good to go.")
			return nil
		},
	}
}

// checkDependencies performs a lookup for binary executables that are
// required for installing service catalog and configuring GCP broker.
// TODO(droot): enhance it to perform connectivity check with Kubernetes Cluster
// and user permissions etc.
func checkDependencies() error {
	requiredCmds := []string{GcloudBinaryName, KubectlBinaryName, CfsslBinaryName, CfssljsonBinaryName}

	var missingCmds []string
	for _, cmd := range requiredCmds {
		_, err := exec.LookPath(cmd)
		if err != nil {
			missingCmds = append(missingCmds, cmd)
		}
	}

	if len(missingCmds) > 0 {
		return fmt.Errorf("%s commands not found in the PATH", strings.Join(missingCmds, ","))
	}

	// Also print out current account, project and zone information.
	configs, err := gcp.GetConfigMap()
	if err != nil {
		return fmt.Errorf("error retrieving gcloud config: %v", err)
	}

	fmt.Printf("account: %s\n", getValueFromConfigMap("core", "account", configs))
	fmt.Printf("project: %s\n", getValueFromConfigMap("core", "project", configs))
	fmt.Printf("zone: %s\n", getValueFromConfigMap("compute", "zone", configs))

	return nil
}

func getValueFromConfigMap(section, property string, configs map[string]interface{}) string {
	switch s := configs[section].(type) {
	case map[string]interface{}:
		switch p := s[property].(type) {
		case string:
			return p
		default:
			// No value or unexpected type
		}
	default:
		// No value or unexpected type.
	}
	return ""
}

func storageClassExists(name string) (bool, error) {
	output, err := exec.Command(KubectlBinaryName, "get", "storageclass", name).CombinedOutput()
	if err != nil {
		outputStr := string(output)
		if strings.Contains(outputStr, "NotFound") {
			return false, nil
		}
		return false, fmt.Errorf("error getting serviceclasses: %v %v", outputStr, err)
	}
	return true, nil
}

// isServerCompatible performs following checks:
// Kubernetes 1.7+
// TODO(droot): configured for Mutual TLS
func isAPIServerCompatible() error {
	v, err := getServerVersion()
	if err != nil {
		return err
	}

	ver17 := semver.MustParse("1.7.0")
	if v.LessThan(ver17) {
		return fmt.Errorf("Service Catalog requires Kubernetes v1.7+.")
	}
	return nil
}

func getServerVersion() (*semver.Version, error) {
	output, err := exec.Command(KubectlBinaryName, "version", "-o", "json").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("error fetching Kubernetes version :%v", string(output))
	}

	var versions map[string]k8sVersion

	err = json.Unmarshal(output, &versions)
	if err != nil {
		return nil, fmt.Errorf("error Unmarshal version info: %v", err)
	}

	serverVersion, found := versions["serverVersion"]
	if !found {
		return nil, fmt.Errorf("error getting server version")
	}

	return semver.NewVersion(serverVersion.GitVersion)
}

type k8sVersion struct {
	GitVersion string `json:"gitVersion"`
}
