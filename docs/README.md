# dd-tf documentation

dd-tf is a Datadog ⇄ Terraform helper CLI that downloads Datadog resources as
JSON files, so you can track them in version control and use them with
Terraform.

## Why

If you're managing your Datadog resources entirely in the UI you've likely
already had the experience of looking for something (a dashboard or monitor) and
finding it unexpectedly changed or missing. When this happens how can you see
what changed, and restore an earlier version if necessary?

The Datadog-account solution is to enable the audit trail, which works great -
but you don't need to rely on that feature, you can also track resources in your
own version control system.

This CLI lets you pull dashboards and monitors as JSON, with the idea that you'd
then wire them into Terraform of if desired as a git archive of account data.

## What it does

- Downloads Datadog dashboards and monitors using Datadog's REST API
- Writes JSON files for each resource to a path of your choice

This CLI intentionally doesn't have any update mechanism back to Datadog; the
intent is to use Terraform as the source of truth for restoring state when
necessary.

## Install

Prerequisites: Go 1.21+

From source (recommended during development):

```bash
make build
```

Binary will be created at `bin/dd-tf`.

Or install directly with Go:

```bash
go install github.com/AD7six/dd-tf/cmd/dd-tf@latest
```

## Configure

The CLI reads environment variables and also supports a local `.env` file
(loaded via `godotenv`).

Minimum required:

- `DD_API_KEY` – your Datadog API key
- `DD_APP_KEY` – your Datadog application key

Optional:

- `DD_SITE` – Datadog site parameter (default: `datadoghq.com`)
- `DATA_DIR` – base folder for data files (default: `data`)
- `DASHBOARDS_PATH_TEMPLATE` – dashboard path pattern (default: `$DATA_DIR/dashboards/{id}.json`)
- `MONITORS_PATH_TEMPLATE` – monitor path pattern (default: `$DATA_DIR/monitors/{id}.json`)
- `HTTP_TIMEOUT` – HTTP client timeout in seconds (default: `60`)

A `.env` file can be created by running `make .env`

`.env` example:

```shell
❯ make .env
Creating .env file...
Enter your Datadog API key: my api key
Enter your Datadog Application key: my app key

✓ .env file created successfully!
You can now edit .env to customize optional settings.
❯ cat .env
# Generated on Fri Nov  7 12:17:49 CET 2025

## Required
# Datadog API key: https://app.datadoghq.eu/organization-settings/api-keys
DD_API_KEY=my api key

# Datadog Application key: https://app.datadoghq.eu/organization-settings/application-keys
DD_APP_KEY=my app key

## Optional - defaults

# Datadog site (default: datadoghq.com)
#DD_SITE=datadoghq.com

# Base data directory (default: data)
#DATA_DIR=data

# Path templates for resources
# Use $DATA_DIR to reference the data directory
# Use {id}, {title}, {name}, {team}, {priority} for resource-specific placeholders
# Use {ANY_ENV_VAR} (uppercase) to reference environment variables
# Use {any_tag} to reference any tag value
#DASHBOARDS_PATH_TEMPLATE=$DATA_DIR/dashboards/{id}.json
#MONITORS_PATH_TEMPLATE=$DATA_DIR/monitors/{id}.json

# HTTP client timeout in seconds (default: 60)
#HTTP_TIMEOUT=60

# Maximum response body size in bytes (default: 10485760 = 10MB)
#HTTP_MAX_BODY_SIZE=10485760

# Page size for paginated API requests (default: 1000)
#PAGE_SIZE=1000
```

## Path templating

Defaults:

- Dashboards: `data/dashboards/{id}.json`
- Monitors: `data/monitors/{id}.json`

Override via CLI:

```bash
dd-tf dashboards download --all --output='/somewhere/else/{id}-{title}.json'
dd-tf monitors   download --all --output='/somewhere/else/{id}-{name}.json'
```

Or set environment variables: `DATA_DIR`, `DASHBOARDS_PATH_TEMPLATE`,
`MONITORS_PATH_TEMPLATE`.

Supported placeholders (rendered with Go templates):

- `{id}`
- `{title}` (dashboards)
- `{name}` (monitors)
- `{team}`
- `{priority}` (monitors)
- `{ANY_ENV_VAR}` (uppercase) to reference environment variables
- `{any_tag}` to reference any tag value

Notes:

- Titles, names, and tag values are sanitized for safe filenames (non-alphanumerics → `-`).
- If a placeholder is missing or empty the string `none` is used.

## Usage

- Dashboards command: see [docs/dashboards.md](./dashboards.md)
- Monitors command: see [docs/monitors.md](./monitors.md)

You can always list commands via:

```bash
bin/dd-tf --help
bin/dd-tf dashboards --help
bin/dd-tf monitors --help
```

