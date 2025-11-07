# Dashboards command

Download Datadog dashboards as JSON files.

## Synopsis

```bash
bin/dd-tf dashboards download [flags]
```

## Flags

- `--id` string: Dashboard ID(s) to download (comma-separated). The ID is visible in the Datadog URL: `https://app.datadoghq.com/dash/<id>`.
- `--all`: Download all dashboards.
- `--update`: Update already-downloaded dashboards by scanning existing JSON files and re-downloading by `id`.
- `--team` string: Filter by team (convenience for tag `team:x`).
- `--tags` string: Comma-separated list of tags to filter dashboards.
- `--output` string: Output path template (supports `{id}`, `{title}`, `{team}`, and any `{tag}` or `{ENV_VAR}`).

At least one of `--update`, `--all`, `--id`, `--team`, or `--tags` must be provided.

## Examples

```bash
# Download specific dashboards by id
bin/dd-tf dashboards download --id=abc-def-gh1,abc-def-gh2

# Refresh dashboards already tracked locally
bin/dd-tf dashboards download --update

# Download all dashboards
bin/dd-tf dashboards download --all

# Download all dashboards owned by my team
bin/dd-tf dashboards download --team=myteam

# Download all dashboards, group by team and include title in filename
bin/dd-tf dashboards download --all --output='data/dashboards/{team}/{title}-{id}.json'
```

## Path templating

Default: `data/dashboards/{id}.json`

Placeholders:

- `data`
- `{id}`
- `{title}`
- `{team}`
- Any tag key placeholder like `{env}` or `{service}` – any tag present on the dashboard

Notes:

- Titles and tag values are sanitized (non-alphanumerics → `-`)
- Missing values render as `none`

## Environment

- `DATA_DIR` – base folder for data files (default: `data`)
- `DASHBOARDS_PATH_TEMPLATE` – dashboard path pattern (default: `$DATA_DIR/dashboards/{id}.json`)

## See also

- General docs: [docs/README.md](./README.md)
- Monitors: [docs/monitors.md](./monitors.md)
