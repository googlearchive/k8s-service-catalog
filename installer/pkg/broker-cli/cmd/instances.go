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
	"encoding/json"
	"fmt"
	"log"

	"broker-cli/client/adapter"
	"broker-cli/cmd/flags"
	"github.com/spf13/cobra"
)

var (
	instanceIDFlag        string
	acceptsIncompleteFlag bool
	serviceIDFlag         string
	planIDFlag            string
	organizationGUIDFlag  string
	spaceGUIDFlag         string
	parametersFlag        string
	contextFlag           string

	// instancesCmd represents the instances command.
	instancesCmd = &cobra.Command{
		Use:   "instances",
		Short: "Manage service instances",
		Long:  "Manage service instances",
	}

	instancesCreateCmd = &cobra.Command{
		Use:   "create",
		Short: "Create a service instance",
		Long:  "Create a service instance",
		Run: func(cmd *cobra.Command, args []string) {
			checkFlags(&serverFlag, &instanceIDFlag, &serviceIDFlag, &planIDFlag)

			client := httpAdapterFromFlag()
			res, err := client.CreateInstance(&adapter.CreateInstanceParams{
				Server:            serverFlag,
				AcceptsIncomplete: acceptsIncompleteFlag,
				InstanceID:        instanceIDFlag,
				ServiceID:         serviceIDFlag,
				PlanID:            planIDFlag,
				Context:           parseStringToObjectMap(contextFlag),
				OrganizationGUID:  organizationGUIDFlag,
				SpaceGUID:         spaceGUIDFlag,
				Parameters:        parseStringToObjectMap(parametersFlag),
			})
			if err != nil {
				log.Fatalf("Error creating instance: %v\n", err)
			}

			if res.Async {
				fmt.Printf("Successfully started the operation to create instance: %+v", *res)
			} else {
				fmt.Printf("Successfully created the instance: %+v", *res)
			}

		},
	}
)

func init() {
	// TODO(maqiuyu): May need to improve the instruction for the serverFlag defined in the root.
	// Flags for `instances` command group and all subgroups.
	flags.StringFlag(instancesCmd.PersistentFlags(), &instanceIDFlag, "instance", "i",
		"[Required] Service Instance ID.")
	flags.BoolFlag(instancesCmd.PersistentFlags(), &acceptsIncompleteFlag, "asynchronous", "a",
		"[Optional] If specified, the broker will execute the request asynchronously. (Default: FALSE)")

	// Flags for `instances create` command group.
	// Flags with no short names won't show up in the help message so every flag has a unique but
	// weird short name.
	flags.StringFlag(instancesCreateCmd.PersistentFlags(), &serviceIDFlag, "service", "r",
		"[Required] The service ID used to create the service instance.")
	flags.StringFlag(instancesCreateCmd.PersistentFlags(), &planIDFlag, "plan", "l",
		"[Required] The plan ID used to create the service instance.")
	flags.StringFlag(instancesCreateCmd.PersistentFlags(), &contextFlag, "context", "t",
		"[Optional] [JSON Object] Platform specific contextual information under which the service "+
			"instance is to be provisioned.")
	flags.StringFlag(instancesCreateCmd.PersistentFlags(), &organizationGUIDFlag, "organization", "o",
		"[Optional] [Deprecated in favor of 'Context'] The platform GUID for the organization under"+
			" which the service instance is to be provisioned.")
	flags.StringFlag(instancesCreateCmd.PersistentFlags(), &spaceGUIDFlag, "space", "e",
		"[Optional] [Deprecated in favor of 'Context'] The identifier for the project space within "+
			"the platform organization.")
	flags.StringFlag(instancesCreateCmd.PersistentFlags(), &parametersFlag, "parameters", "m",
		"[Optional] [JSON Object] Configuration options for the service instance.")

	RootCmd.AddCommand(instancesCmd)
	instancesCmd.AddCommand(instancesCreateCmd)
}

func parseStringToObjectMap(s string) map[string]interface{} {
	if s == "" {
		return nil
	}

	var objMap map[string]interface{}
	err := json.Unmarshal([]byte(s), &objMap)
	if err != nil {
		log.Fatalf("Error unmarshalling string %q to object map: %v\n", s, err)
	}

	return objMap
}
