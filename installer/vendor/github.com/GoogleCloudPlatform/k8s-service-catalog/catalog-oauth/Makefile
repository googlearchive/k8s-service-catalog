IMAGE ?= google-auth-manager
BUILDER_IMAGE ?= google_auth_builder

build:
	# build binary and put in ./bin
	docker build . -t $(BUILDER_IMAGE) -f docker/build/Dockerfile
	docker run -v $(PWD)/bin:/go/src/plori/catalog-oauth/bin $(BUILDER_IMAGE)
	# build docker image with binary
	docker build . -t $(IMAGE) -f docker/catalog-oauth/Dockerfile
