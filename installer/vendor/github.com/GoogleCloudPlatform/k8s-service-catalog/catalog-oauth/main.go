/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

catalog-oauth manages the creation of Secrets that contain OAuth access
tokens to use with the Kubernetes Service Catalog.
To use it, you need to create another Secret which contains the json private
key among other things (see README.md) to generate the OAuth
access token
*/

package main

import (
	"context"
	"flag"
	"time"

	"plori/catalog-oauth/auth"
	"plori/catalog-oauth/watcher"

	"github.com/golang/glog"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {
	var namespace string
	var resyncInterval time.Duration
	var timeout time.Duration

	flag.StringVar(&namespace, "n", "google-oauth", "namespace secrets will live in")
	flag.DurationVar(&resyncInterval, "i", 10*time.Minute, "default resync interval. Note this must be shorter than access token expiration")
	flag.DurationVar(&timeout, "t", 3*time.Minute, "request timeout duration")
	flag.Parse()

	ctx, _ := context.WithTimeout(context.Background(), timeout)

	kubeConfig, err := rest.InClusterConfig()
	if err != nil {
		glog.Fatalf("unable to get in cluster config: %v", err)
	}

	klient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		glog.Fatalf("unable to generate clientset from config: %v", err)
	}

	checkAndWriteToken := func(obj interface{}) {
		secret, ok := obj.(*v1.Secret)
		if !ok {
			glog.Error("obj in add function is not a secret")
		}
		if secret.Namespace != namespace {
			return
		}
		if err := auth.WriteTokenSecret(ctx, klient.CoreV1(), secret); err != nil {
			glog.Errorf("error writing token secret: %v", err)
		}
	}

	watcher := watcher.Watcher{}
	watcher.Watch(klient, resyncInterval, checkAndWriteToken,
		func(oldObj, newObj interface{}) {
			checkAndWriteToken(newObj)
		},
		nil)
}
