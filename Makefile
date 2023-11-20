SHELL = /bin/bash

PROJECT_NAME := "github.com/cloud-barista/cm-honeybee/agent"
PKG_LIST := $(shell go list ${PROJECT_NAME}/...)

GOPROXY_OPTION := GOPROXY=direct GOSUMDB=off
GO_COMMAND := ${GOPROXY_OPTION} go

.PHONY: all lint test race coverage coverhtml gofmt update build clean help

all: build

lint: ## Lint the files
	@echo "Running linter..."
	@if [ ! -f "$(GOPATH)/bin/golangci-lint" ] && [ ! -f "$(GOROOT)/bin/golangci-lint" ]; then \
	  ${GO_COMMAND} install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	@golangci-lint run -E contextcheck -E revive

test: ## Run unittests
	@echo "Running tests..."
	@${GO_COMMAND} test -v ${PKG_LIST}

race: ## Run data race detector
	@echo "Checking races..."
	@${GO_COMMAND} test -race -v ${PKG_LIST}

coverage: ## Generate global code coverage report
	@echo "Generating coverage report..."
	@${GO_COMMAND} test -v -coverprofile=coverage.out ${PKG_LIST}
	@${GO_COMMAND} tool cover -func=coverage.out

coverhtml: coverage ## Generate global code coverage report in HTML
	@echo "Generating coverage report in HTML..."
	@${GO_COMMAND} tool cover -html=coverage.out

gofmt: ## Run gofmt for go files
	@echo "Running gofmt..."
	@find -type f -name '*.go' -not -path "./vendor/*" -exec $(GOROOT)/bin/gofmt -s -w {} \;

update: ## Update all of module dependencies
	@echo Updating dependencies...
	@${GO_COMMAND} get -u
	@echo Checking dependencies...
	@${GO_COMMAND} mod tidy
	@echo Syncing vendor...
	@${GO_COMMAND} mod vendor

build: lint ## Build the binary file
	@echo Checking dependencies...
	@${GO_COMMAND} mod tidy
	@echo Syncing vendor...
	@${GO_COMMAND} mod vendor
	@echo Building...
	@CGO_ENABLED=0 ${GO_COMMAND} build -o ${PROJECT_NAME} main.go
	@echo Build finished!

clean: ## Remove previous build
	@echo Cleaning build...
	@rm -f coverage.out
	@${GO_COMMAND} clean

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
