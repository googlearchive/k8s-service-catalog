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

package cmd

import (
	"fmt"
	"log"

	"broker-cli/client/adapter"
	"broker-cli/cmd/flags"
	"github.com/spf13/cobra"
)

var (
	brokerFlag   string
	titleFlag    string
	catalogsFlag []string

	// brokersCmd represents the brokers command.
	brokersCmd = &cobra.Command{
		Use:   "brokers",
		Short: "Manage service brokers",
		Long:  "Manage service brokers",
	}

	// brokersGetCmd represents the brokers create command.
	brokersCreateCmd = &cobra.Command{
		Use:   "create",
		Short: "Create a service broker",
		Long:  "Create a service broker",
		Run: func(cmd *cobra.Command, args []string) {
			checkFlags(&projectFlag, &serverFlag, &brokerFlag, &catalogsFlag)

			http := httpAdapterFromFlag()
			res, err := http.CreateBroker(&adapter.CreateBrokerParams{
				RegistryURL: serverFlag,
				Project:     projectFlag,
				Name:        brokerFlag,
				Title:       titleFlag,
				Catalogs:    catalogsFlag,
			})
			processResult(res, err)
		},
	}

	// brokersDeleteCmd represents the brokers delete command.
	brokersDeleteCmd = &cobra.Command{
		Use:   "delete",
		Short: "Delete a service broker",
		Long:  "Delete a service broker",
		Run: func(cmd *cobra.Command, args []string) {
			checkFlags(&projectFlag, &serverFlag, &brokerFlag)

			http := httpAdapterFromFlag()
			params := &adapter.DeleteBrokerParams{
				RegistryURL: serverFlag,
				Project:     projectFlag,
				Name:        brokerFlag,
			}
			res, err := http.DeleteBroker(params)
			processResult(res, err)
		},
	}

	// brokersGetCmd represents the brokers get command.
	brokersGetCmd = &cobra.Command{
		Use:   "get",
		Short: "Get a service broker",
		Long:  "Get a service broker",
		Run: func(cmd *cobra.Command, args []string) {
			checkFlags(&projectFlag, &serverFlag, &brokerFlag)

			http := httpAdapterFromFlag()
			res, err := http.GetBroker(&adapter.GetBrokerParams{
				RegistryURL: serverFlag,
				Project:     projectFlag,
				Name:        brokerFlag,
			})
			processResult(res, err)
		},
	}

	// brokersListCmd represents the brokers list command.
	brokersListCmd = &cobra.Command{
		Use:   "list",
		Short: "List service brokers for a project",
		Long:  "List service brokers for a project",
		Run: func(cmd *cobra.Command, args []string) {
			checkFlags(&projectFlag, &serverFlag)

			http := httpAdapterFromFlag()
			res, err := http.ListBrokers(&adapter.ListBrokersParams{
				RegistryURL: serverFlag,
				Project:     projectFlag})
			processResult(res, err)
		},
	}
)

func init() {
	flags.StringFlag(brokersCreateCmd.PersistentFlags(), &brokerFlag, "broker", "b", "Required. Name of broker to create")
	flags.StringFlag(brokersCreateCmd.PersistentFlags(), &titleFlag, "title", "t", "Required. Title of broker to create")
	// TODO(richardfung): can we make this more user friendly by not forcing them to specify projects/...?
	// TODO(richardfung): what should the short flag actually be here?
	flags.StringArrayFlag(brokersCreateCmd.PersistentFlags(), &catalogsFlag, "catalog", "g", "Required. Catalogs for broker to use. Should be of the form \"projects/<project>/catalogs/<catalog>\"")

	flags.StringFlag(brokersDeleteCmd.PersistentFlags(), &brokerFlag, "broker", "b", "Required. The name of the broker")

	flags.StringFlag(brokersGetCmd.PersistentFlags(), &brokerFlag, "broker", "b", "Required. The name of the broker")

	RootCmd.AddCommand(brokersCmd)
	brokersCmd.AddCommand(brokersCreateCmd, brokersDeleteCmd, brokersGetCmd, brokersListCmd)
}

func processResult(res []byte, err error) {
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(string(res))
}
