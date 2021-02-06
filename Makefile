MAKEFILE_PATH := $(abspath $(dir $(abspath $(lastword $(MAKEFILE_LIST)))))
PATH := $(MAKEFILE_PATH):$(PATH)

export GOBIN := $(MAKEFILE_PATH)/bin

PATH := $(GOBIN):$(PATH)

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
	@go test -count=1 -race -v ./...

.PHONY: lint
lint:
	@echo lint
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint
	@$(GOBIN)/golangci-lint run

.PHONY: format
format:
	@echo format
	@go fmt $(PKGS)

.PHONY: vendor
vendor:
	@echo vendor
	@-rm -rf vendor/
	@go mod vendor

.PHONY: generate
generate:
	@echo generate
	@go generate ./...

.PHONY: run
run:
	@echo run
	@go run -race .
