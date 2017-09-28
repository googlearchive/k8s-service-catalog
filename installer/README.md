# CLI for managing Service Catalog installation in Kubernetes Cluster

## How to Install the installer

## Installer is written in Go and can be installed using `go get`
```
go get github.com/GoogleCloudPlatform/k8s-service-catalog/installer/cmd/
```

## Steps you need to do before running the installer

 ### Install the binaries needed to run the installer
	* Install Cfssl binaries.
	* Install gcloud installed and configured to talk to GCP.
	* Install kubectl and configured to connect to Kubernetes cluster you want
	  to install Service Catalog.

 ### Steps to configure Gcloud CLI
 	* Run `gcloud login` to authorize Gcloud CLI to access GCP services.
	* Run `gcloud auth application-default login`

## Steps to build the CLI
```
1) This project uses [Dep](https://github.com/golang/dep) for dependency management. Install `dep` using `go get`
go get -u github.com/golang/dep/cmd/dep

2) Install `go-bindata` using `go get`:
go get -u github.com/jteeuwen/go-bindata/...

3) Run `make build` to build the CLI. You should `sc` binary created in output/bin directory
```

## Steps for Running the installer
Assuming you have the CLI `sc` in your PATH.
```
	* To print usage instructions for CLI, run `sc --help`
	* To check if all dependencies are in PATH, run `sc check`
	* To install Service Catalog, run `sc install` 
	* To uninstall Service Catalog, run `sc uninstall`
```
