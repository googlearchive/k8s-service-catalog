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

package cmd

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"
)

func NewUpdateCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "update",
		Short: "updates Service Catalog components in Kubernetes cluster",
		Long:  "updates Service Catalog components in Kubernetes cluster",
		Args:  cobra.MinimumNArgs(1),
	}
	// add all update sub-commands
	c.AddCommand(
		newServiceCatalogUpdateCmd(),
		newAuthManagerUpdateCmd(),
	)
	return c
}

// scUpdateArgs contains Service Catalog update Arguments.
type scUpdateArgs struct {
	Version                string
	APIserverImage         string
	ControllermanagerImage string
}

func newServiceCatalogUpdateCmd() *cobra.Command {
	uargs := &scUpdateArgs{}
	c := &cobra.Command{
		Use: "service-catalog",
		Run: func(cmd *cobra.Command, args []string) {
			if err := updateServiceCatalog(uargs); err != nil {
				fmt.Printf("failed to update service catalog components: %v", err)
				return
			}
			fmt.Println("Service Catalog updated successfully.")
		},
	}
	c.Flags().StringVar(&uargs.Version, "version", "", "Service Catalog Version")
	c.Flags().StringVar(&uargs.APIserverImage, "apiserver.image", "", "APIServer Image")
	c.Flags().StringVar(&uargs.ControllermanagerImage, "controllermanager.image", "", "Controllermanager Image")
	return c
}

func updateServiceCatalog(args *scUpdateArgs) error {

	found, err := isServiceCatalogInstalled()
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("service catalog is not installed")
	}

	var apiServerImage, controllerManagerImage string

	if args.Version != "" {
		apiServerImage = "quay.io/kubernetes-service-catalog/apiserver:v" + args.Version
		controllerManagerImage = "quay.io/kubernetes-service-catalog/controller-manager:v" + args.Version
	} else {
		// user has specified image versions of API Server and ControllerManager individually
		apiServerImage = args.APIserverImage
		controllerManagerImage = args.ControllermanagerImage
	}

	if apiServerImage == "" || controllerManagerImage == "" {
		return fmt.Errorf("empty Image arguments for service-catalog components")
	}

	ns := "service-catalog"
	cmds := []*exec.Cmd{
		exec.Command("kubectl", "set", "image", "deployments/apiserver",
			"apiserver="+apiServerImage, "-n", ns),
		exec.Command("kubectl", "set", "image", "deployments/controller-manager",
			"controller-manager="+controllerManagerImage, "-n", ns),
	}

	// TODO(droot): Current implementation is not atomic. Figure out a way to do
	// it atomically or rollback in case of failure.
	for _, c := range cmds {
		if o, err := c.CombinedOutput(); err != nil {
			return fmt.Errorf("error updating service catalog :%v", string(o))
		}
	}
	return nil
}

func newAuthManagerUpdateCmd() *cobra.Command {
	uargs := &authManagerUpdateArgs{}
	c := &cobra.Command{
		Use: "auth-manager",
		Run: func(cmd *cobra.Command, args []string) {
			if err := updateAuthManager(uargs); err != nil {
				fmt.Printf("failed to update auth-manager :%v \n", err)
				return
			}
			fmt.Println("updated auth manager successfully.")
		},
	}
	c.Flags().StringVar(&uargs.Image, "authmanager.image", "", "AuthManager Image")
	return c
}

type authManagerUpdateArgs struct {
	Image string
}

func updateAuthManager(args *authManagerUpdateArgs) error {
	found, err := isServiceCatalogInstalled()
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("service catalog is not installed")
	}

	if args.Image == "" {
		return fmt.Errorf("empty image arguments for auth manager")
	}

	ns := "service-catalog"
	out, err := exec.Command("kubectl", "set", "image", "deployments/google-oauth",
		"catalog-oauth="+args.Image, "-n", ns).CombinedOutput()
	if err != nil {
		return fmt.Errorf("error updating auth manager :%v", string(out))
	}
	return nil
}
