# Kubernetes Cluster Development for GKE

[Detailed Instructions]: https://g3doc.corp.google.com/cloud/kubernetes/g3doc/dev/dev_project_setup.md?cl=head
[Detailed Instructions for]: https://g3doc.corp.google.com/cloud/kubernetes/g3doc/dev/sandbox.md?cl=head

## TL;DR


```
1) Start Google GKE binaries in borg. Within google3 directory of piper client:

$ sandman cloud/kubernetes/engprod/gke_sandbox.gcl TearDown Build SetUp Start --vars gke_zonal.hosted_master_project=$HOSTED_MASTER_PROJECT

2) Build/Push kubernetes containers. Within kubernetes base directory:

$ go run hack/e2e.go -- -v --build=quick --stage=gs://kubernetes-release-dev/ci --stage-suffix=$USER

3) Start the kubernetes cluster. Within google3 directory:

$ gcloud container clusters create test1 --zone=us-central1-c --cluster-version=1.7.0-${USER}
```

## Overview

One can think of Kubernetes development on GKE as having three layers. First, there is the
Google specific binaries running on borg to administer a Kubernetes cluster. These binaries
include (confusingly) Google API Server, mixer, and a couple others. The Kubernetes
cluster is divided into two parts: the control plane (running in the "Hosted Master" gcloud
project), and the user nodes (running in a regular "GKE Dev" gcloud project).

## Preliminaries

* Create a piper client for Google's GKE binaries.
* Create a "GKE Dev" gcloud project to organize the Kubernetes user nodes.
* Create a "Hosted Master" gcloud project to organize the Kubernetes control plane.

### Create a google3 Client

Create a google3 client to build/push/run the Google-specific binaries needed to run
a GKE cluster.

```
$ g4 citc <PROJECT-NAME>
$ cd /google/src/cloud/<LDAP>/<PROJECT-NAME>/google3
```

**(Optional)** Build/test the source to ensure the client works

```
$ g4 sync
$ blaze test cloud/kubernetes/...
```

### Create the GKE Dev and Hosted Master Projects

[Detailed Instructions] for creating these two gcloud projects. The main steps are run within the piper client google3 directory:


```
$ export PROJECT_ID=seans-gke-dev
$ blaze run cloud/kubernetes/config/projects/create:create_projects -- --project=$PROJECT_ID --project_type="DEV" --dry_run=true
$ export HOSTED_MASTER_PROJECT=gke-seans-hosted-master
$ blaze run //cloud/kubernetes/config/projects/create:create_project --  --project=$HOSTED_MASTER_PROJECT  --project_type="DEV_HOSTED_MASTER"  --dry_run=false
```

## Start the Google GKE Binaries

[Detailed Instructions for] creating a sandbox for GKE binaries.

Use sandman to create a sandbox for Google GKE binaries to run in borg. You may need to add the sandman binary to your path.
It lives at: 

```
/google/data/ro/projects/sandman/sandman
```

Invoke sandman:

```
$ sandman cloud/kubernetes/engprod/gke_sandbox.gcl TearDown Build SetUp Start --vars gke_zonal.hosted_master_project=$HOSTED_MASTER_PROJECT
```

This script will initiate the following step:

* TearDown - Takes down your current sandbox in borg. If it doesn't exist, it does nothing.
* Build - blaze build the necessary GKE targets.
* SetUp - Run the SandMan daemon, and create an instance of the spanner database.
* Start - Borg start the necessary GKE binaries (GKE APIServer, mixer, etc.)

The end of the process will look like:

```
Finished Start on Borg service using config file seans-gke-sandbox_lease_service.borg [0:01:00]
Waiting on Start on Borg service using config file seans-gke-sandbox_mixer.borg [0:01:30]
Waiting on Start on Borg service using config file seans-gke-sandbox_us-central1-c_apiserver.borg [0:01:30]
Waiting on Start on Borg service using config file seans-gke-sandbox_mixer.borg [0:02:00]
Waiting on Start on Borg service using config file seans-gke-sandbox_us-central1-c_apiserver.borg [0:02:00]
Finished Start on Borg service using config file seans-gke-sandbox_us-central1-c_apiserver.borg [0:02:13]
Finished Start on Borg service using config file seans-gke-sandbox_mixer.borg [0:02:19]
View your sandbox at https://sandman.corp.google.com/i/seans-gke-sandbox
```

View Sandman UI: **https://sandman.corp.google.com/i/${USER}-gke-sandbox**
Within the SandMan UI, Click on the "Search Sigma" button to view the jobs on borg.

