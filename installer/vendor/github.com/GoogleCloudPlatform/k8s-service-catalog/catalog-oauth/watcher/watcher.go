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

Watcher is a higher level abstraction to Kubernetes informers. You can
provide functions which should be called on add, update, or delete.
Also, the update function will be called periodically because.. well
Kubernetes decides that there's an update even when there's nothing new
*/

package watcher

import (
	"time"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// Watcher is basically a wrapper around a Kubernetes informer that will let
// you watch secrets.
type Watcher struct {
	stop chan struct{}
}

// Watch calls addFunc, updateFunc, and deleteFunc, when a Secret is created,
// updated, or deleted respectively. updateFunc also will get called
// periodically with defaultResync time between calls.
func (watcher *Watcher) Watch(klient kubernetes.Interface, defaultResync time.Duration, addFunc func(interface{}), updateFunc func(interface{}, interface{}), deleteFunc func(interface{})) {
	// should only have one informer factory. If we ever have more than one watcher then we need to reuse
	informer := v1.New(informers.NewSharedInformerFactory(klient, defaultResync)).Secrets().Informer()
	informer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    addFunc,
			UpdateFunc: updateFunc,
			DeleteFunc: deleteFunc,
		})
	watcher.stop = make(chan struct{})
	informer.Run(watcher.stop)
}

// Stop stops watching for secrets
func (watcher *Watcher) Stop() {
	close(watcher.stop)
}
