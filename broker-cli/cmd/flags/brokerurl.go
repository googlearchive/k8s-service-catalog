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

package flags

import (
	"fmt"
	"strings"
)

const (
	brokersKey  = "brokers"
	projectsKey = "projects"
	betaKey     = "v1beta1"
)

// BrokerURLConstructor is a struct that describes the fields which can be used to
// generate a broker URL via BrokerURL().
type BrokerURLConstructor struct {
	Broker  string
	Host    string
	Project string
	Server  string
}

func ConstructBrokerURL(host, project, broker string) string {
	return fmt.Sprintf("%s/v1beta1/projects/%s/brokers/%s", host, project, broker)
}

func (flags *BrokerURLConstructor) validateServer() error {
	if !strings.HasPrefix(flags.Server, flags.Host) {
		return fmt.Errorf("broker server URL %q should always begin with service end point %q", flags.Server, flags.Host)
	}

	parts := strings.Split(flags.Server, "/")
	partsLen := len(parts)

	if partsLen < 8 || parts[partsLen-2] != brokersKey || parts[partsLen-4] != projectsKey || parts[partsLen-5] != betaKey {
		return fmt.Errorf("broker server URL %q is invalid", flags.Server)
	}
	flags.Broker = parts[partsLen-1]
	flags.Project = parts[partsLen-3]
	return nil
}

// There are two available options for service broker commands. Users are
// allowed to pass either --server or (--project and --broker). If the latter
// is used then we generate the URL assuming we are using a GCP broker.
// BrokerURL checks that only one of the two options is passed
// in, that is, only either (--server) or (--project and --broker) are used,
// and returns the generated broker URL.
func (flags *BrokerURLConstructor) BrokerURL() (string, error) {
	if flags.Server == "" && (flags.Project == "" || flags.Broker == "") {
		fmt.Printf("Either the value of %s(= %q) or the values of %s(= %q) and %s(= %q) must be specified\n", ServerLongName, flags.Server, ProjectLongName, flags.Project, BrokerLongName, flags.Broker)
		return "", fmt.Errorf("Either the value of %s(= %q) or the values of %s(= %q) and %s(= %q) must be specified\n", ServerLongName, flags.Server, ProjectLongName, flags.Project, BrokerLongName, flags.Broker)
	}
	if flags.Server != "" && flags.Project != "" && flags.Broker != "" {
		fmt.Printf("Either the value of %s(= %q) or the values of %s(= %q) and %s(= %q) needs to be specified, not both\n", ServerLongName, flags.Server, ProjectLongName, flags.Project, BrokerLongName, flags.Broker)
		return "", fmt.Errorf("Either the value of %s(= %q) or the values of %s(= %q) and %s(= %q) needs to be specified, not both\n", ServerLongName, flags.Server, ProjectLongName, flags.Project, BrokerLongName, flags.Broker)
	}
	if flags.Server != "" {
		if err := flags.validateServer(); err != nil {
			return "", err
		}
		return flags.Server, nil
	}
	return ConstructBrokerURL(flags.Host, flags.Project, flags.Broker), nil
}
