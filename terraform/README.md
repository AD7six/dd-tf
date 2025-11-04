# Terraform example for Datadog dashboards

This project applies every JSON file under `../data/dashboards` as a
`datadog_dashboard_json` resource.

## Prereqs

- Terraform >= 1.3
- Datadog credentials, provided either via variables (recommended here) or environment variables.
  - Using variables (JSON): copy `terraform.tfvars.json.example` to `terraform.tfvars.json` and fill in values.
  - Using environment vars (if you remove variables from provider block): `DATADOG_API_KEY`, `DATADOG_APP_KEY`, and the API URL via `DATADOG_HOST`/`DATADOG_SITE` per provider docs.

## What this does

- Recursively discovers all `*.json` files in `../data/dashboards`
- Uses the `id` field from each JSON as a stable key
- Applies each JSON to Datadog using the `datadog_dashboard_json` resource

## Try it

```bash
# From this folder
$ tree ../data/dashboards # Previously download dashboards of interest
../data/dashboards
├── abc-def-gh1.json
├── abc-def-gh2.json
└── abc-def-gh3.json
$ cp terraform.tfvars.json.example terraform.tfvars.json   
$ vim terraform.tfvars.json # then edit values
$ terraform init
$ make import.tf # generate import statements from dashboard files
$ terraform plan
...
Plan: 42 to import, 0 to add, 42 to change, 0 to destroy.
$ terraform apply
...
No changes. Your infrastructure matches the configuration.
...
```

Before applying, check that all dashboards are going to be imported, and none
created.

If your dashboards are in a different location, update `locals.dashboards_root`
in `main.tf`.
