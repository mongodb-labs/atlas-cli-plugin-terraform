CLI_SOURCE_FILES?=./cmd/plugin
CLI_BINARY_NAME?=binary
CLI_DESTINATION=./bin/$(CLI_BINARY_NAME)
MANIFEST_FILE?=./bin/manifest.yml
WIN_MANIFEST_FILE?=./bin/manifest.windows.yml

GOLANGCI_VERSION=v1.64.7 # Also update golangci-lint GH action in code-health.yml when updating this version

.PHONY: build 
build: ## Generate the binary in ./bin
	@echo "==> Building plugin binary: $(CLI_BINARY_NAME)"
	go build -o $(CLI_DESTINATION) $(CLI_SOURCE_FILES)

.PHONY: tools 
tools: ## Install the dev tools (dependencies)
	@echo "==> Installing dev tools..."
	go telemetry off # disable sending telemetry data, more info: https://go.dev/doc/telemetry
	go install github.com/rhysd/actionlint/cmd/actionlint@latest
	go install golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment@latest
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin $(GOLANGCI_VERSION)

.PHONY: clean
clean: ## Clean binary folders
	rm -rf ./bin ./bin-plugin

.PHONY: test
test: ## Run unit tests
	go test ./internal/... -timeout=30s -parallel=4 -race

.PHONY: test-update
test-update: ## Run unit tests and update the golden files
	go test ./internal/... -timeout=30s -parallel=4 -race -update

.PHONY: test-e2e
test-e2e: local ## Run E2E tests (running the plugin binary)
	ATLAS_CLI_EXTRA_PLUGIN_DIRECTORY="${PWD}/bin-plugin" go test ./test/... -timeout=30s -parallel=4 -race

.PHONY: local
local: clean build ## Allow to run the plugin locally
	@echo "==> Configuring plugin locally"
	VERSION=0.0.1-local GITHUB_REPOSITORY_OWNER=owner GITHUB_REPOSITORY_NAME=repo $(MAKE) generate-manifest
	@mkdir -p ./bin-plugin
	cp -r ./bin ./bin-plugin/atlas-cli-plugin-terraform
	@echo
	@echo "==> Plugin is ready to be used locally"	
	@echo "run: export ATLAS_CLI_EXTRA_PLUGIN_DIRECTORY=./bin-plugin"
	@echo "then this command should show the plugin: atlas plugin list"

.PHONY: generate-all-manifests
generate-all-manifests: generate-manifest generate-manifest-windows ## Generate all the manifest files

.PHONY: generate-manifest
generate-manifest: ## Generate the manifest file for non-windows OSes
	@echo "==> Generating non-windows manifest file"
	@mkdir -p ./bin
	BINARY=$(CLI_BINARY_NAME) envsubst < manifest.template.yml > $(MANIFEST_FILE)

.PHONY: generate-manifest-windows
generate-manifest-windows: ## Generate the manifest file for windows OSes
	@echo "==> Generating windows manifest file"
	CLI_BINARY_NAME="${CLI_BINARY_NAME}.exe" MANIFEST_FILE="$(WIN_MANIFEST_FILE)" $(MAKE) generate-manifest

.PHONY: help
.DEFAULT_GOAL := help
help:
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' | sort
	
