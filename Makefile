DEFAULT_GOAL := help

.PHONY: help
help: ## Lists help commands
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-36s\033[0m %s\n", $$1, $$2}'

.PHONY: tf-lock
tf-lock: ## Locks all terraform projects dependencies
	@BASE=$$(pwd);\
	for d in ${ATLANTIS_PROJECTS_DIRS}; do\
		echo "Locking $$d/"; \
		cd $$BASE/$$d;\
		terraform init -upgrade;\
		terraform providers lock ${TF_PLATFORMS};\
	done

.PHONY: fmt 
fmt: go-fmt tf-fmt ## Format code

.PHONY: lint 
lint: go-lint ## Lint code

.PHONY: test
test: ## Runs go tests
	@go test -v ./...	

.PHONY: build
build: ## Builds the go binary
	@go build -o bin/dd-tf ./cmd/dd-tf/main.go

.PHONY: run
run: ## Compile and run the dev cli
	@go run cmd/dd-tf/main.go

.PHONY: clean
clean: ## Cleans the build artifacts
	@rm -rf bin/

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


