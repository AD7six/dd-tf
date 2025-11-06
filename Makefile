DEFAULT_GOAL := help

.PHONY: help
help: ## Lists help commands
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-36s\033[0m %s\n", $$1, $$2}'

.PHONY: fmt 
fmt: go-fmt tf-fmt ## Format code

.PHONY: lint 
lint: go-lint ## Lint code

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
run: ## Compile and run the dev cli
	go run cmd/dd-tf/main.go

.PHONY: clean
clean: ## Cleans the build artifacts
	@rm -rf bin/

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

.PHONY: go-lint
go-lint: # Go only, lint files
	@go vet ./...


