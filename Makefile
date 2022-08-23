IMAGEVERSION?=1.1.3
IMAGETAG?=opentdf/client-go
PLATFORMS?=linux/arm64,linux/amd64

.PHONY: clean
clean:
	@echo "Removing vendored Go module folder"
	@rm -rf vendor

# Set up a custom buildx context that supports building a multi-arch image
.PHONY: docker-buildx-armsetup
docker-buildx-armsetup:
    # Try to create builder context, ignoring failure if one already exists
	docker buildx create --name client-go-cross || true
	docker buildx use client-go-cross
	docker buildx inspect --bootstrap

# This will build (in parallel) Docker images for every arch in PLATFORMS
# using Docker's crossbuild environment: https://docs.docker.com/build/buildx/multiplatform-images/
.PHONY: dockerbuild
dockerbuild: clean docker-buildx-armsetup
	@echo "Building '$(IMAGETAG):$(IMAGEVERSION)' Docker image"
	@DOCKER_BUILDKIT=1 docker buildx build --platform $(PLATFORMS) -t $(IMAGETAG):$(IMAGEVERSION) .

# This will build AND PUSH (in parallel) Docker images for every arch in PLATFORMS
# using Docker's crossbuild environment: https://docs.docker.com/build/buildx/multiplatform-images/
.PHONY: dockerbuildpush
dockerbuildpush: clean docker-buildx-armsetup
	@echo "Publishing '$(IMAGETAG):$(IMAGEVERSION)' to Dockerhub"
	@DOCKER_BUILDKIT=1 docker buildx build --platform $(PLATFORMS) -t $(IMAGETAG):$(IMAGEVERSION) --push .

#List targets in makefile
.PHONY: list
list:
	@$(MAKE) -pRrq -f $(lastword $(MAKEFILE_LIST)) : 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | sort | egrep -v -e '^[^[:alnum:]]' -e '^$@$$'

