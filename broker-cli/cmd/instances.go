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
	"github.com/GoogleCloudPlatform/k8s-service-catalog/broker-cli/client/osb"
	"github.com/GoogleCloudPlatform/k8s-service-catalog/broker-cli/cmd/flags"
	"github.com/spf13/cobra"
)

// Instance is a Service Instance.
type Instance struct {
	ID            string `json:"instance_id"`
	ServiceID     string `json:"service_id"`
	PlanID        string `json:"plan_id"`
	CreateTime    string `json:"createTime"`
	NumOfBindings int    `json:"numOfBindings"`
}

// listInstancesResult is the output from ListInstances.
type listInstancesResult struct {
	Instances []*Instance `json:"instances"`
}

var (
	instancesFlags struct {
		flags.BrokerURLConstructor
		apiVersion             string
		instanceID             string
		acceptsIncomplete      bool
		serviceID              string
		planID                 string
		organizationGUID       string
		spaceGUID              string
		parameters             string
		context                string
		wait                   bool
		operationID            string
		previousServiceID      string
		previousPlanID         string
		previousOrganizationID string
		previousSpaceID        string
	}

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
			flags.CheckFlags(&instancesFlags.instanceID, &instancesFlags.serviceID, &instancesFlags.planID)

			client := httpAdapterFromFlag()
			brokerURL, err := instancesFlags.BrokerURL()
			if err != nil {
				log.Fatalf("Error creating instance %s: %v", instancesFlags.instanceID, err)
			}
			res, err := client.CreateInstance(&adapter.CreateInstanceParams{
				Server:            brokerURL,
				APIVersion:        instancesFlags.apiVersion,
				AcceptsIncomplete: instancesFlags.acceptsIncomplete,
				InstanceID:        instancesFlags.instanceID,
				ServiceID:         instancesFlags.serviceID,
				PlanID:            instancesFlags.planID,
				Context:           parseStringToObjectMap(instancesFlags.context),
				OrganizationGUID:  instancesFlags.organizationGUID,
				SpaceGUID:         instancesFlags.spaceGUID,
				Parameters:        parseStringToObjectMap(instancesFlags.parameters),
			})
			if err != nil {
				log.Fatalf("Error creating instance %s in broker %s: %v", instancesFlags.instanceID, brokerURL, err)
			}

			if !res.Async {
				fmt.Printf("Successfully created the instance %s: %+v\n", instancesFlags.instanceID, *res)
				return
			}

			if !instancesFlags.wait {
				fmt.Printf("Successfully started the operation to create instance %s: %+v\n", instancesFlags.instanceID, *res)
				return
			}

			op, err := waitOnOperation(pollInstanceOpFunc(client, instancesFlags.apiVersion, brokerURL, instancesFlags.instanceID, instancesFlags.serviceID,
				instancesFlags.planID, res.OperationID, adapter.OperationCreate), false)
			if err != nil {
				log.Fatalf("Error polling last operation %q for instance %s: %v", res.OperationID, instancesFlags.instanceID, err)
			}

			if op.State == adapter.OperationSucceeded {
				fmt.Printf("Successfully created the instance %s asynchronously (operation %q): %+v\n", instancesFlags.instanceID, res.OperationID, *op)
				return
			}

			log.Fatalf("Failed creating instance %s asynchronously (operation %q): %+v\n", instancesFlags.instanceID, res.OperationID, *op)
		},
	}

	instancesListCmd = &cobra.Command{
		Use:   "list",
		Short: "List service instances in a broker",
		Long:  "List service instances in a broker",
		Run: func(cmd *cobra.Command, args []string) {
			brokerURL, err := instancesFlags.BrokerURL()
			if err != nil {
				log.Fatalf("Error listing instances: %v", err)
			}
			res, err := listInstances(brokerURL)
			if err != nil {
				log.Fatalf("Error listing instances in broker %s: %v", brokerURL, err)
			}

			if len(res.Instances) == 0 {
				fmt.Printf("Broker %q in project %q has no associated instances\n", instancesFlags.Broker, instancesFlags.Project)
				return
			}

			fmt.Printf("Successfully listed service instances in broker %q within project %q!!\n\n", instancesFlags.Broker, instancesFlags.Project)
			printListInstances(res)

		},
	}

	instancesDeleteCmd = &cobra.Command{
		Use:   "delete",
		Short: "Delete a service instance",
		Long:  "Delete a service instance",
		Run: func(cmd *cobra.Command, args []string) {
			flags.CheckFlags(&instancesFlags.instanceID, &instancesFlags.serviceID, &instancesFlags.planID)

			client := httpAdapterFromFlag()
			brokerURL, err := instancesFlags.BrokerURL()
			if err != nil {
				log.Fatalf("Error deleting instance %s: %v", instancesFlags.instanceID, err)
			}
			res, err := client.DeleteInstance(&adapter.DeleteInstanceParams{
				APIVersion:        instancesFlags.apiVersion,
				Server:            brokerURL,
				AcceptsIncomplete: instancesFlags.acceptsIncomplete,
				InstanceID:        instancesFlags.instanceID,
				ServiceID:         instancesFlags.serviceID,
				PlanID:            instancesFlags.planID,
			})
			if err != nil {
				log.Fatalf("Error deleting instance %s in broker %s: %v", instancesFlags.instanceID, brokerURL, err)
			}

			if !res.Async {
				fmt.Printf("Successfully deleted the instance %s: %+v\n", instancesFlags.instanceID, *res)
				return
			}

			if !instancesFlags.wait {
				fmt.Printf("Successfully started the operation to delete instance %s: %+v\n", instancesFlags.instanceID, *res)
				return
			}

			op, err := waitOnOperation(pollInstanceOpFunc(client, instancesFlags.apiVersion, brokerURL, instancesFlags.instanceID, instancesFlags.serviceID,
				instancesFlags.planID, res.OperationID, adapter.OperationDelete), false)
			if err != nil {
				log.Fatalf("Error polling last operation %q for instance %s: %v", instancesFlags.operationID, instancesFlags.instanceID, err)
			}

			if op.State == adapter.OperationSucceeded {
				fmt.Printf("Successfully deleted the instance %s asynchronously (operation %q): %+v\n", instancesFlags.instanceID, res.OperationID, *op)
				return
			}

			log.Fatalf("Failed deleting instance %s asynchronously (operation %q): %+v\n", instancesFlags.instanceID, res.OperationID, *op)
		},
	}

	instancesUpdateCmd = &cobra.Command{
		Use:   "update",
		Short: "Update a service instance",
		Long:  "Update a service instance",
		Run: func(cmd *cobra.Command, args []string) {
			flags.CheckFlags(&instancesFlags.instanceID, &instancesFlags.serviceID)

			client := httpAdapterFromFlag()
			brokerURL, err := instancesFlags.BrokerURL()
			if err != nil {
				log.Fatalf("Error updating instance %s: %v", instancesFlags.instanceID, err)
			}
			res, err := client.UpdateInstance(&adapter.UpdateInstanceParams{
				APIVersion:             instancesFlags.apiVersion,
				Server:                 brokerURL,
				AcceptsIncomplete:      instancesFlags.acceptsIncomplete,
				InstanceID:             instancesFlags.instanceID,
				ServiceID:              instancesFlags.serviceID,
				PlanID:                 instancesFlags.planID,
				Context:                parseStringToObjectMap(instancesFlags.context),
				Parameters:             parseStringToObjectMap(instancesFlags.parameters),
				PreviousServiceID:      instancesFlags.previousServiceID,
				PreviousPlanID:         instancesFlags.previousPlanID,
				PreviousOrganizationID: instancesFlags.previousOrganizationID,
				PreviousSpaceID:        instancesFlags.previousSpaceID,
			})
			if err != nil {
				log.Fatalf("Error updating instance %s in broker %s: %v", instancesFlags.instanceID, brokerURL, err)
			}

			if !res.Async {
				fmt.Printf("Successfully updated the instance %s: %+v\n", instancesFlags.instanceID, *res)
				return
			}

			if !instancesFlags.wait {
				fmt.Printf("Successfully started the operation to update instance %s: %+v\n", instancesFlags.instanceID, *res)
				return
			}

			op, err := waitOnOperation(pollInstanceOpFunc(client, instancesFlags.apiVersion, brokerURL, instancesFlags.instanceID, instancesFlags.serviceID, instancesFlags.planID, res.OperationID, adapter.OperationUpdate), false)
			if err != nil {
				log.Fatalf("Error polling last operation %q for instance %s: %v", res.OperationID, instancesFlags.instanceID, err)
			}

			if op.State == adapter.OperationSucceeded {
				fmt.Printf("Successfully updated the instance %s asynchronously (operation %q): %+v\n", instancesFlags.instanceID, res.OperationID, *op)
				return
			}

			log.Fatalf("Failed updating instance %s asynchronously (operation %q): %+v\n", instancesFlags.instanceID, res.OperationID, *op)
		},
	}

	instancesPollCmd = &cobra.Command{
		Use:   "poll",
		Short: "Poll the operation for the service instance",
		Long:  "Poll the operation for the service instance",
		Run: func(cmd *cobra.Command, args []string) {
			flags.CheckFlags(&instancesFlags.instanceID)

			client := httpAdapterFromFlag()
			brokerURL, err := instancesFlags.BrokerURL()
			if err != nil {
				log.Fatalf("Error polling operation %s for instance %s: %v", instancesFlags.operationID, instancesFlags.instanceID, err)
			}
			pollInstanceOp := pollInstanceOpFunc(client, instancesFlags.apiVersion, brokerURL, instancesFlags.instanceID, instancesFlags.serviceID, instancesFlags.planID, instancesFlags.operationID, adapter.OperationUnknown)
			op, err := pollInstanceOp()
			if err != nil {
				log.Fatalf("Error polling operation %q for instance %s in broker %s: %v", instancesFlags.operationID, instancesFlags.instanceID, brokerURL, err)
			}

			fmt.Printf("Successfully polled the operation %q for instance %s in broker %s: %+v\n", instancesFlags.operationID, instancesFlags.instanceID, brokerURL, *op)
		},
	}
)

