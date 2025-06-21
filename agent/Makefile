SHELL = /bin/bash

MODULE_NAME := "cm-honeybee-agent"
PROJECT_NAME := "github.com/cloud-barista/cm-honeybee/agent"
PKG_LIST := $(shell go list ${PROJECT_NAME}/... 2>&1)

GOPROXY_OPTION := GOPROXY=direct GOSUMDB=sum.golang.org GONOSUMDB=*,!golang.org/toolchain
GO_COMMAND := ${GOPROXY_OPTION} go
GOPATH := $(shell go env GOPATH)

.PHONY: all dependency lint test race coverage coverhtml gofmt update swag swagger build build-only linux windows run run_docker stop stop_docker clean help

all: build

dependency: ## Get dependencies
	@echo Checking dependencies...
	@${GO_COMMAND} mod tidy

lint: dependency ## Lint the files
	@echo "Running linter..."
	@go_path=${GOPATH}; \
	  kernel_name=`uname -s` && \
	  if [[ $$kernel_name == "CYGWIN"* ]] || [[ $$kernel_name == "MINGW"* ]]; then \
	    drive=`go env GOPATH | cut -f1 -d':' | tr '[:upper:]' '[:lower:]'`; \
	    path=`go env GOPATH | cut -f2 -d':' | sed 's@\\\\\\@\/@g'`; \
	    cygdrive_prefix=`mount -p | tail -n1 | awk '{print $$1}'`; \
	    go_path=`echo $$cygdrive_prefix/$$drive/$$path | sed 's@\/\/@\/@g'`; \
	  fi; \
	  if [ ! -f "$$go_path/bin/golangci-lint" ]; then \
	    ${GO_COMMAND} install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@feat/go1.25; \
	  fi; \
	  $$go_path/bin/golangci-lint run

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
	@cd cmd/${MODULE_NAME} && ${GO_COMMAND} get -u
	@echo Checking dependencies...
	@${GO_COMMAND} mod tidy

swag swagger: ## Generate Swagger Documentation
	@echo "Running swag..."
	@go_path=${GOPATH}; \
	  kernel_name=`uname -s` && \
	  if [[ $$kernel_name == "CYGWIN"* ]] || [[ $$kernel_name == "MINGW"* ]]; then \
	    drive=`go env GOPATH | cut -f1 -d':' | tr '[:upper:]' '[:lower:]'`; \
	    path=`go env GOPATH | cut -f2 -d':' | sed 's@\\\\\\@\/@g'`; \
	    cygdrive_prefix=`mount -p | tail -n1 | awk '{print $$1}'`; \
	    go_path=`echo $$cygdrive_prefix/$$drive/$$path | sed 's@\/\/@\/@g'`; \
	  fi; \
	  if [ ! -f "$$go_path/bin/swag" ]; then \
	    ${GO_COMMAND} install github.com/swaggo/swag/cmd/swag@latest; \
	  fi; \
	  $$go_path/bin/swag init -g ./pkg/api/rest/server/server.go --pd -o ./pkg/api/rest/docs/ > /dev/null

build: lint swag ## Build the binary file
	@echo Building...
	@kernel_name=`uname -s` && \
	  if [[ $$kernel_name == "Linux" ]]; then \
	    cd cmd/${MODULE_NAME} && CGO_ENABLED=0 ${GO_COMMAND} build -o ${MODULE_NAME} main.go; \
	  elif [[ $$kernel_name == "CYGWIN"* ]] || [[ $$kernel_name == "MINGW"* ]]; then \
	    cd cmd/${MODULE_NAME} && GOOS=windows CGO_ENABLED=0 ${GO_COMMAND} build -o ${MODULE_NAME}.exe main.go; \
	  else \
	    echo $$kernel_name; \
	    echo "Not supported Operating System. ($$kernel_name)"; \
	  fi
	@git diff > .diff_last_build
	@echo Build finished!

build-only: swag ## Build the binary file without running linter
	@echo Building...
	@kernel_name=`uname -s` && \
	  if [[ $$kernel_name == "Linux" ]]; then \
	    cd cmd/${MODULE_NAME} && CGO_ENABLED=0 ${GO_COMMAND} build -o ${MODULE_NAME} main.go; \
	  elif [[ $$kernel_name == "CYGWIN"* ]] || [[ $$kernel_name == "MINGW"* ]]; then \
	    cd cmd/${MODULE_NAME} && GOOS=windows CGO_ENABLED=0 ${GO_COMMAND} build -o ${MODULE_NAME}.exe main.go; \
	  else \
	    echo $$kernel_name; \
	    echo "Not supported Operating System. ($$kernel_name)"; \
	  fi
	@git diff > .diff_last_build
	@git rev-parse HEAD > .git_hash_last_build
	@echo Build finished!

linux: lint swag ## Build the binary file for Linux
	@echo Building...
	@cd cmd/${MODULE_NAME} && GOOS=linux CGO_ENABLED=0 ${GO_COMMAND} build -o ${MODULE_NAME} main.go
	@git diff > .diff_last_build
	@echo Build finished!

windows: lint swag ## Build the binary file for Windows
	@echo Building...
	@cd cmd/${MODULE_NAME} && GOOS=windows CGO_ENABLED=0 ${GO_COMMAND} build -o ${MODULE_NAME}.exe main.go
	@git diff > .diff_last_build
	@echo Build finished!

run: ## Run the built binary
	@sudo killall ${MODULE_NAME} | true
	@git diff > .diff_current
	@STATUS=`diff .diff_last_build .diff_current 2>&1 > /dev/null; echo $$?` && \
	  GIT_HASH_MINE=`git rev-parse HEAD` && \
	  GIT_HASH_LAST_BUILD=`cat .git_hash_last_build 2>&1 > /dev/null | true` && \
	  if [ "$$STATUS" != "0" ] || [ "$$GIT_HASH_MINE" != "$$GIT_HASH_LAST_BUILD" ]; then \
	    "$(MAKE)" build; \
	  fi
	@@kernel_name=`uname -s` && \
	  cp -RpPf conf cmd/${MODULE_NAME}/ && \
	  if [[ $$kernel_name == "CYGWIN"* ]] || [[ $$kernel_name == "MINGW"* ]]; then \
	    ./cmd/${MODULE_NAME}/${MODULE_NAME}.exe & \
	  else \
        ./cmd/${MODULE_NAME}/${MODULE_NAME} || echo "Trying with sudo..." && sudo ./cmd/${MODULE_NAME}/${MODULE_NAME} & \
	  fi

run_docker: ## Run the built binary within Docker
	@git diff > .diff_current
	@STATUS=`diff .diff_last_build .diff_current 2>&1 > /dev/null; echo $$?` && \
	  GIT_HASH_MINE=`git rev-parse HEAD` && \
	  GIT_HASH_LAST_BUILD=`cat .git_hash_last_build 2>&1 > /dev/null | true` && \
	  if [ "$$STATUS" != "0" ] || [ "$$GIT_HASH_MINE" != "$$GIT_HASH_LAST_BUILD" ]; then \
	    docker rmi -f cm-honeybee-agent:latest; \
	  fi
	@docker compose up -d

stop: ## Stop the built binary
	@sudo killall ${MODULE_NAME} | true

stop_docker: ## Stop the Docker container
	@docker compose down

clean: ## Remove previous build
	@echo Cleaning build...
	@rm -f coverage.out
	@cd cmd/${MODULE_NAME} && ${GO_COMMAND} clean

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
