DEFAULT_GOAL := help

.PHONY: help
help: ## Lists help commands
	@grep -h -E '^[a-zA-Z_.-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-36s\033[0m %s\n", $$1, $$2}'

import.tf: ## Generates import.tf from dashboards in ../data/dashboards
	@echo "# After downloading dashboards - import them to add them to terraform state." > import.tf
	@echo "# Otherwise, copies of the dashboards will be created." >> import.tf
	@echo "# Requires Terraform >= 1.5.0" >> import.tf
	@echo "" >> import.tf
	@find ../data/dashboards -name '*.json' -type f | while read file; do \
		id=$$(basename "$$file" .json); \
		echo "import {" >> import.tf; \
		echo "  to = datadog_dashboard_json.dashboard[\"$$id\"]" >> import.tf; \
		echo "  id = \"$$id\"" >> import.tf; \
		echo "}" >> import.tf; \
		echo "" >> import.tf; \
	done
	@echo "Generated import.tf with $$(grep -c '^import {' import.tf) dashboard(s)"