func init() {
	// Flags for `instances` command group and all subgroups.
	flags.StringFlag(instancesCmd.PersistentFlags(), &instancesFlags.Server, flags.ServerLongName, flags.ServerShortName,
		fmt.Sprintf("[Required if %s and %s are not given] Broker URL to make request to (https://...).", flags.ProjectLongName, flags.BrokerLongName))
	flags.StringFlagWithDefault(instancesCmd.PersistentFlags(), &instancesFlags.apiVersion,
		flags.ApiVersionLongName, flags.ApiVersionShortName, flags.ApiVersionDefault, flags.ApiVersionDescription)
	flags.StringFlag(instancesCmd.PersistentFlags(), &instancesFlags.Project, flags.ProjectLongName, flags.ProjectShortName,
		fmt.Sprintf("[Required if %s is not given] the GCP project of the broker", flags.ServerLongName))
	flags.StringFlag(instancesCmd.PersistentFlags(), &instancesFlags.Broker, flags.BrokerLongName, flags.BrokerShortName,
		fmt.Sprintf("[Required if %s is not given] the broker name", flags.ServerLongName))
	instancesCmd.PersistentFlags().StringVar(&instancesFlags.Host, flags.HostLongName, flags.HostBrokerDefault, "")
	instancesCmd.PersistentFlags().MarkHidden(flags.HostLongName)

	// Flags for `instances create` command group.
	// Flags with no short names won't show up in the help message so every flag has a unique but
	// weird short name.
	flags.StringFlag(instancesCreateCmd.PersistentFlags(), &instancesFlags.instanceID, "instance", "i",
		"[Required] Service instance ID.")
	flags.BoolFlag(instancesCreateCmd.PersistentFlags(), &instancesFlags.acceptsIncomplete, "asynchronous", "a",
		"[Optional] If specified, the broker will execute the request asynchronously. (Default: FALSE)")
	flags.BoolFlag(instancesCreateCmd.PersistentFlags(), &instancesFlags.wait, "wait", "w",
		"[Optional] If specified, the broker will keep polling the last operation when the broker "+
			"is executing the operation asynchronously. (Default: FALSE)")
	flags.StringFlag(instancesCreateCmd.PersistentFlags(), &instancesFlags.serviceID, "service", "r",
		"[Required] The service ID used to create the service instance.")
	flags.StringFlag(instancesCreateCmd.PersistentFlags(), &instancesFlags.planID, "plan", "l",
		"[Required] The plan ID used to create the service instance.")
	flags.StringFlag(instancesCreateCmd.PersistentFlags(), &instancesFlags.context, "context", "t",
		"[Optional] [JSON Object] Platform specific contextual information under which the service "+
			"instance is to be provisioned.")
	flags.StringFlag(instancesCreateCmd.PersistentFlags(), &instancesFlags.organizationGUID, "organization", "o",
		"[Optional] [Deprecated in favor of 'Context'] The platform GUID for the organization under"+
			" which the service instance is to be provisioned.")
	flags.StringFlag(instancesCreateCmd.PersistentFlags(), &instancesFlags.spaceGUID, "space", "e",
		"[Optional] [Deprecated in favor of 'Context'] The identifier for the project space within "+
			"the platform organization.")
	flags.StringFlag(instancesCreateCmd.PersistentFlags(), &instancesFlags.parameters, "parameters", "m",
		"[Optional] [JSON Object] Configuration options for the service instance.")

	// Flags for `instances delete` command group.
	flags.StringFlag(instancesDeleteCmd.PersistentFlags(), &instancesFlags.instanceID, "instance", "i",
		"[Required] Service instance ID.")
	flags.BoolFlag(instancesDeleteCmd.PersistentFlags(), &instancesFlags.acceptsIncomplete, "asynchronous", "a",
		"[Optional] If specified, the broker will execute the request asynchronously. (Default: FALSE)")
	flags.BoolFlag(instancesDeleteCmd.PersistentFlags(), &instancesFlags.wait, "wait", "w",
		"[Optional] If specified, the broker will keep polling the last operation when the broker "+
			"is executing the operation asynchronously. (Default: FALSE)")
	flags.StringFlag(instancesDeleteCmd.PersistentFlags(), &instancesFlags.serviceID, "service", "r",
		"[Required] The service ID used by the service instance.")
	flags.StringFlag(instancesDeleteCmd.PersistentFlags(), &instancesFlags.planID, "plan", "l",
		"[Required] The plan ID used by the service instance.")

	// Flags for `instances update` command group.
	flags.StringFlag(instancesUpdateCmd.PersistentFlags(), &instancesFlags.instanceID, "instance", "i",
		"[Required] Service instance ID.")
	flags.BoolFlag(instancesUpdateCmd.PersistentFlags(), &instancesFlags.acceptsIncomplete, "asynchronous", "a",
		"[Optional] If specified, the broker will execute the request asynchronously. (Default: FALSE)")
	flags.BoolFlag(instancesUpdateCmd.PersistentFlags(), &instancesFlags.wait, "wait", "w",
		"[Optional] If specified, the broker will keep polling the last operation when the broker "+
			"is executing the operation asynchronously. (Default: FALSE)")
	flags.StringFlag(instancesUpdateCmd.PersistentFlags(), &instancesFlags.serviceID, "service", "r",
		"[Required] The service ID used by the service instance.")
	flags.StringFlag(instancesUpdateCmd.PersistentFlags(), &instancesFlags.planID, "plan", "l",
		"[Optional] The plan ID used by the service instance.")
	flags.StringFlag(instancesUpdateCmd.PersistentFlags(), &instancesFlags.context, "context", "t",
		"[Optional] [JSON Object] Platform specific contextual information under which the service "+
			"instance is provisioned.")
	flags.StringFlag(instancesUpdateCmd.PersistentFlags(), &instancesFlags.parameters, "parameters", "m",
		"[Optional] [JSON Object] Configuration options for the service instance.")
	flags.StringFlag(instancesUpdateCmd.PersistentFlags(), &instancesFlags.previousServiceID, "oldservice", "f",
		"[Optional] [Deprecated because it is immutable] The service ID used by the service instance.")
	flags.StringFlag(instancesUpdateCmd.PersistentFlags(), &instancesFlags.previousPlanID, "oldplan", "n",
		"[Optional] The plan ID used by the service instance prior to the update.")
	flags.StringFlag(instancesUpdateCmd.PersistentFlags(), &instancesFlags.previousOrganizationID, "oldorganization", "o",
		"[Optional] [Deprecated in favor of 'Context'] ID of the organization specified for the service instance.")
	flags.StringFlag(instancesUpdateCmd.PersistentFlags(), &instancesFlags.previousSpaceID, "oldspace", "e",
		"[Optional] [Deprecated in favor of 'Context'] ID of the space specified for the service instance.")

	// Flags for `instances poll` command group.
	flags.StringFlag(instancesPollCmd.PersistentFlags(), &instancesFlags.instanceID, "instance", "i",
		"[Required] Service instance ID.")
	flags.StringFlag(instancesPollCmd.PersistentFlags(), &instancesFlags.serviceID, "service", "r",
		"[Optional] The service ID used to create the service instance. If present, must not be an empty string.")
	flags.StringFlag(instancesPollCmd.PersistentFlags(), &instancesFlags.planID, "plan", "l",
		"[Optional] The plan ID used to create the service instance. If present, must not be an empty string.")
	flags.StringFlag(instancesPollCmd.PersistentFlags(), &instancesFlags.operationID, "operation", "o",
		"[Optional] The operation ID used to poll the operation for the service instance. If present, must not be an empty string.")

	RootCmd.AddCommand(instancesCmd)
	instancesCmd.AddCommand(instancesCreateCmd)
	instancesCmd.AddCommand(instancesListCmd)
	instancesCmd.AddCommand(instancesDeleteCmd)
	instancesCmd.AddCommand(instancesUpdateCmd)
	instancesCmd.AddCommand(instancesPollCmd)
}

