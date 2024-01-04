SHELL = /bin/bash

MODULE_NAME := "cm-honeybee"
PROJECT_NAME := "github.com/cloud-barista/${MODULE_NAME}"
PKG_LIST := $(shell go list ${PROJECT_NAME}/...)

GOPROXY_OPTION := GOPROXY=direct GOSUMDB=off
GO_COMMAND := ${GOPROXY_OPTION} go
GOPATH := $(shell go env GOPATH)

.PHONY: all dependency lint test race coverage coverhtml gofmt update build windows swag swagger clean help

all: build

dependency: ## Get dependencies
	@echo Checking dependencies...
	@${GO_COMMAND} mod tidy

lint: dependency ## Lint the files
	@echo "Running linter..."
	@if [ ! -f "${GOPATH}/bin/golangci-lint" ] && [ ! -f "$(GOROOT)/bin/golangci-lint" ]; then \
	  ${GO_COMMAND} install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	@golangci-lint run -E contextcheck -D unused

test: dependency ## Run unittests
	@echo "Running tests..."
	@${GO_COMMAND} test -v ${PKG_LIST}

race: dependency ## Run data race detector
	@echo "Checking races..."
	@${GO_COMMAND} test -race -v ${PKG_LIST}

coverage: dependency ## Generate global code coverage report
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

build: lint swag ## Build the binary file
	@echo Building...
	@CGO_ENABLED=0 ${GO_COMMAND} build -o ${MODULE_NAME} main.go
	@echo Build finished!

windows: lint ## Build the Windows exe binary file
	@echo Building for Windows system...
	@GOOS=windows CGO_ENABLED=0 ${GO_COMMAND} build -o ${MODULE_NAME}.exe main.go
	@echo Build finished!

swag swagger: ## Generate Swagger Documentation
	@echo "Running swag..."
	@if [ ! -f "${GOPATH}/bin/swag" ] && [ ! -f "$(GOROOT)/bin/swag" ]; then \
	  ${GO_COMMAND} install github.com/swaggo/swag/cmd/swag@latest; \
	fi
	@swag init --parseDependency

clean: ## Remove previous build
	@echo Cleaning build...
	@rm -f coverage.out
	@rm -f docs/docs.go docs/swagger.*
	@${GO_COMMAND} clean

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
