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
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

// Binary names that we depend on.
const (
	GcloudBinaryName    = "gcloud"
	KubectlBinaryName   = "kubectl"
	CfsslBinaryName     = "cfssl"
	CfssljsonBinaryName = "cfssljson"
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

			ic := &InstallConfig{
				Namespace:               "service-catalog",
				APIServerServiceName:    "service-catalog-api",
				CleanupTempDirOnSuccess: false,
			}

			if err := installServiceCatalog(ic); err != nil {
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
	return nil
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
