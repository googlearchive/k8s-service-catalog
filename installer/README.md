# CLI for managing Service Catalog installation in Kubernetes Cluster

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
