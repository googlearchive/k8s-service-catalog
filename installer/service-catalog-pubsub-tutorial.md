# Service Catalog Tutorial


In this tutorial, we will walkthrough the steps required to connect [a sample PubSub application](https://github.com/apelisse/sc-pubsub) to GCP pubsub service using Service Catalog. This tutorial assumes that you have installed Service Catalog in your Kubernetes cluster and added a GCP Service Broker in it.

In this tutorial, we will accomplish the following tasks:

- Discover all the Service Brokers and their statuses

- Discover available services and their plans through Service Catalog

- Provision a GCP PubSub Topic using ServiceInstance resource in Service Catalog

- Create a Binding with `role/publisher` role for the Provisioned ServiceInstance in previous step

- Consume results of the Binding in an App which publishes to the PubSub topic.

## Discover ServiceBrokers


Before consuming GCP services using service catalog, lets ensure we have a GCP broker added and ready.

```bash
 # list all the service brokers with their status
 kubectl get servicebrokers -o custom-columns=BROKER:.metadata.name,STATUS:.status.conditions[0].type
 BROKER       STATUS
 gcp-broker   Ready

 # To print service brokers with details, run following command
 kubectl get servicebrokers -o yaml

```

## Discover services


Once you add a GCP Service Broker, Service Catalog will fetch the details of GCP services from GCP broker. We can query Service Catalog to discover available services and their plans as shown below.

```bash
 # list all serviceclasses with their plans
 kubectl get serviceclass -o custom-columns=BROKER:.brokerName,SERVICE:.metadata.name,PLANS:.plans[].name
 BROKER       SERVICE   PLANS
 gcp-broker   pubsub    pubsub-plan

 # To print serviceclasses with details, run following command
 kubectl get serviceclasses -o yaml

```

## Provisioning a service instance


Our sample app publishes messages on a PubSub topic, so we need to provision a topic first. Given below is an example config for provisioning
PubSub topic. Source code for the sample app can be found [at](https://github.com/apelisse/sc-pubsub).

```YAML
apiVersion: servicecatalog.k8s.io/v1alpha1
kind: ServiceInstance
metadata:
  name: gcp-pubsub-instance
  namespace: gcp-pubsub-app
spec:
  # this should match with one of the service classes listed in the `discover services` step above.
  serviceClassName: pubsub
  # this should match with one of the plans listed in the `discover services` step above.
  planName: pubsub-plan
```

Follow the steps below to create namespace and create serviceinstance.

```bash
# create `gcp-pubsub-app` namespace first.
kubectl create -f examples/gcp-pubsub-app/namespace.yaml
namespace "gcp-pubsub-app" created

# create pubsub service instance which will provision a topic.
kubectl create -f examples/gcp-pubsub-app/service-instance.yaml
serviceinstance "gcp-pubsub-instance" created

# list service instance and its status
kubectl get serviceinstances -n gcp-pubsub-app -o custom-columns=NAME:.metadata.name,SERVICE:.spec.serviceClassName,PLAN:.spec.planName,STATUS:.status.conditions[0].type
NAME                  SERVICE   PLAN          STATUS
gcp-pubsub-instance   pubsub    pubsub-plan   Ready

# list service instances in details
kubectl get serviceinstances -n gcp-pubsub-app -o yaml

```

## Creating Service Binding


We need GCP service account credentials to consume GCP services. Assuming you have downloaded the JSON key for the GCP service account for the sample app, you can store the service account credentials in a secret in Kubernetes using following steps.

```bash
# creating secret for storing GCP Service Account key
kubectl create secret generic sa-key --from-file=key.json=<path/to/service-account-key.json> -n gcp-pubsub-app
secret "sa-key" created

```

In order to publish messages on the topic provisioned above, we need to create service binding. An example config for creating Service Binding is given below:

```YAML
apiVersion: servicecatalog.k8s.io/v1alpha1
kind: ServiceInstanceCredential
metadata:
  name: gcp-pubsub-binding
  namespace: gcp-pubsub-app
spec:
  instanceRef:
    name: gcp-pubsub-instance
  # Secret to store returned data from bind call
  # Currently:
  #   project: GCP project id
  #   serviceAccount: same as passed as parameter
  #   subscription: generated subscription name
  #   topic: generated topic name
  secretName: gcp-pubsub-credentials
  parameters:
    # GCP *app* service account
    serviceAccount: "bind-test@sunilarora-sandbox.iam.gserviceaccount.com"
    # publisher or subscriber
    roles: ["roles/pubsub.publisher"]
```

Follow the steps below to create binding and examine its status.

```bash
# create the service binding
kubectl create -f examples/gcp-pubsub-app/service-binding.yaml
serviceinstancecredential "gcp-pubsub-binding" created

# list the service binding with its status
kubectl get serviceinstancecredentials -n gcp-pubsub-app -o custom-columns=NAME:.metadata.name,SERVICE-INSTANCE:.spec.instanceRef.name,STATUS:.status.conditions[0].type,OUTPUT-SECRET:.spec.secretName
NAME                 SERVICE-INSTANCE      STATUS    OUTPUT-SECRET
gcp-pubsub-binding   gcp-pubsub-instance   Ready     gcp-pubsub-credentials

# list service bindings with full details, run following command.
kubectl get serviceinstancecredentials -n gcp-pubsub-app -o yaml

# examine result of the binding result that is stored in the secret we specific in the config (in this case gcp-pubsub-credentials)
kubectl get secret gcp-pubsub-credentials -n gcp-pubsub-app -o yaml
apiVersion: v1
data:
  project: c3VuaWxhcm9yYS1zYW5kYm94
  serviceAccount: YmluZC10ZXN0QHN1bmlsYXJvcmEtc2FuZGJveC5pYW0uZ3NlcnZpY2VhY2NvdW50LmNvbQ==
  topic: ejAyMjEyYTczLWI0NWYtNDU2Ny04ZDU1LWU5M2M3YWFhNTk4OS10b3BpYw==
kind: Secret
metadata:
  creationTimestamp: 2017-10-12T00:06:18Z
  name: gcp-pubsub-credentials
  namespace: gcp-pubsub-app
  ownerReferences:
  - apiVersion: servicecatalog.k8s.io/v1alpha1
    blockOwnerDeletion: true
    controller: true
    kind: ServiceInstanceCredential
    name: gcp-pubsub-binding
    uid: 1a45e5c9-aee1-11e7-addf-0a580a300166
  resourceVersion: "3479470"
  selfLink: /api/v1/namespaces/gcp-pubsub-app/secrets/gcp-pubsub-credentials
  uid: 2c287f23-aee1-11e7-9115-42010af0004d
type: Opaque

# decoding the topic name
echo -n "ejAyMjEyYTczLWI0NWYtNDU2Ny04ZDU1LWU5M2M3YWFhNTk4OS10b3BpYw=="|base64 -d
z02212a73-b45f-4567-8d55-e93c7aaa5989-topic
```

## Using Service Binding in PubSub app


Here is a deployment config for the PubSub app that consumes the Service Binding created in the previous step by using the Secret `gcp-pubsub-credentials` created as result of Service Binding.

```YAML
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: echo
  namespace: gcp-pubsub-app
spec:
  replicas: 1
  template:
    metadata:
      labels:
        app: echo
      name: echo
    spec:
      volumes:
        - name: google-cloud-key
          secret:
           secretName: sa-key
      containers:
        - name: echo
          image: "gcr.io/apelisse-test/echo"
          volumeMounts:
          - name: google-cloud-key
            mountPath: /var/secrets/google
          ports:
          - name: echo-port
            containerPort: 80
          env:
          - name: "PROJECT_ID"
            valueFrom:
                secretKeyRef:
                   name: gcp-pubsub-credentials
                   key: project
          - name: "TOPIC"
            valueFrom:
                secretKeyRef:
                   name: gcp-pubsub-credentials
                   key: topic
          - name: GOOGLE_APPLICATION_CREDENTIALS
            value: "/var/secrets/google/key.json"
```

Follow the steps below to deploy the sample app:

```bash
# creating secret for storing GCP Service Account key
kubectl create secret generic sa-key --from-file=key.json=<path/to/service-account-key.json> -n gcp-pubsub-app
secret "sa-key" created

# deploy the pubsub app
kubectl create -f examples/gcp-pubsub-app/echo.yaml
deployment "echo" created
service "echo" created

# check deployment for the example pubsub app echo
kubectl get deployment -n gcp-pubsub-app
NAME      DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
echo      1         1         1            1           30s

# check the service exposing echo webservice to publish messages
kubectl get services echo
NAME      TYPE        CLUSTER-IP     EXTERNAL-IP   PORT(S)   AGE
echo      ClusterIP   10.51.251.89   <none>        80/TCP    52s

```

## Testing the app


For testing the app, first we will create a subscription on the topic using `gcloud` commands so that we can read messages on that topic.
Then we will perform a simple test by publishing a message using the sample app and reading the message using `gcloud` command.
Here are the steps:

```bash
# creating subscription to the provisioned topic (following instructions in Service-Binding step to get the topic name)
gcloud beta pubsub subscriptions create echo --topic z02212a73-b45f-4567-8d55-e93c7aaa5989-topic
gcloud beta pubsub subscriptions pull echo --auto-ack


# enable port forwarding so that we can invoke the publish endpoint locally.
kubectl port-forward -n gcp-pubsub-app $(kubectl get pods -n gcp-pubsub-app -l app=echo -o jsonpath='{.items[0].metadata.name}') 8080:8080
Forwarding from 127.0.0.1:8080 -> 8080

#publish the message
curl http://127.0.0.1:8080/pubsub -d 'Hello GKE!'

# read the messages from the topic
gcloud beta pubsub subscriptions pull echo --auto-ack
┌────────────┬─────────────────┬────────────┐
│    DATA    │    MESSAGE_ID   │ ATTRIBUTES │
├────────────┼─────────────────┼────────────┤
│ Hello GKE! │ 159146582889146 │            │
└────────────┴─────────────────┴────────────┘

```
