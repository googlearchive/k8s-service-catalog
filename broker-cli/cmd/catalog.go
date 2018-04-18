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

package cmd

import (
	"fmt"
	"log"

	"github.com/GoogleCloudPlatform/k8s-service-catalog/broker-cli/client/adapter"
	"github.com/GoogleCloudPlatform/k8s-service-catalog/broker-cli/cmd/flags"
	"github.com/spf13/cobra"
)

var (
	catalogFlags struct {
		flags.BrokerURLConstructor
		apiVersion string
	}
	// catalogCmd represents the catalogs command.
	catalogCmd = &cobra.Command{
		Use:   "catalog",
		Short: "Get broker catalog",
		Long:  "Get broker catalog",
		Run: func(cmd *cobra.Command, args []string) {
			client := httpAdapterFromFlag()
			brokerURL, err := catalogFlags.BrokerURL()
			if err != nil {
				log.Fatalf("Error getting catalog: %v\n", err)
			}

			res, err := client.GetCatalog(&adapter.GetCatalogParams{
				APIVersion: catalogFlags.apiVersion,
				Server:     brokerURL,
			})
			if err != nil {
				log.Fatalf("Error getting catalog %q: %v\n", brokerURL, err)
			}

			if len(res.Services) == 0 {
				fmt.Printf("Broker %q in project %q has no associated services\n", catalogFlags.Broker, catalogFlags.Project)
				return
			}

			fmt.Printf("Successfully fetched service catalog for broker %q within project %q!!\n\n", catalogFlags.Broker, catalogFlags.Project)
			fmt.Println("Services:")
			for index, svc := range res.Services {
				fmt.Printf("%d. %s (%s)\n", index+1, svc.Name, svc.ID)
				fmt.Printf("   Description: %s\n", svc.Description)
				fmt.Printf("   Plans:\n")
				for index, plan := range svc.Plans {
					fmt.Printf("   %d. %s (%s)\n", index+1, plan.Name, plan.ID)
					fmt.Printf("      Description: %s", plan.Description)
					fmt.Printf("      Bindable: %t\n\n", *plan.Bindable)
				}
				fmt.Println()
			}
		},
	}
)

func init() {
	flags.StringFlag(catalogCmd.PersistentFlags(), &catalogFlags.Server, flags.ServerLongName, flags.ServerShortName, fmt.Sprintf("[Required if %s and %s are not given] Broker URL to make request to (https://...).", flags.ProjectLongName, flags.BrokerLongName))
	flags.StringFlag(catalogCmd.PersistentFlags(), &catalogFlags.Project, flags.ProjectLongName, flags.ProjectShortName, fmt.Sprintf("[Required if %s is not given] the GCP project of the broker", flags.ServerLongName))
	flags.StringFlag(catalogCmd.PersistentFlags(), &catalogFlags.Broker, flags.BrokerLongName, flags.BrokerShortName, fmt.Sprintf("[Required if %s is not given] the broker name", flags.ServerLongName))
	flags.StringFlagWithDefault(catalogCmd.PersistentFlags(), &catalogFlags.apiVersion, flags.ApiVersionLongName, flags.ApiVersionShortName, flags.ApiVersionDefault,
		flags.ApiVersionDescription)

	// Host is the hostname to use for Service Broker API calls. There's no help message here since it's a hidden flag.
	catalogCmd.PersistentFlags().StringVar(&catalogFlags.Host, flags.HostLongName, flags.HostBrokerDefault, "")
	catalogCmd.PersistentFlags().MarkHidden(flags.HostLongName)

	RootCmd.AddCommand(catalogCmd)
}
