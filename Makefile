# Package related
BINARY_NAME=rdma
PACKAGE=rdma-cni
ORG_PATH=github.com/k8snetworkplumbingwg
REPO_PATH=$(ORG_PATH)/$(PACKAGE)
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
BINDIR =$(PROJECT_DIR)/bin
BUILDDIR=$(PROJECT_DIR)/build
GOFILES=$(shell find . -name *.go | grep -vE "(\/vendor\/)|(_test.go)")
PKGS=$(or $(PKG),$(shell cd $(PROJECT_DIR) && go list ./... | grep -v "^$(PACKAGE)/vendor/" | grep -v ".*/mocks"))
TESTPKGS = $(shell go list -f '{{ if or .TestGoFiles .XTestGoFiles }}{{ .ImportPath }}{{ end }}' $(PKGS))

# Version
VERSION?=master
DATE=`date -Iseconds`
COMMIT?=`git rev-parse --verify HEAD`
LDFLAGS="-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

# Docker
IMAGE_BUILDER?=@docker
IMAGEDIR=$(PROJECT_DIR)/images
DOCKERFILE?=$(PROJECT_DIR)/Dockerfile
TAG?=ghcr.io/k8snetworkplumbingwg/rdma-cni
IMAGE_BUILD_OPTS?=
# Accept proxy settings for docker
# To pass proxy for Docker invoke it as 'make image HTTP_POXY=http://192.168.0.1:8080'
DOCKERARGS=
ifdef HTTP_PROXY
	DOCKERARGS += --build-arg http_proxy=$(HTTP_PROXY)
endif
ifdef HTTPS_PROXY
	DOCKERARGS += --build-arg https_proxy=$(HTTPS_PROXY)
endif
IMAGE_BUILD_OPTS += $(DOCKERARGS)

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN := $(shell go env GOPATH)/bin
else
GOBIN := $(shell go env GOBIN)
endif

TARGET_OS ?= $(shell go env GOOS)
TARGET_ARCH ?= $(shell go env GOARCH)

# Options for go build command
GO_BUILD_OPTS ?= CGO_ENABLED=0 GOOS=$(TARGET_OS) GOARCH=$(TARGET_ARCH)
GO_TAGS ?=-tags no_openssl

# Go tools
GOLANGCI_LINT = $(BINDIR)/golangci-lint
# golangci-lint version should be updated periodically
# we keep it fixed to avoid it from unexpectedly failing on the project
# in case of a version bump
GOLANGCI_LINT_VER = v2.7.2
MOCKERY_VERSION ?= v3.5.4
TIMEOUT = 30
Q = $(if $(filter 1,$V),,@)

.PHONY: all
all: generate lint build

$(BUILDDIR): ; $(info Creating build directory...)
	@cd $(PROJECT_DIR) && mkdir -p $@

$(BINDIR): ; $(info Creating bin directory...)
	@cd $(PROJECT_DIR) && mkdir -p $@

build: $(BUILDDIR)/$(BINARY_NAME) ; $(info Building $(BINARY_NAME)...) @ ## Build executable file
	$(info Done!)

$(BUILDDIR)/$(BINARY_NAME): $(GOFILES) | $(BUILDDIR)
	@$(GO_BUILD_OPTS) go build -o $(BUILDDIR)/$(BINARY_NAME) $(GO_TAGS) -ldflags $(LDFLAGS) -v cmd/rdma/main.go

# Tools
$(GOLANGCI_LINT): | $(BINDIR) ; $(info  installing golangci-lint...)
	$Q GOBIN=$(BINDIR) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VER)

MOCKERY= $(BINDIR)/mockery
$(MOCKERY): ; $(info  building mockery...)
	$Q GOBIN=$(BINDIR) go install github.com/vektra/mockery/v3@$(MOCKERY_VERSION)

GOVERALLS = $(BINDIR)/goveralls
$(BINDIR)/goveralls: ; $(info  building goveralls...)
	$Q GOBIN=$(BINDIR) go install github.com/mattn/goveralls@v0.0.12

# Tests
.PHONY: lint
lint: | $(GOLANGCI_LINT) ; $(info  running golangci-lint...) @ ## Run golangci-lint
	$Q $(GOLANGCI_LINT) run ./...

TEST_TARGETS := test-default test-bench test-short test-verbose test-race
.PHONY: $(TEST_TARGETS) test-xml check test tests
test-bench:   ARGS=-run=__absolutelynothing__ -bench=. ## Run benchmarks
test-short:   ARGS=-short        ## Run only short tests
test-verbose: ARGS=-v            ## Run tests in verbose mode with coverage reporting
test-race:    ARGS=-race         ## Run tests with race detector
$(TEST_TARGETS): NAME=$(MAKECMDGOALS:test-%=%)
$(TEST_TARGETS): test
check test tests: ; $(info  running $(NAME:%=% )tests...) @ ## Run tests
	$Q go test -timeout $(TIMEOUT)s $(ARGS) $(TESTPKGS)

test-xml: | $(GO2XUNIT) ; $(info  running $(NAME:%=% )tests...) @ ## Run tests with xUnit output
	$Q 2>&1 go test -timeout $(TIMEOUT)s -v $(TESTPKGS) | tee test/tests.output
	$(GO2XUNIT) -fail -input test/tests.output -output test/tests.xml

COVERAGE_MODE = set
.PHONY: test-coverage test-coverage-tools
test-coverage-tools: | $(GOVERALLS)
test-coverage: COVERAGE_DIR := $(PROJECT_DIR)/test
test-coverage: test-coverage-tools ; $(info  running coverage tests...) @ ## Run coverage tests
	$Q go test -covermode=$(COVERAGE_MODE) -coverprofile=rdma-cni.cover $(PKGS)

# Container image
.PHONY: image
image: ; $(info Building Docker image...)  @ ## Build conatiner image
	$(IMAGE_BUILDER) build -t $(TAG) -f $(DOCKERFILE) $(PROJECT_DIR) $(IMAGE_BUILD_OPTS)

# Misc
.PHONY: clean
clean: ; $(info  Cleaning...)	 @ ## Cleanup everything
	@rm -rf $(BUILDDIR)
	@rm -rf $(BINDIR)
	@rm -rf  test

.PHONY: generate
generate: generate-mocks ## Run all generate-* targets

generate-mocks: $(MOCKERY) ## Generate mocks
	$(MOCKERY)

.PHONY: help
help: ## Show this message
	@grep -E '^[ a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