## Workflows

A brief overview of workflows where this tool can be helpful.

### Updating terraform-managed resources

This only really applies to dashboards, as their data structure is not easily
represented via the [standard dashboard resource][datadog_dashboard_resource)
whereas monitors and other resources are likely better suited to their non‑JSON
Terraform representations.

Anything but trivial edits to a terraform-tracked dashboard can be pretty
painful, and even more so to create a dashboard from scratch. The typical
process to do this is either

#### _Really_ painful

1. Modify [datadog_dashboard][datadog_dashboard_resource] terraform resource
2. Look at the `terraform plan` output
3. Hope for the best
4. `terraform apply`, spot a mistake and return to 1.

#### Slightly-less painful

1. Make live edits to the dashboard via Datadog's UI
2. Modify [datadog_dashboard][datadog_dashboard_resource] terraform resource
3. run `terraform plan` until there are no differences reported 
4. `terraform apply` when config matches current state

#### Almost, but still painful

1. Make live edits to the dashboard via Datadog's UI
2. Modify [datadog_dashboard_json][datadog_dashboard_json_resource] terraform resource
3. Run `terraform plan` until there are no differences reported 
5. Run `terraform apply` (no-op)

Even this permutation still has some rough edges:

* The easiest way to get the json representation of a dashboard is to use
  Configure ➡️ `Export dashboard json` 
* 👆🏻 returns a reduced set of properties compared to the Datadog API response
  for a dashboard
* Even if the missing properties have no functional effect, it introduces
  differences and commit churn unless every person updating the json uses the
  same process.

#### Edit in the UI; sync with dd-tf

Where `dd-tf` comes in is to smooth out that process such that it becomes:

1. Make live edits to the dashboard
2. In the relevant terraform project, run `dd-tf dashboards download --update`
3. `git diff` or other validation to verify the live changes are desired
4. Run `terraform plan` with no differences reported
5. Run `terraform apply` (no-op)

This provides the best of both worlds with version-control of the contents of
important dashboards, without a problematic process when they need to update.

See the [sample Terraform project](../terraform/) for a working example that
demonstrates this workflow.

### Archiving all data

Periodically run the equivalent of these commands:

```
cd /my/datadog/git/archive
mv data olddata                 # Move previous data out the way
dd-tf dashboards download --all # Download all dashboards
dd-tf monitors download --all   # Download all monitors
git add data                    # Add all current data
git commit -am "current state"
```

In this way you'll have a full archive of all your dashboards and monitors to
refer to at any time, and then:

* "My dashboard is broken!" ➡️ Find it in the archive ➡️ Restore it
* "A monitor has been deleted!" ➡️ Find it in the archive ➡️ Restore it

For resources that are already tracked by terraform this may not be necessary,
but if there are parameterized resources (create one monitor per environment)
this kind of archive can _still_ be useful when tooling changes are made and it
turns out all the staging environment monitors went missing 🙃.

## Design choices

Datadog frequently adds new features which are not supported by the official Go
SDK (at least, that's been my experience). This tool uses the Datadog REST API
directly to avoid scope for missing data, and a dependency on the official SDK
version.

## Troubleshooting

- 401/403 from the API: Check `DD_API_KEY`, `DD_APP_KEY`, `DD_SITE`.
- 5xx from the API: Retry later; the API may be degraded.
- Files not where you expect: Verify your templating flags and env vars.

## Repository layout

- `cmd/dd-tf/` – CLI entrypoint
- `internal/commands/` – individual commands and subcommands
- `internal/config/` – settings and environment configuration
- `internal/datadog/` – Datadog specific (API) logic
- `internal/http/` – HTTP client with retry logic and rate limiting
- `internal/storage/` – file I/O and JSON writing
- `internal/utils/` – generic string utilities
- `data/` – default output directory for JSON files

## Roadmap

- Additional resource types (tbd)
- Optional helpers for Terraform generation (tbd)

## References

- [Terraform Datadog dashboard resource][datadog_dashboard_resource]
- [Terraform Datadog dashboard_json resource][datadog_dashboard_json_resource]
- [Terraform Datadog monitor resource][datadog_monitor_resource]
- [Terraform Datadog monitor_json resource][datadog_monitor_json_resource]

[datadog_dashboard_resource]: https://registry.terraform.io/providers/DataDog/datadog/latest/docs/resources/dashboard
[datadog_dashboard_json_resource]: https://registry.terraform.io/providers/DataDog/datadog/latest/docs/resources/dashboard_json
[datadog_monitor_resource]: https://registry.terraform.io/providers/DataDog/datadog/latest/docs/resources/monitor
[datadog_monitor_json_resource]: https://registry.terraform.io/providers/DataDog/datadog/latest/docs/resources/monitor_json