func pollInstanceOpFunc(client adapter.Adapter, apiVersion, brokerURL, instanceID, serviceID, planID, opID string, opType adapter.OperationType) func() (*adapter.Operation, error) {
	cb := func() (*adapter.Operation, error) {
		return client.InstanceLastOperation(&adapter.InstanceLastOperationParams{
			Server:     brokerURL,
			InstanceID: instanceID,
			LastOperationParams: &adapter.LastOperationParams{
				APIVersion:    apiVersion,
				ServiceID:     serviceID,
				PlanID:        planID,
				OperationID:   opID,
				OperationType: opType,
			},
		})
	}
	return cb
}

func listInstances(brokerURL string) (*listInstancesResult, error) {
	client := httpAdapterFromFlag()
	lir, err := client.ListInstances(&adapter.ListInstancesParams{Server: brokerURL})
	if err != nil {
		return nil, err
	}

	result := &listInstancesResult{}
	var instances []*Instance
	for _, i := range lir.Instances {
		lbr, err := client.ListBindings(&adapter.ListBindingsParams{
			Server:     brokerURL,
			InstanceID: i.ID,
		})
		if err != nil {
			return nil, err
		}
		instances = append(instances, &Instance{
			ID:            i.ID,
			ServiceID:     i.ServiceID,
			PlanID:        i.PlanID,
			CreateTime:    i.CreateTime,
			NumOfBindings: len(lbr.Bindings),
		})
	}
	result.Instances = instances
	return result, nil
}

