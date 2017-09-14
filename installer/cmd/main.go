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
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

const (
	CmdGCloudName  = "gcloud"
	CmdKubectlName = "kubectl"
	CmdCFSSLName   = "cfssl"
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

	var rootCmd = &cobra.Command{Use: "installer"}
	rootCmd.AddCommand(
		cmdCheck,
		cmdInstallServiceCatalog,
	)
	rootCmd.Execute()
}

// checkDependencies performs a lookup for binary executables that are
// required for installing service catalog and configuring GCP broker.
// TODO(droot): enhance it to perform connectivity check with Kubernetes Cluster
// and user permissions etc.
func checkDependencies() error {
	requiredCmds := []string{CmdGCloudName, CmdKubectlName, CmdCFSSLName}

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

func installServiceCatalog() error {

	if err := checkDependencies(); err != nil {
		return err
	}
	fmt.Println("Service Catalog installed successfully")
	return nil
}
