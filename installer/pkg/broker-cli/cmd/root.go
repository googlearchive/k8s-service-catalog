// Copyright © 2017 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package cmd contains all the commands in broker-cli
package cmd

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"broker-cli/auth"
	"broker-cli/client/adapter"
	"broker-cli/cmd/flags"
	"github.com/spf13/cobra"
)

// RootCmd represents the base command when called without any subcommands
var (
	RootCmd = &cobra.Command{
		Use:   "broker-cli",
		Short: "Service Broker Client CLI",
		Long: "broker-cli is the client CLI for Service Broker.\n" +
			"This application is a tool to call Service Broker\n" +
			"APIs directly.",
	}

	// Values that are set from flags.
	credsFlag   string
	projectFlag string
	serverFlag  string
)

func init() {
	flags.StringFlag(RootCmd.PersistentFlags(), &credsFlag, "creds", "c", "Optional. Private, json key file to use for authenticating requests. If not specified, we use gcloud authentication")
	flags.StringFlag(RootCmd.PersistentFlags(), &projectFlag, "project", "p", "GCP Project")
	flags.StringFlag(RootCmd.PersistentFlags(), &serverFlag, "server", "s", "[Required] Server URL to make request to, corresponding to either a broker or registry URL. (https://...)")
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// getContext returns a context using information from flags.
func getContext() context.Context {
	// TODO(richardfung): add flags so users can control this?
	return context.Background()
}

// checkFlags Checks whether all given flags were specified. If any are missing this
// will print an error message and call os.Exit(2).
// requiredFlags should be pointers to the flag variables (e.g. &credsFlag).
func checkFlags(requiredFlags ...interface{}) {
	if missingFlags := flags.CheckRequiredFlags(requiredFlags...); missingFlags != nil {
		flags.PrintMissingFlags(missingFlags)
		os.Exit(2)
	}
}

// httpAdapterFromFlag returns an http adapter with credentials to gcloud if
// credsFlag is not set and to a service account if it is set.
func httpAdapterFromFlag() *adapter.HttpAdapter {
	var client *http.Client
	var err error
	if credsFlag != "" {
		client, err = auth.HttpClientFromFile(getContext(), credsFlag)
		if err != nil {
			log.Fatalf("Error creating http client from service account file %s: %v", credsFlag, err)
		}
	} else {
		client, err = auth.HttpClientWithDefaultCredentials(getContext())
		if err != nil {
			log.Fatalf("Error creating http client using gcloud credentials: %v", err)
		}
	}
	return adapter.NewHttpAdapter(client)
}