func deleteInstance(client adapter.Adapter, apiVersion, brokerURL string, i *osb.Instance, showProgress bool) error {
	if showProgress {
		fmt.Printf("Deleting instance %q in broker %q\n", i.ID, brokerURL)
	}

	res, err := client.DeleteInstance(&adapter.DeleteInstanceParams{
		Server:            brokerURL,
		InstanceID:        i.ID,
		ServiceID:         i.ServiceID,
		PlanID:            i.PlanID,
		AcceptsIncomplete: true,
		APIVersion:        apiVersion,
	})
	if err != nil {
		return err
	}

	op, err := waitOnOperation(pollInstanceOpFunc(client, apiVersion, brokerURL, i.ID, i.ServiceID, i.PlanID, res.OperationID, adapter.OperationDelete), showProgress)
	if err != nil {
		return fmt.Errorf("Error polling last operation %q for instance %q in broker %q: %v", res.OperationID, i.ID, brokerURL, err)
	}

	if op.State == adapter.OperationSucceeded {
		if showProgress {
			fmt.Print("Done\n")
		}
		return nil
	}

	return fmt.Errorf("Failed to delete instance %q in broker %q: %+v", i.ID, brokerURL, *op)
}

func printListInstances(result *listInstancesResult) {
	for index, i := range result.Instances {
		fmt.Printf("%d. Instance ID: %s\n", index+1, i.ID)
		fmt.Printf("   Service ID: %s, Plan ID: %s\n", i.ServiceID, i.PlanID)
		fmt.Printf("   Number of bindings: %d\n\n", i.NumOfBindings)
	}
}
