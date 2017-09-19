/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const (
	CmdGCloud    = "gcloud"
	CmdKubectl   = "kubectl"
	CmdCFSSL     = "cfssl"
	CmdCFSSLJson = "cfssljson"
)

func main() {

	var cmdCheck = &cobra.Command{
		Use:   "check",
		Short: "performs a dependency check",
		Long: `This utility requires cfssl, gcloud, kubectl binaries to be 
present in PATH. This command performs the dependency check.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := checkDependencies(); err != nil {
				fmt.Println("Dependency check failed")
				fmt.Println(err)
				return
			}
			fmt.Println("Dependency check passed. You are good to go.")
		},
	}

	var cmdInstallServiceCatalog = &cobra.Command{
		Use:   "install-service-catalog",
		Short: "installs Service Catalog in Kubernetes cluster",
		Long: `installs Service Catalog in Kubernetes cluster.
assumes kubectl is configured to connect to the Kubernetes cluster.`,
		// Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := installServiceCatalog(); err != nil {
				fmt.Println("Service Catalog could not be installed")
				fmt.Println(err)
				return
			}
		},
	}

	var cmdUninstallServiceCatalog = &cobra.Command{
		Use:   "uninstall-service-catalog",
		Short: "uninstalls Service Catalog in Kubernetes cluster",
		Long: `uninstalls Service Catalog in Kubernetes cluster.
assumes kubectl is configured to connect to the Kubernetes cluster.`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := uninstallServiceCatalog(args[0]); err != nil {
				fmt.Println("Service Catalog could not be installed")
				fmt.Println(err)
				return
			}
		},
	}

	var rootCmd = &cobra.Command{Use: "installer"}
	rootCmd.AddCommand(
		cmdCheck,
		cmdInstallServiceCatalog,
		cmdUninstallServiceCatalog,
	)
	rootCmd.Execute()
}

// checkDependencies performs a lookup for binary executables that are
// required for installing service catalog and configuring GCP broker.
// TODO(droot): enhance it to perform connectivity check with Kubernetes Cluster
// and user permissions etc.
func checkDependencies() error {
	requiredCmds := []string{CmdGCloud, CmdKubectl, CmdCFSSL, CmdCFSSLJson}

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
	return nil
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

func installServiceCatalog() error {
	if err := checkDependencies(); err != nil {
		return err
	}

	ns := "service-catalog"

	dir, err := ioutil.TempDir("/tmp", "service-catalog")
	if err != nil {
		return fmt.Errorf("error creating temporary dir: %v", err)
	}

	// defer os.RemoveAll(dir)

	caFilePath, apiServerCertFilePath, apiServerPKFilePath, err := generateSSLArtificats(ns, dir)
	if err != nil {
		return fmt.Errorf("error generating SSL artifacts : %v", err)
	}

	fmt.Printf("generated caFilePath: %s, apiServerCertFilePath: %s, apiServerPKFilePath: %v \n",
		caFilePath, apiServerCertFilePath, apiServerPKFilePath)
	err = generateYAMLs(dir, caFilePath, apiServerCertFilePath, apiServerPKFilePath)
	if err != nil {
		return fmt.Errorf("error generating YAML files: %v", err)
	}

	err = deployYAML(dir)
	if err != nil {
		return fmt.Errorf("error deploying YAML files: %v", err)
	}
	fmt.Println("Service Catalog installed successfully")

	return nil
}

func generateYAMLs(dir, caFilePath, apiServerCertFilePath, apiServerPKFilePath string) error {
	ca, err := base64FileContent(caFilePath)
	if err != nil {
		return err
	}
	apiServerCert, err := base64FileContent(apiServerCertFilePath)
	if err != nil {
		return err
	}
	apiServerPK, err := base64FileContent(apiServerPKFilePath)
	if err != nil {
		return err
	}

	data := map[string]string{
		"CA_PUBLIC_KEY":   ca,
		"SVC_PUBLIC_KEY":  apiServerCert,
		"SVC_PRIVATE_KEY": apiServerPK,
	}

	err = generateYAML(filepath.Join(dir, "api-registration.yaml"), "templates/api-registration.yaml.tmpl", data)
	if err != nil {
		return err
	}

	err = generateYAML(filepath.Join(dir, "tls-cert-secret.yaml"), "templates/tls-cert-secret.yaml.tmpl", data)
	if err != nil {
		return err
	}

	files := []string{"namespace.yaml", "service-accounts.yaml", "rbac.yaml", "service.yaml", "etcd.yaml", "etcd-svc.yaml", "apiserver-deployment.yaml", "controller-manager-deployment.yaml"}

	for _, f := range files {
		err := dumpYAML("templates/"+f, filepath.Join(dir, f))
		if err != nil {
			return err
		}
	}
	return nil
}

func deployYAML(dir string) error {
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

func generateYAML(dst, src string, data map[string]string) error {
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

func dumpYAML(src, dst string) error {
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
	fmt.Printf("\n['%s']\n", encoded)
	return
}

func generateSSLArtificats(ns, dir string) (caFilePath, apiServerCertFilePath, apiServerPKFilePath string, err error) {
	apiServerSvcName := "service-catalog-api"

	host1 := fmt.Sprintf("%s.%s", apiServerSvcName, ns)
	host2 := host1 + ".svc"

	csrInputJSON := fmt.Sprintf(`{
"hosts": [ "%s","%s" ],
"key": {
	"algo": "rsa",
	"size": 2048
},
"names": [
	{
		"C": "US",
		"L": "san jose",
		"O": "kube",
		"OU": "WWW",
		"ST": "California"
	}
]
}`, host1, host2)

	// fmt.Println(csrInputJSON)
	genKeyCmd := exec.Command("cfssl", "genkey", "--initca", "-")
	genKeyCmd.Stdin = strings.NewReader(csrInputJSON)

	// getKeyOut, err := genKeyCmd.CombinedOutput()
	// if err != nil {
	// 	return
	// }

	caFilePath = filepath.Join(dir, "ca")
	cmd2 := exec.Command("cfssljson", "-bare", caFilePath)

	out, outErr, err := Pipeline(genKeyCmd, cmd2)
	if err != nil {
		return
	}

	fmt.Printf("%s: %s\n", out, outErr)

	certGenConfigContent := `{
"signing": { 
  "default": {
	"expiry": "43800h",
	"usages": [ "signing", "key encipherment", "server" ]
  }
}
}
`
	certConfigFilePath := filepath.Join(dir, "cert-config.json")
	err = ioutil.WriteFile(certConfigFilePath, []byte(certGenConfigContent), 0666)
	if err != nil {
		err = fmt.Errorf("error generating cert config : %v", err)
		return
	}

	certGenJSON := fmt.Sprintf(`
{
"CN": "%s",
"hosts": [ "%s","%s" ],
"key": {
  "algo": "rsa",
  "size": 2048
}
}
`, apiServerSvcName, host1, host2)
	certGenCmd := exec.Command("cfssl", "gencert",
		"-ca="+caFilePath+".pem",
		"-ca-key="+caFilePath+"-key.pem",
		"-config="+certConfigFilePath, "-")
	certGenCmd.Stdin = strings.NewReader(certGenJSON)

	apiServerCertFilePath = filepath.Join(dir, "apiserver")
	certSignCmd := exec.Command("cfssljson", "-bare", apiServerCertFilePath)

	_, _, err = Pipeline(certGenCmd, certSignCmd)
	if err != nil {
		err = fmt.Errorf("error signing api server cert: %v", err)
		return
	}

	caFilePath = caFilePath + ".pem"
	apiServerPKFilePath = apiServerCertFilePath + "-key.pem"
	apiServerCertFilePath = apiServerCertFilePath + ".pem"
	return
}

//
// Note: This code is copied from https://gist.github.com/kylelemons/1525278
//

// Pipeline strings together the given exec.Cmd commands in a similar fashion
// to the Unix pipeline.  Each command's standard output is connected to the
// standard input of the next command, and the output of the final command in
// the pipeline is returned, along with the collected standard error of all
// commands and the first error found (if any).
//
// To provide input to the pipeline, assign an io.Reader to the first's Stdin.
func Pipeline(cmds ...*exec.Cmd) (pipeLineOutput, collectedStandardError []byte, pipeLineError error) {
	// Require at least one command
	if len(cmds) < 1 {
		return nil, nil, nil
	}

	// Collect the output from the command(s)
	var output bytes.Buffer
	var stderr bytes.Buffer

	last := len(cmds) - 1
	for i, cmd := range cmds[:last] {
		var err error
		// Connect each command's stdin to the previous command's stdout
		if cmds[i+1].Stdin, err = cmd.StdoutPipe(); err != nil {
			return nil, nil, err
		}
		// Connect each command's stderr to a buffer
		cmd.Stderr = &stderr
	}

	// Connect the output and error for the last command
	cmds[last].Stdout, cmds[last].Stderr = &output, &stderr

	// Start each command
	for _, cmd := range cmds {
		if err := cmd.Start(); err != nil {
			return output.Bytes(), stderr.Bytes(), err
		}
	}

	// Wait for each command to complete
	for _, cmd := range cmds {
		if err := cmd.Wait(); err != nil {
			return output.Bytes(), stderr.Bytes(), err
		}
	}

	// Return the pipeline output and the collected standard error
	return output.Bytes(), stderr.Bytes(), nil
}
