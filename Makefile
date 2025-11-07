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

.PHONY: docker-build
docker-build:
	@echo docker-build
	@docker build -t github.com/alexandear/import-gitlab-commits .

.PHONY: docker-run
docker-run: docker-build
	@echo docker-run
	@docker run --rm \
		-e GITLAB_BASE_URL="${GITLAB_BASE_URL}" \
		-e GITLAB_TOKEN="${GITLAB_TOKEN}" \
		-e COMMITTER_NAME="${COMMITTER_NAME}" \
		-e COMMITTER_EMAIL="${COMMITTER_EMAIL}" \
		-v $$(pwd)/output:/root \
		github.com/alexandear/import-gitlab-commits

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
	@go tool -modfile=tools/go.mod golangci-lint run

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
