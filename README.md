# Using Google Cloud Platform Broker with Service Catalog

[cfssl]: https://github.com/cloudflare/cfssl
[service catalog walkthrough]: https://github.com/kubernetes-incubator/service-catalog/blob/master/docs/walkthrough.md
[GCloud project]: https://cloud.google.com/sdk/downloads
[GCP pubsub]: https://cloud.google.com/pubsub/

Full **ServiceCatalog** and **Google Cloud Platform Broker** instructions,
including how to launch the service catalog on a Kubernetes cluster, how to
connect to the GCP Broker, and steps necessary to consume GCP services.
Adapted from the [service catalog walkthrough].

## TL;DR

After bringing up a kubernetes cluster, and from within the service catalog base directory:

```
1) Bring up service catalog

$ ./service-catalog-up.sh

2) Connect to GCP Broker

$ ./gcp/gcp-broker-up.sh

```

## Preconditions

 * A [GCloud project]. Export the project name in the PROJECT_ID environment
 variable. Example: PROJECT_ID=seans-sandbox.
 * A Kubernetes cluster. Steps for running in [GCE](./gce-k8s-cluster.md).
 * The user running these scripts must have cluster-admin role for the
   Kubernetes cluster:

```
kubectl create clusterrolebinding cluster-admin-binding --clusterrole=cluster-admin --user=seans@google.com
```

 * [cfssl] tools are needed for TLS certificate creation and manipulation. These
   tools should be downloaded into ${GOPATH}/bin, and they should be visible in
   the ${PATH}:

```
$ go get -u github.com/cloudflare/cfssl/cmd/...

$ which cfssl

/usr/local/google/home/seans/go/bin/cfssl

$ which cfssljson

/usr/local/google/home/seans/go/bin/cfssljson
```

**TODO: Remove the cfssl dependency (probably using openssl)**

## Start the Service Catalog in the Kubernetes Cluster

Optional instructions for [building/pushing](./build-service-catalog.md) the service
catalog. NOTE: If want to bring up this newly built/pushed service-catalog
image, you need to change the *image:* tag in both **apiserver-deployment.yaml** and
**controller-manager-deployment.yaml**.

Within the service catalog base directory (ex: *.../github.com/kubernetes-incubator/service-catalog*):

```
$ ./service-catalog-up.sh
```

This script performs the following steps:

  * Creates a service catalog namespace (**namespace.yaml**)
  * Creates a Kubernetes service account to run service catalog (**service-accounts.yaml**)
  * Creates roles for the service account, and sets up appropriate bindings
    (**roles.yaml**)
  * Creates a service for the service catalog (**service.yaml**)
  * Registers an API (servicecatalog.k8s.io) with the main APIServer
    (**api-registration.yaml**)
  * Creates an etcd stateful set with a persistent volume (**etcd.yaml**,
    **etcd-svc.yaml**)
  * Creates a secret from the TLS certificates so the service catalog and the
  main APIServer can communicate securely (**tls-cert-secret.yaml**)
  * Bring up the service catalog's extension APIServer
    (**apiserver-deployment.yaml**)
  * Bring up the service catalog's controller manager (**controller-manager-deployment.yaml**)

Sanity checks after running script:

```
$ kubectl api-versions

```

Returns the API's, including:

```
...
servicecatalog.k8s.io/v1alpha1
...
```

Checking the pods in the service-catalog namespace should look like:

```
$ kubectl get pods -n service-catalog

NAME                                  READY     STATUS    RESTARTS   AGE
apiserver-423379232-hsjp7             1/1       Running   0          27s
controller-manager-2169645497-1phtf   1/1       Running   2          26s
etcd-0                                1/1       Running   0          28s
```

Finally, when checking for brokers, the user should see:

```
$ kubectl get servicebrokers

No resources found.
```

Which means the main APIServer understands the servicebrokers resource (but none were
found yet).

In order to cleanly take down the service catalog:

```
$ ./service-catalog-down.sh
```

## Connect to the GCP Broker

Within the service catalog base directory:

```
$ ./gcp/gcp-broker-up.sh
```

