DEFAULT_GOAL := help

.PHONY: help
help: ## Lists help commands
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-36s\033[0m %s\n", $$1, $$2}'

.PHONY: fmt 
fmt: go-fmt tf-fmt ## Format code

.PHONY: lint 
lint: go-lint ## Lint code (vet + golangci-lint)

.PHONY: test
test: ## Runs go tests
	@go test ./...	

.PHONY: build
build: ## Builds the go binary
	@go build \
		-ldflags "-X github.com/AD7six/dd-tf/internal/commands/version.Version=$(shell git describe --tags --always --dirty 2>/dev/null || echo dev) \
		          -X github.com/AD7six/dd-tf/internal/commands/version.Commit=$(shell git rev-parse --short HEAD 2>/dev/null || echo unknown) \
		          -X github.com/AD7six/dd-tf/internal/commands/version.BuildDate=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)" \
		-o bin/dd-tf ./cmd/dd-tf/main.go

.PHONY: run
run: .env ## Compile and run the dev cli
	go run cmd/dd-tf/main.go

.PHONY: clean
clean: ## Cleans the build artifacts
	@rm -rf bin/

.env: # Create .env file with prompts for required variables
	@echo "Creating .env file..."
	@echo "# Generated on $$(date)" > .env
	@echo "" >> .env
	@echo "## Required" >> .env
	@echo "# Datadog API key: https://app.datadoghq.com/organization-settings/api-keys" >> .env
	@default_api_key=$${DD_API_KEY:-$${DATADOG_API_KEY:-}}; \
	if [ -n "$$default_api_key" ]; then \
		read -p "Enter your Datadog API key [$$default_api_key]: " api_key; \
		api_key=$${api_key:-$$default_api_key}; \
	else \
		read -p "Enter your Datadog API key: " api_key; \
	fi; \
	echo "DD_API_KEY=$$api_key" >> .env
	@echo "" >> .env
	@echo "# Datadog Application key: https://app.datadoghq.com/organization-settings/application-keys" >> .env
	@default_app_key=$${DD_APP_KEY:-$${DATADOG_APP_KEY:-$${DD_APPLICATION_KEY:-}}}; \
	if [ -n "$$default_app_key" ]; then \
		read -p "Enter your Datadog Application key [$$default_app_key]: " app_key; \
		app_key=$${app_key:-$$default_app_key}; \
	else \
		read -p "Enter your Datadog Application key: " app_key; \
	fi; \
	echo "DD_APP_KEY=$$app_key" >> .env
	@echo "" >> .env
	@echo "## Optional - defaults" >> .env
	@echo "" >> .env
	@sed -n '/^## Optional/,$$p' internal/config/defaults.env | sed '1,2d' | sed 's/^\([A-Z_]*=\)/#\1/' >> .env
	@echo ""
	@echo "✓ .env file created successfully!"
	@echo "You can now edit .env to customize optional settings."

.PHONY: release
release: ## Interactive release tagging (creates semver git tag)
	@./scripts/release.sh

###
# These targets are intentionally not documented (single #) so they don't show
# up in help output. There are called by the above targets
###
.PHONY: tf-fmt
tf-fmt: # Terraform only format files
	@terraform fmt -recursive .

.PHONY: go-fmt
go-fmt: # Go only, format files
	@go fmt ./...


# Install golangci-lint if not present (helper target)
.PHONY: tools
tools: # Install developer tools (golangci-lint)
	@command -v golangci-lint >/dev/null 2>&1 || { echo "Installing golangci-lint"; go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.62.0; }

.PHONY: go-lint
go-lint: tools # Run vet and golangci-lint
	@echo "Running go vet" && go vet ./...
	@echo "Running golangci-lint (govet only)" && golangci-lint run --disable-all -E govet ./...


