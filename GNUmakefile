CLI_SOURCE_FILES?=./cmd/plugin
CLI_BINARY_NAME?=binary
CLI_DESTINATION=./bin/$(CLI_BINARY_NAME)

GOLANGCI_VERSION=v1.63.4 # Also update golangci-lint GH action in code-health.yml when updating this version

.PHONY: build
build:
	@echo "==> Building plugin binary: $(CLI_BINARY_NAME)"
	go build -o $(CLI_DESTINATION) $(CLI_SOURCE_FILES)

.PHONY: tools
tools:  ## Install dev tools
	@echo "==> Installing dependencies..."
	go telemetry off # disable sending telemetry data, more info: https://go.dev/doc/telemetry
	go install github.com/rhysd/actionlint/cmd/actionlint@latest
	go install golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment@latest
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin $(GOLANGCI_VERSION)