This script performs the following steps:

 * Enables the API's to talk to GCP Broker (if not already enabled)
 * Creates the service account to communicate with the GCP Broker (if not
   already created)
 * Generates the service account key, and stores it in a secret (**service-account-secret.yaml**)
 * Connects to the GCP Broker (**gcp-broker.yaml**)
 * Creates a namespace for future instance provisions and bindings
   (**gcp-instance-namespace.yaml**)

Sanity checks after running script:

```
$ kubectl get servicebrokers

NAME         KIND
gcp-broker   Broker.v1alpha1.servicecatalog.k8s.io

$ kubectl get servicebrokers gcp-broker -n service-catalog -o yaml

...

status:
  conditions:
  - lastTransitionTime: 2017-08-08T17:12:13Z
    message: Successfully fetched catalog entries from broker.
    reason: FetchedCatalog
    status: "True"
    type: Ready

$ kubect get serviceclasses

NAME            KIND                                          BINDABLE   BROKER NAME   DESCRIPTION
demo-logbook    ServiceClass.v1alpha1.servicecatalog.k8s.io   false      gcp-broker    demo logbook service
google-pubsub   ServiceClass.v1alpha1.servicecatalog.k8s.io   false      gcp-broker    A global service for real-time and reliable messaging and streaming data
```

If you see:

> ```
> ERROR: (gcloud.service-management.enable) You do not have permission to
> access service [staging-serviceregistry.sandbox.googleapis.com:enable]
> ```

then the GCP API is not available to you, and the most likely reason
is that you've not authenticated with a `@google.com` account. Confirm that the
apis are available:

```
gcloud service-management list --available | grep staging | egrep '(broker|registry)'
```

In order to cleanly take down the GCP broker:

```
$ ./gcp/gcp-broker-down.sh
```

## Consume GCP Services

### Provision a Broker Instance

In this example we provision an instance of [GCP pubsub]. The pubsub parameters
**subscription** and **topic** are specified in the file used to provision the
instance:

```
apiVersion: servicecatalog.k8s.io/v1alpha1
kind: Instance
metadata:
  name: gcp-pubsub-instance
  namespace: gcp-instances
spec:
  serviceClassName: google-pubsub
  planName: default
  parameters:
    resources:
    - name: my-topic
      type: pubsub.v1.topic
      properties:
        topic: my-topic
    - name: my-subscription
      type: pubsub.v1.subscription
      properties:
        subscription: my-subscription
        topic: "$(ref.my-topic.name)"
```


Within the service catalog base directory:

```
$ kubectl create -f config/gcp/gcp-pubsub-instance.yaml

instance "gcp-pubsub-instance" created
```

Sanity check after provisioning the instance:

```
$ kubectl get serviceinstances gcp-pubsub-instance -n gcp-instances -o yaml

...
status:
  asyncOpInProgress: false
  checksum: d09aec5c6a76d2f6360b62001caf3aa9b841258db8b078681e2d69ebe0765866
  conditions:
  - lastTransitionTime: 2017-08-09T17:58:29Z
    message: The instance was provisioned successfully
    reason: ProvisionedSuccessfully
    status: "True"
    type: Ready
  lastOperation: operation-1502301496797-55655d30fe749-374280bc-140db26b
```

### Create the binding

The binding returns the credentials into a secret (named
*gcp-pubsub-credentials*). These credentials are then used by the application
library during communication with GCP.

Within the service catalog base directory:

```
$ kubectl create -f config/gcp/gcp-pubsub-binding.yaml

binding "gcp-pubsub-binding" created
```

Sanity checks after binding:

```

# Do a fully qualified group.kind.version because of a namespace
# bug currently in service catalog.
$ kubectl get serviceinstancecredentials gcp-pubsub-binding -n gcp-instances -o yaml

...
status:
  checksum: 755978b28d2df2332fc6cca7e50a0f348116ab5decb4e657e7eba96c70a52bbd
  conditions:
  - lastTransitionTime: 2017-08-09T18:04:44Z
    message: Injected bind result
    reason: InjectedBindResult
    status: "True"
    type: Ready

```

Also, the credentials should have been populated in the secret *gcp-pubsub-credentials*

```

$ kubectl get secrets gcp-pubsub-credentials -n gcp-instances -o yaml

```

## TODO insert real pubsub example

I.e. run a k8s app that exploits GCP pubsub via these broker bindings.

## This is not an official Google product.

