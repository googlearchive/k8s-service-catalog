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
	"flag"
	"os"

	"github.com/golang/glog"
	"github.com/spf13/cobra"

	"github.com/GoogleCloudPlatform/k8s-service-catalog/installer/pkg/cmd"
)

func main() {
	defer glog.Flush()

	c := NewCommand()
	if err := c.Execute(); err != nil {
		os.Exit(1)
	}
}

func NewCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "sc",
		Short: "CLI to manage Service Catalog in a Kubernetes Cluster",
		Long: `sc is a CLI for managing lifecycle of Service Catalog and 
Service brokers in a Kubernetes Cluster. It implements commands to
install, uninstall Service Catalog and add/remove GCP service broker
in a Kubernets Cluster.`,

		// turn off the usage by default on any error
		SilenceUsage: true,
	}
	c.AddCommand(
		cmd.NewCheckDependenciesCmd(),
		cmd.NewServiceCatalogInstallCmd(),
		cmd.NewServiceCatalogUnInstallCmd(),
		cmd.NewAddGCPBrokerCmd(),
		cmd.NewRemoveGCPBrokerCmd(),
		cmd.NewUpdateCmd(),
		cmd.NewVersionCmd(),
	)

	// Add any globals flags here

	// add the glog flags
	c.PersistentFlags().AddGoFlagSet(flag.CommandLine)

	return c
}
