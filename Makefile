MAKEFILE_PATH := $(abspath $(dir $(abspath $(lastword $(MAKEFILE_LIST)))))
PATH := $(MAKEFILE_PATH):$(PATH)

export GOBIN := $(MAKEFILE_PATH)/bin

PATH := $(GOBIN):$(PATH)

GOLANGCI_LINT_VERSION ?= $(shell cd tools; go list -m -f '{{.Version}}' github.com/golangci/golangci-lint)

.PHONY: all
all: clean format build lint test

.PHONY: clean
clean:
	@echo clean
	@go clean

.PHONY: build
build:
	@echo build
	@go build -o $(GOBIN)/import-gitlab-commits

.PHONY: test
test:
	@echo test
	@go test -shuffle=on -count=1 -race -v ./...

.PHONY: test-integration
test-integration:
	@echo test-integration
	@go test -tags=integration -run=TestGitLab -shuffle=on -count=1 -race -v ./...

.PHONY: lint
lint:
	@echo lint
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	@$(GOBIN)/golangci-lint run

.PHONY: gh-lint-version
gh-lint-version:
	@echo "GOLANGCI_LINT_VERSION=$(GOLANGCI_LINT_VERSION)"

.PHONY: format
format:
	@echo format
	@go fmt $(PKGS)

.PHONY: generate
generate:
	@echo generate
	@go generate ./...

.PHONY: run
run:
	@echo run
	@go run -race .
