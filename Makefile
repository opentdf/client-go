GO_WRAP_DOCKER_VER?=2.1.0
GO_WRAP_DOCKER_NAME?=opentdf/client-go

.PHONY: clean
clean:
	@echo "Removing vendored Go module folder"
	@rm -rf vendor

.PHONY: docker-build
dockerbuild: clean
	@echo "Vendoring Go module dependencies outside of the container, for convenience"
	@go mod vendor
	@echo "Building '$(GO_WRAP_DOCKER_NAME):$(GO_WRAP_DOCKER_VER)' Docker image"
	@docker build -t $(GO_WRAP_DOCKER_NAME):$(GO_WRAP_DOCKER_VER) .

.PHONY: docker-publish
dockerbuildpublish: docker-build
	@echo "Publishing '$(GO_WRAP_DOCKER_NAME):$(GO_WRAP_DOCKER_VER)' to Vitru Dockerhub"
	@docker push $(GO_WRAP_DOCKER_NAME):$(GO_WRAP_DOCKER_VER)

#List targets in makefile
.PHONY: list
list:
	@$(MAKE) -pRrq -f $(lastword $(MAKEFILE_LIST)) : 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | sort | egrep -v -e '^[^[:alnum:]]' -e '^$@$$'

