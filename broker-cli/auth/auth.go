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
// Package auth handles authentication.

package auth

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"

	"golang.org/x/oauth2/google"
)

const scope = "https://www.googleapis.com/auth/cloud-platform"

// HttpClientFromFile returns an http client which is configured to use the
// service account credentials from credsFile.
func HttpClientFromFile(ctx context.Context, credsFile string) (*http.Client, error) {
	jsonKey, err := ioutil.ReadFile(credsFile)
	if err != nil {
		return nil, fmt.Errorf("error getting json key: %v", err)
	}

	config, err := google.JWTConfigFromJSON(jsonKey, scope)
	if err != nil {
		return nil, fmt.Errorf("error getting google jwt config: %v", err)
	}

	return config.Client(ctx), nil
}

// HttpClientWithDefaultCredentials returns an http client which will use the gcloud
// credentials of the currently logged in user.
func HttpClientWithDefaultCredentials(ctx context.Context) (*http.Client, error) {
	client, err := google.DefaultClient(ctx, scope)
	if err != nil {
		return nil, fmt.Errorf("error getting google default client: %v", err)
	}
	return client, nil
}
