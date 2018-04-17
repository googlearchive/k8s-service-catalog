// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Package cmd contains all the commands in broker-cli.
package cmd

import (
	"fmt"
	"os"

	"github.com/GoogleCloudPlatform/k8s-service-catalog/broker-cli/cmd/flags"
	"github.com/spf13/cobra"
)

// RootCmd represents the base command when called without any subcommands.
var (
	RootCmd = &cobra.Command{
		Use:   "broker-cli",
		Short: "Service Broker Client CLI",
		Long: "broker-cli is the client CLI for Service Broker.\n" +
			"This application is a tool to call Service Broker\n" +
			"APIs directly.",
	}

	// Values that are set from flags.
	credsFlag string
)

func init() {
	flags.StringFlag(RootCmd.PersistentFlags(), &credsFlag, "creds", "c", "[Optional] Private, json key file to use for authenticating requests. If not specified, we use gcloud authentication.")
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
