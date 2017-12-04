# Starting a Kubernetes Cluster on GCE

[gcloud sdk]: https://cloud.google.com/sdk/downloads
[Create a new GCP project]: https://cloud.google.com/resource-manager/docs/creating-managing-projects
[billable]: https://support.google.com/cloud/answer/6158867?hl=en

## TL;DR

```
1) Create Project
gcloud projects create $PROJECT_ID

2) Enable billing
gcloud alpha billing projects link $PROJECT_ID --billing-account XXXXXX-XXXXXX-XXXXXX

3) Enable compute
gcloud alpha service-management enable container --project $PROJECT_ID
gcloud alpha service-management enable test-container.sandbox.googleapis.com --project $PROJECT_ID
gcloud alpha service-management enable compute-component.googleapis.com --project $PROJECT_ID

4) Start k8s cluster
$ cluster/kube-up.sh
```

## Detailed steps

* Have the [gcloud sdk] installed; you should be able to enter
  `gcloud version`.
* [Create a new GCP project] to avoid config collisions with
  existing projects. Make sure the project is [billable].

## Crucial environment variables

Set these as desired:

```
# Specify the case-sensitive Project ID, either
# an existing ID or a new one.
export PROJECT_ID=<some-project-name, e.g. binary-pumpkin-toast>

# Workspace used by this tutorial to create files.
TEMP_DIR=$(mktemp -d)
```

Optional, for visiting link that take shell vars as args:
```
# Handy to open URLs from command line.
BROWSER=/opt/google/chrome/chrome  # chromium-browser

# The GCP developer console
CONSOLE_URL=https://console.cloud.google.com
```

## Preliminaries

This section covers things a user would likely already have
done - e.g. choose a project, install gcloud, start a k8s
cluster, compile the service-catalog, etc.  This may inform an
integration test later.

This section takes around 5 minutes to perform, while the rest of
the (key generation, kubectl resource instantiation, etc.) takes
well under a minute.

### Login

```
gcloud config set account <someuser>@google.com
gcloud auth login
```

### Create a project

If not already created, do so:
```
gcloud projects create $PROJECT_ID
```

Point to it.
```
gcloud config set project $PROJECT_ID
gcloud config list
```

Verify `billingEnabled: true` for your project
(see [how to enable billing][billable].

```
gcloud alpha billing accounts list
gcloud alpha billing accounts projects describe $PROJECT_ID
```

### Start a Cluster

#### Install and Run Kubernetes

Before bringing up a cluster, one must enable these alpha features.

```
gcloud alpha service-management enable container --project $PROJECT_ID
gcloud alpha service-management enable test-container.sandbox.googleapis.com --project $PROJECT_ID
gcloud alpha service-management enable compute-component.googleapis.com --project $PROJECT_ID
```

Bring up a Google Computer Engine (GCE) cluster
using the latest official _k8s_ release.
Within .../k8s.io/kubernetes directory:
```
$ cluster/kube-up.sh
```

This should conclude with something like:

> ```
> Kubernetes master is running at FOO
> GLBCDefaultBackend is running at FOO/...
> Heapster is running at FOO/api/v1/...
> KubeDNS is running at FOO/api/v1/...
> ```

etc.