## Build/Push the Kubernetes Artifacts

Build the kubernetes binaries, create the containers, and push them to the **gcr.io** repository.

The Kubernetes base directory is: .../src/k8s.io/kubernetes

From within the kubernetes base directory run:
```
$ go run hack/e2e.go -- -v --build=quick --stage=gs://kubernetes-release-dev/ci --stage-suffix=$USER
```

The output will include:

```
push-build.sh::release::gcs::copy_release_artifacts(): /usr/local/google/home/seans/google-cloud-sdk/bin/gsutil ls -lhr gs://kubernetes-release-dev/ci/v1.7.0-seans
gs://kubernetes-release-dev/ci/v1.7.0-seans/:
 31.78 MiB  2017-08-18T18:17:58Z  gs://kubernetes-release-dev/ci/v1.7.0-seans/kubernetes-client-linux-amd64.tar.gz
      33 B  2017-08-18T18:17:54Z  gs://kubernetes-release-dev/ci/v1.7.0-seans/kubernetes-client-linux-amd64.tar.gz.md5
      41 B  2017-08-18T18:17:54Z  gs://kubernetes-release-dev/ci/v1.7.0-seans/kubernetes-client-linux-amd64.tar.gz.sha1
775.95 KiB  2017-08-18T18:17:55Z  gs://kubernetes-release-dev/ci/v1.7.0-seans/kubernetes-manifests.tar.gz
```

**IMPORTANT** This output indicates the cluster version is: v1.7.0-seans. This information is necessary to start
the cluster.

## Start the Kubernetes Cluster

Finally, start the Kubernetes cluster into the HOSTED_MASTER_PROJECT and the PROJECT_ID.

Make sure you're pointing to the correct endpoint:

```
export CLOUDSDK_API_ENDPOINT_OVERRIDES_CONTAINER=https://${USER}-gke-sandbox-test-container.sandbox.googleapis.com/
```

**IMPORTANT** You need to match the cluster version **WITHOUT** the preceding "v". So for a cluster version of:

*v1.7.0-seans*

the cluster version is: 

*1.7.0-seans*

Back in the piper client google3 directory:

```
$ gcloud container clusters create test1 --zone=us-central1-c --cluster-version=1.7.0-seans
```

If the cluster version image was found, the output should look like:

```
Creating cluster test1...done.
Created [https://seans-gke-sandbox-test-container.sandbox.googleapis.com/v1/projects/seans-gke-dev/zones/us-central1-c/clusters/test1].
kubeconfig entry generated for test1.
NAME   ZONE           MASTER_VERSION  MASTER_IP      MACHINE_TYPE   NODE_VERSION  NUM_NODES  STATUS
test1  us-central1-c  1.7.0-seans     35.193.82.179  n1-standard-1  1.7.0-seans   3          RUNNING
```

Check the kubernetes cluster has been created.

```
$ k get po --all-namespaces
NAMESPACE     NAME                                              READY     STATUS    RESTARTS   AGE
kube-system   event-exporter-v0.1.4-1771975458-cjcsh            2/2       Running   0          2m
kube-system   fluentd-gcp-v2.0-kjx1f                            2/2       Running   0          2m
kube-system   fluentd-gcp-v2.0-nz9hk                            2/2       Running   0          2m
kube-system   fluentd-gcp-v2.0-t3rgs                            2/2       Running   0          2m
kube-system   heapster-v1.4.0-3984216332-8h8df                  2/2       Running   0          1m
kube-system   kube-dns-3097350089-1rlj0                         3/3       Running   0          1m
kube-system   kube-dns-3097350089-s9ms9                         3/3       Running   0          2m
kube-system   kube-dns-autoscaler-244676396-ww7d6               1/1       Running   0          2m
kube-system   kube-proxy-gke-test1-default-pool-050d14b0-0bdw   1/1       Running   0          1m
kube-system   kube-proxy-gke-test1-default-pool-050d14b0-jgwz   1/1       Running   0          1m
kube-system   kube-proxy-gke-test1-default-pool-050d14b0-qnlq   1/1       Running   0          2m
kube-system   kubernetes-dashboard-1265873680-l05c8             1/1       Running   0          2m
kube-system   l7-default-backend-3623108927-h8kmp               1/1       Running   0          2m
```

*NOTE*: Only the pods in "user" space nodes (not the control plane in the HOSTED_MASTER_PROJECT) will be shown.
