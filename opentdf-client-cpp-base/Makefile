CLIENT_CPP_SDK_VER?=0.5.2-dcr
CLIENT_CPP_BRANCH?=0.5.2
BASE_IMG_TAG?='opentdf/client-cpp-base'
BUILDX_NAME='opentdf-client-cpp-base'
PLATFORMS='linux/amd64,linux/arm64'

# Destroy any previously-created buildx context
.PHONY: docker-buildx-armclean
docker-buildx-armclean:
	docker buildx use default
	docker buildx rm $(BUILDX_NAME) || true

.PHONY: repo-clone
repo-clone:
	@echo "Cloning opentdf/client-cpp repo branch $(CLIENT_CPP_BRANCH)"
	@rm -rf client-cpp
	@git clone -b $(CLIENT_CPP_BRANCH) 'git@github.com:opentdf/client-cpp.git'


# Set up a custom buildx context that supports building a multi-arch image
.PHONY: docker-buildx-armsetup
docker-buildx-armsetup: docker-buildx-armclean
	docker buildx create --name $(BUILDX_NAME)
	docker buildx use $(BUILDX_NAME)
	docker buildx inspect --bootstrap

.PHONY: dockerbuild
dockerbuild: repo-clone docker-buildx-armsetup
	@echo "Building '$(BASE_IMG_TAG):$(CLIENT_CPP_SDK_VER)' Docker image"
	@DOCKER_BUILDKIT=1 docker buildx build --platform ${PLATFORMS} --ssh default -t $(BASE_IMG_TAG):$(CLIENT_CPP_SDK_VER) .

.PHONY: dockerbuildpublish
dockerbuildpush: dockerbuild
	@echo "Publishing '$(BASE_IMG_TAG):$(CLIENT_CPP_SDK_VER)' to Dockerhub"
	@DOCKER_BUILDKIT=1 docker buildx build --platform ${PLATFORMS} --ssh default -t $(BASE_IMG_TAG):$(CLIENT_CPP_SDK_VER) . --push

#List targets in makefile
.PHONY: list
list:
	@$(MAKE) -pRrq -f $(lastword $(MAKEFILE_LIST)) : 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | sort | egrep -v -e '^[^[:alnum:]]' -e '^$@$$'
