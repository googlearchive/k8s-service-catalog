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
	"encoding/base64"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
)

// InstallConfig contains installation configuration.
type InstallConfig struct {
	// namespace for service catalog
	Namespace string

	// APIServerServiceName refers to the API Server's service name
	APIServerServiceName string

	// whether to delete temporary files
	CleanupTempDirOnSuccess bool

	// generate YAML files for deployment, do not deploy them
	DryRun bool

	// CA options (self sign or use kubernetes root CA)

	// storage options to be implemented
}

type SSLArtifacts struct {
	// CA related SSL files
	CAFile           string
	CAPrivateKeyFile string

	// API Server related SSL files
	APIServerCertFile       string
	APIServerPrivateKeyFile string
}

func uninstallServiceCatalog(dir string) error {
	// ns := "service-catalog"

	files := []string{
		"apiserver-deployment.yaml",
		"controller-manager-deployment.yaml",
		"tls-cert-secret.yaml",
		"etcd-svc.yaml",
		"etcd.yaml",
		"api-registration.yaml",
		"service.yaml",
		"rbac.yaml",
		"service-accounts.yaml",
		"namespace.yaml",
	}

	for _, f := range files {
		output, err := exec.Command("kubectl", "delete", "-f", filepath.Join(dir, f)).CombinedOutput()
		if err != nil {
			fmt.Errorf("error deleting resources in file: %v :: %v", f, string(output))
			// TODO(droot): ignore failures and continue for deleting
			continue
			// return fmt.Errorf("deploy failed with output: %s :%v", err, output)
		}
	}
	return nil
}

func installServiceCatalog(ic *InstallConfig) error {

	if err := checkDependencies(); err != nil {
		return err
	}

	// create temporary directory for k8s artifacts and other temporary files
	dir, err := ioutil.TempDir("/tmp", "service-catalog")
	if err != nil {
		return fmt.Errorf("error creating temporary dir: %v", err)
	}

	if ic.CleanupTempDirOnSuccess {
		defer os.RemoveAll(dir)
	}

	sslArtifacts, err := generateSSLArtificats(dir, ic)
	if err != nil {
		return fmt.Errorf("error generating SSL artifacts : %v", err)
	}

	fmt.Printf("generated ssl artifacts: %+v \n", sslArtifacts)

	err = generateDeploymentConfigs(dir, sslArtifacts)
	if err != nil {
		return fmt.Errorf("error generating YAML files: %v", err)
	}

	if ic.DryRun {
		return nil
	}

	err = deploy(dir)
	if err != nil {
		return fmt.Errorf("error deploying YAML files: %v", err)
	}

	fmt.Println("Service Catalog installed successfully")
	return nil
}

// generateCertConfig generates config files required for generating CA and
// SSL certificates for API Server.
func generateCertConfig(dir string, ic *InstallConfig) (caCSRFilepath, certConfigFilePath string, err error) {
	host1 := fmt.Sprintf("%s.%s", ic.APIServerServiceName, ic.Namespace)
	host2 := host1 + ".svc"

	data := map[string]string{
		"Host1":          host1,
		"Host2":          host2,
		"APIServiceName": ic.APIServerServiceName,
	}

	caCSRFilepath = filepath.Join(dir, "ca_csr.json")
	err = generateFileFromTmpl(caCSRFilepath, "templates/ca_csr.json.tmpl", data)
	if err != nil {
		return
	}

	certConfigFilePath = filepath.Join(dir, "gencert_config.json")
	err = generateFileFromTmpl(certConfigFilePath, "templates/gencert_config.json.tmpl", data)
	if err != nil {
		return
	}
	return
}

func generateDeploymentConfigs(dir string, sslArtifacts *SSLArtifacts) error {
	ca, err := base64FileContent(sslArtifacts.CAFile)
	if err != nil {
		return err
	}
	apiServerCert, err := base64FileContent(sslArtifacts.APIServerCertFile)
	if err != nil {
		return err
	}
	apiServerPK, err := base64FileContent(sslArtifacts.APIServerPrivateKeyFile)
	if err != nil {
		return err
	}

	data := map[string]string{
		"CAPublicKey":          ca,
		"APIServicePublicKey":  apiServerCert,
		"APIServicePrivateKey": apiServerPK,
	}

	// err = generateFileFromTmpl(filepath.Join(dir, "api-registration.yaml"), "templates/api-registration.yaml.tmpl", data)
	// if err != nil {
	// 	return err
	// }
	//
	// err = generateFileFromTmpl(filepath.Join(dir, "tls-cert-secret.yaml"), "templates/tls-cert-secret.yaml.tmpl", data)
	// if err != nil {
	// 	return err
	// }

	files := []string{
		"tls-cert-secret",
		"api-registration",
		"namespace",
		"service-accounts",
		"rbac",
		"service",
		"etcd",
		"etcd-svc",
		"apiserver-deployment",
		"controller-manager-deployment",
	}
	for _, f := range files {
		err = generateFileFromTmpl(filepath.Join(dir, f+".yaml"), "templates/"+f+".yaml.tmpl", data)
		if err != nil {
			return err
		}
		// err := generateFile("templates/"+f, filepath.Join(dir, f))
		// if err != nil {
		// 	return err
		// }
	}
	return nil
}

func deploy(dir string) error {
	files := []string{
		"namespace.yaml",
		"service-accounts.yaml",
		"rbac.yaml",
		"service.yaml",
		"api-registration.yaml",
		"etcd.yaml",
		"etcd-svc.yaml",
		"tls-cert-secret.yaml",
		"apiserver-deployment.yaml",
		"controller-manager-deployment.yaml"}

	for _, f := range files {
		output, err := exec.Command("kubectl", "create", "-f", filepath.Join(dir, f)).CombinedOutput()
		// TODO(droot): cleanup
		if err != nil {
			return fmt.Errorf("deploy failed with output: %s :%v", err, string(output))
		}
	}
	return nil
}

func generateFileFromTmpl(dst, src string, data map[string]string) error {
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

func generateSSLArtificats(dir string, ic *InstallConfig) (result *SSLArtifacts, err error) {
	csrInputJSON, certGenJSON, err := generateCertConfig(dir, ic)
	if err != nil {
		err = fmt.Errorf("error generating cert config :%v", err)
		return
	}

	certConfigFilePath := filepath.Join(dir, "ca_config.json")
	err = generateFile("templates/ca_config.json", certConfigFilePath)
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

	result = &SSLArtifacts{
		CAFile:                  caFilePath + ".pem",
		CAPrivateKeyFile:        caFilePath + "-key.pem",
		APIServerPrivateKeyFile: apiServerCertFilePath + "-key.pem",
		APIServerCertFile:       apiServerCertFilePath + ".pem",
	}
	return
}
