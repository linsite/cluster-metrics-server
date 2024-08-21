REGISTRY?=kubernetes-sigs
IMAGE?=k8s-cluster-metrics-adapter
TEMP_DIR:=$(shell mktemp -d)
ARCH?=amd64
OUT_DIR?=./_output
GOPATH:=$(shell go env GOPATH)

VERSION?=latest

GOLANGCI_VERSION:=1.55.2

.PHONY: all
all: build-adapter


# Build
# -----

.PHONY: build-adapter
build-adapter: 
	CGO_ENABLED=0 GOOS=linux GOARCH=$(ARCH) go build -o $(OUT_DIR)/$(ARCH)/adapter main.go


# Format and lint
# ---------------

HAS_GOLANGCI_VERSION:=$(shell $(GOPATH)/bin/golangci-lint version --format=short)
.PHONY: golangci
golangci:
ifneq ($(HAS_GOLANGCI_VERSION), $(GOLANGCI_VERSION))
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH)/bin v$(GOLANGCI_VERSION)
endif

.PHONY: verify-lint
verify-lint: golangci
	$(GOPATH)/bin/golangci-lint run --modules-download-mode=readonly || (echo 'Run "make update-lint"' && exit 1)

.PHONY: update-lint
update-lint: golangci
	$(GOPATH)/bin/golangci-lint run --fix --modules-download-mode=readonly


# License
# -------

HAS_ADDLICENSE:=$(shell which $(GOPATH)/bin/addlicense)
.PHONY: verify-licenses
verify-licenses:addlicense
	find -type f -name "*.go" | xargs $(GOPATH)/bin/addlicense -check || (echo 'Run "make update-licenses"' && exit 1)

.PHONY: update-licenses
update-licenses: addlicense
	find -type f -name "*.go" | xargs $(GOPATH)/bin/addlicense -c "The Kubernetes Authors."

.PHONY: addlicense
addlicense:
ifndef HAS_ADDLICENSE
	go install -mod=readonly github.com/google/addlicense
endif


# Verify
# ------

.PHONY: verify
verify: verify-deps verify-lint verify-licenses verify-generated

.PHONY: verify-deps
verify-deps:
	go mod verify
	go mod tidy
	@git diff --exit-code -- go.sum go.mod

.PHONY: verify-generated
verify-generated: update-generated
	@git diff --exit-code -- $(generated_files)


# Test
# ----

.PHONY: test
test:
	CGO_ENABLED=0 go test ./pkg/...

.PHONY: adapter-container
adapter-container: build-adapter
	cp deployments/Dockerfile $(TEMP_DIR)
	cp $(OUT_DIR)/$(ARCH)/adapter $(TEMP_DIR)/adapter
	cd $(TEMP_DIR) && sed -i.bak "s|BASEIMAGE|scratch|g" Dockerfile
	docker build -t $(REGISTRY)/$(IMAGE)-$(ARCH):$(VERSION) $(TEMP_DIR)

