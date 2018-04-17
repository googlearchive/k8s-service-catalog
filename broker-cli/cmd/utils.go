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
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"time"

	"github.com/GoogleCloudPlatform/k8s-service-catalog/broker-cli/auth"
	"github.com/GoogleCloudPlatform/k8s-service-catalog/broker-cli/client/adapter"
)

// httpAdapterFromFlag returns an http adapter with credentials to gcloud if
// credsFlag is not set and to a service account if it is set.
func httpAdapterFromFlag() adapter.Adapter {
	var client *http.Client
	var err error
	ctx := context.Background()
	if credsFlag != "" {
		client, err = auth.HttpClientFromFile(ctx, credsFlag)
		if err != nil {
			log.Fatalf("Error creating http client from service account file %s: %v", credsFlag, err)
		}
	} else {
		client, err = auth.HttpClientWithDefaultCredentials(ctx)
		if err != nil {
			log.Fatalf("Error creating http client using gcloud credentials: %v", err)
		}
	}
	return adapter.NewHttpAdapter(client)
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

func waitOnOperation(pollOperation func() (*adapter.Operation, error), showProgress bool) (*adapter.Operation, error) {
	baseDelay := 100 * time.Millisecond
	maxDelay := 6 * time.Second
	curState := adapter.OperationInProgress
	delay := baseDelay

	var op *adapter.Operation
	var err error
	for curState == adapter.OperationInProgress {
		if showProgress {
			fmt.Print(".")
		}
		time.Sleep(delay)
		op, err = pollOperation()
		if err != nil {
			return nil, err
		}

		if delay < maxDelay {
			delay = time.Duration(math.Min(float64(maxDelay), 2*float64(delay)))
		}
		curState = op.State
	}

	// Operation states other than "in progress" are all considered as end states.
	return op, nil
}
