VERSION ?= 0.2.1

COMMIT := $(shell git rev-parse --short HEAD)
DATE := $(shell date +%Y%m%d)
VERSION := $(VERSION)-dev.$(COMMIT)-$(DATE)

ORG ?= k8ssandra
IMAGE_TAG_BASE ?= $(ORG)/k8ssandra-client

# Image URL to use all building/pushing image targets
IMG ?= $(IMAGE_TAG_BASE):v$(VERSION)
IMG_LATEST ?= $(IMAGE_TAG_BASE):latest

SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: fmt vet lint ## Run tests
	go test -v ./...

.PHONY: lint
lint: golangci-lint ## Run golangci-lint against code
	$(GOLANGCI_LINT) run ./...

.PHONY: build
build: test ## Build kubectl-k8ssandra
	CGO_ENABLED=0 go build -o kubectl-k8ssandra cmd/kubectl-k8ssandra/main.go

.PHONY: docker-build
docker-build: ## Build k8ssandra-client docker image
	docker buildx build --build-arg VERSION=${VERSION} -t ${IMG_LATEST} -t ${IMG} . --load -f cmd/kubectl-k8ssandra/Dockerfile

.PHONY: kind-load
kind-load: ## Load k8ssandra-client:latest to kind
	kind load docker-image ${IMG_LATEST}
	kind load docker-image ${IMG}

##@ Tools / Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool binaries

GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint

GOLINT_VERSION ?= 1.53.2

.PHONY: golangci-lint
golangci-lint:
	@if test -x $(LOCALBIN)/golangci-lint && ! $(LOCALBIN)/golangci-lint version | grep -q $(GOLINT_VERSION); then \
		echo "$(LOCALBIN)/golangci-lint version is not expected $(GOLINT_VERSION). Removing it before installing."; \
		rm -rf $(LOCALBIN)/golangci-lint; \
	fi
	test -s $(LOCALBIN)/golangci-lint || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v$(GOLINT_VERSION)
