SHELL = /bin/bash

.PHONY: all dependency lint test race coverage coverhtml gofmt update swag swagger build linux windows run run_docker stop stop_docker clean help

all: build

dependency: ## Get dependencies
	@"$(MAKE)" -C ./agent dependency
	@"$(MAKE)" -C ./server dependency

lint: ## Lint the files
	@"$(MAKE)" -C ./agent lint
	@"$(MAKE)" -C ./server lint

test: ## Run unit tests
	@"$(MAKE)" -C ./agent test
	@"$(MAKE)" -C ./server test

race: ## Run data race detector
	@"$(MAKE)" -C ./agent race
	@"$(MAKE)" -C ./server race

coverage: ## Generate global code coverage report
	@"$(MAKE)" -C ./agent coverage
	@"$(MAKE)" -C ./server coverage

coverhtml: ## Generate global code coverage report in HTML
	@"$(MAKE)" -C ./agent coverhtml
	@"$(MAKE)" -C ./server coverhtml

gofmt: ## Run gofmt for go files
	@"$(MAKE)" -C ./agent gofmt
	@"$(MAKE)" -C ./server gofmt

update: ## Update all of module dependencies
	@"$(MAKE)" -C ./agent update
	@"$(MAKE)" -C ./server update

swag swagger: ## Generate Swagger Documentation
	@"$(MAKE)" -C ./agent swag
	@"$(MAKE)" -C ./server swag

build: ## Build binary files
	@"$(MAKE)" -C ./agent build
	@"$(MAKE)" -C ./server build

build-only: ## Build binary files without running linter
	@"$(MAKE)" -C ./agent build-only
	@"$(MAKE)" -C ./server build-only

linux: ## Build binary files for Linux
	@"$(MAKE)" -C ./agent linux
	@"$(MAKE)" -C ./server linux

windows: ## Build binary files for Windows
	@"$(MAKE)" -C ./agent windows
	@"$(MAKE)" -C ./server windows

run: ## Run built binaries
	@"$(MAKE)" -C ./agent run
	@"$(MAKE)" -C ./server run

run_docker: ## Run built binaries within Docker
	@"$(MAKE)" -C ./agent run_docker
	@"$(MAKE)" -C ./server run_docker

stop: ## Stop built binaries
	@"$(MAKE)" -C ./agent stop
	@"$(MAKE)" -C ./server stop

stop_docker: ## Stop Docker containers
	@"$(MAKE)" -C ./agent stop_docker
	@"$(MAKE)" -C ./server stop_docker

clean: ## Remove previous builds
	@"$(MAKE)" -C ./agent clean
	@"$(MAKE)" -C ./server clean

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
