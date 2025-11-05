# dd-tf

[![CI](https://github.com/AD7six/dd-tf/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/AD7six/dd-tf/actions/workflows/ci.yml)
[![AIL 3](https://img.shields.io/badge/AIL-3-blue)](https://danielmiessler.com/blog/ai-influence-level-ail)

Datadog ⇄ Terraform helper CLI.

Functionality useful for managing Datadog resources as JSON files.

Status: works - Expect sharp edges though, contributions welcome 🙂.

## Why

If you're managing your Datadog account config entirely in the UI you've likely
hit problems such as things being changed (unintentionally) by colleagues. How
can you see what has changed, and if necessary restore an earlier version?

That's the general context for this cli, with dashboards being the start point.

## What it does

* Downloads Datadog dashboards filtered by ID, team tag or all of them using
  Datadog's API.
* Writes JSON files for each dashboard to a path of your choice.

It does not push changes back to Datadog; the intent is to use Terraform is the
source of truth for restoring state when necessary.

## Install

Prerequisites: Go 1.21+

* From source (recommended during development):

```bash
make build
```

Binary will be created at `bin/dd-tf`.

* Or install directly with Go:

```bash
go install github.com/AD7six/dd-tf/cmd/dd-tf@latest
```

## Configure

The CLI reads environment variables and also supports a local `.env` file (loaded via `godotenv`).

Minimum required:

* `DD_API_KEY` – your Datadog API key
* `DD_APP_KEY` – your Datadog application key

Optional:

* `DD_API_DOMAIN` – Datadog site API domain (default: `api.datadoghq.com`)
* `DASHBOARDS_DIR` – base folder for dashboard files (default: `data/dashboards`)
* `DASHBOARDS_FILENAME_PATTERN` – filename pattern (default: `{id}.json`)
* `DASHBOARDS_PATH_PATTERN` – full path pattern (default: `{DASHBOARDS_DIR}/{id}.json`)

Create a `.env` file in your repo root:

```dotenv
DD_API_KEY=your_api_key
DD_APP_KEY=your_app_key
# DD_API_DOMAIN=api.datadoghq.eu
# DASHBOARDS_DIR=data/dashboards
# DASHBOARDS_PATH_PATTERN={DASHBOARDS_DIR}/{team}/{title}-{id}.json
```

### Path templating

By default, dashboard files are created in `data/dashboards/{id}.json`.

This can be overridden via cli arguments:

```
dd-tf dashboards download --all --output='/somewhere/else/{id}-{title}.json'
```

Or by setting `DASHBOARDS_PATH_PATTERN`, `DASHBOARDS_FILENAME_PATTERN` or
`DASHBOARDS_DIR` environment variables. Literal strings can of course be used,
additionally the following placeholders are supported:

* `{DASHBOARDS_DIR}`
* `{id}`
* `{title}`
* `{team}`
* `{anyTagKey}`

Examples:

* Group by team, include title: `{DASHBOARDS_DIR}/{team}/{title}-{id}.json`
* Flat layout by ID only: `{DASHBOARDS_DIR}/{id}.json`

Titles and tag values are sanitized for safe filenames (non-alphanumerics →
`-`). Missing tags render as "none".

## Usage

List commands:

```bash
bin/dd-tf --help
bin/dd-tf dashboards --help
bin/dd-tf dashboards download --help
```

Common flows:

1) Pull specific dashboards by ID (comma-separated):

```bash
# ID is visible in the Datadog URL: https://app.datadoghq.com/dash/<id>
bin/dd-tf dashboards download --id=f6z-bm3-amx,3ir-qw8-ccn
```

2) Refresh all dashboards currently tracked in the dashboards dir (re-download
   by scanning existing JSON files for their `id`):

```bash
bin/dd-tf dashboards download --update
```

3) Download all dashboards from your Datadog account:

```bash
bin/dd-tf dashboards download --all
```

4) Download team dashboards:

```bash
bin/dd-tf dashboards download --team=platform
```

Notes:

* The tool prints each file path it writes (or errors if any).

## End-to-end workflow

### UI → file → Terraform

_See the `./terraform` folder for an example, functional terraform project._

1. Make your changes in the Datadog UI.
2. Download/update the json files to your terraform project:
     - For one dashboard: `dd-tf dashboards download --id=<id>`
     - For all dashboards: `dd-tf dashboards download --all`
     - To refresh all existing tracked dashboards to match current state: `dd-tf dashboards download --update`
4. Commit the resulting JSON files.
5. Reference the JSON in Terraform using the Datadog provider - see
   `./terraform` project for details.

### Drift/unwanted change → revert

Unwanted changes to something? Re-apply your last committed Terraform configuration:

```bash
terraform apply
```

3. Optionally re-run `dd-tf dashboards download --update` to confirm the remote now matches your files.

## Design choice: raw JSON over the official Datadog Go client

I intentionally chose not to use the official Datadog Go API client for this
tool. I previously found it can lag behind new features and may not even expose
certain api endpoints, and sometimes strips data present in api responses -
because it reads the api response, parses it, and generates json from that
parsed representation. Rather than worry about that, using the REST API directly
and persisting the raw JSON ensures confidence of avoiding such issues, with
simpler code.

## Troubleshooting

* 401/403 from the API: Check `DD_API_KEY`, `DD_APP_KEY`, and `DD_API_DOMAIN`.
* 5xx from the API: The API is having a bad day... struggle on or try again
  later.
* Files not where you expect: Verify `DASHBOARDS_PATH_PATTERN` or your
  `--output` flag and remember titles/tags are sanitized.

## Repository layout

* `cmd/dd-tf/` – CLI entrypoint
* `internal/datadog/` – dashboard download logic and path templating
* `internal/utils/` – settings, HTTP client, helpers
* `data/dashboards/` – example/scratch output directory for JSON files

## Roadmap

* Add monitors resource handling ?
* Additional resource types (tbd)
* Optional helpers for Terraform generation (tbd)
* HTTP request queue/concurrency limiter to cap parallel API calls

## License

MIT – see `LICENSE`.

datadog_dashboard_resource: https://registry.terraform.io/providers/DataDog/datadog/latest/docs/resources/dashboard
datadog_dashboard_json_resource: https://registry.terraform.io/providers/DataDog/datadog/latest/docs/resources/dashboard_json
