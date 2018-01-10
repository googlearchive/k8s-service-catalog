[service catalog repo]: https://github.com/kubernetes-incubator/service-catalog
[service catalog developer guide]: https://github.com/kubernetes-incubator/service-catalog/blob/master/docs/devguide.md

# Build and Upload Service Catalog Binaries

Clone the [service catalog repo] and grab patches needed to build
the service catalog binaries. The current service catalog only supports basic
auth (user/password), so a patch is needed to implement Google's oAuth system.

```
TUT_DIR=${GOPATH}/src/github.com/kubernetes-incubator
cd $TUT_DIR
git clone git@github.com:kubernetes-incubator/service-catalog.git
cd service-catalog

git remote add richardfung \
    https://github.com/richardfung/service-catalog.git
git fetch richardfung
git checkout -b gcp richardfung/google
```

Define the `REGISTRY` environment variable for the benefit of the
makefile (see the [service catalog developer guide]), and build
the necessary images. In this example the PROJECT_ID is: **seans-sandbox**

```
# *** Don't forget trailing slash - make won't work without. ***
export REGISTRY=gcr.io/seans-sandbox/

# Clean first if PROJECT_ID has changed since last build.
# Some of the binaries created by
# docker in the build end up owned by root, requiring a chown.
cd $TUT_DIR/service-catalog
sudo chown -R $USER ./
make clean

cd $TUT_DIR/service-catalog
make build
```

To upload images, your project must have enabled the _Google
Container Registry API_ for billing.

There are lots of APIs:

```
gcloud service-management list --available --sort-by="NAME"
```

Enable the the (staged) Container Registry API, and the Compute Engine API:

```
gcloud service-management enable containerregistry.googleapis.com
```

Confirm

```
gcloud service-management list --enabled --filter='NAME:containerregistry*'
```

Before uploading, review your current list of images:

```
gcloud alpha container images list --repository=gcr.io/seans-sandbox
```

Upload your newly built images for apiserver, controller, etc.

```
# Authenticate with the docker runtime
gcloud docker -a

# Build and upload.
make images push
```

Verify the upload

```
gcloud alpha container images list --repository=gcr.io/seans-sandbox
```

Expect these:
> ```
> gcr.io/seans-sandbox/apiserver
> gcr.io/seans-sandbox/controller-manager
> gcr.io/seans-sandbox/user-broker
> ```
