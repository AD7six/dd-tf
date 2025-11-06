# Monitors command

Download Datadog monitors as JSON files.

## Synopsis

```bash
bin/dd-tf monitors download [flags]
```

## Flags

- `--id` string: Monitor ID(s) to download (comma-separated integers).
- `--all`: Download all monitors.
- `--update`: Update already-downloaded monitors by scanning existing JSON files under `{DATA_DIR}/monitors` and re-downloading by `id`.
- `--team` string: Filter by team (convenience for tag `team:x`).
- `--tags` string: Comma-separated list of tags to filter monitors.
- `--priority` int: Filter by monitor priority.
- `--output` string: Output path template (supports `{DATA_DIR}`, `{id}`, `{name}`, `{team}`, `{priority}`, and any tag key placeholder like `{env}`).

At least one of `--update`, `--all`, `--id`, `--team`, `--tags`, or `--priority` must be provided.

## Examples

```bash
# Download specific monitors by id
bin/dd-tf monitors download --id=1234,5678

# Refresh monitors already tracked locally
bin/dd-tf monitors download --update

# Download all monitors
bin/dd-tf monitors download --all

# Download all monitors tagged with service:my-service
bin/dd-tf monitors download --tags="service:my-service"

# Group by team and include name and priority in filename
bin/dd-tf monitors download --all --output='{DATA_DIR}/monitors/{team}/{priority}/{name}-{id}.json'
```

## Path templating

Default: `{DATA_DIR}/monitors/{id}.json`

Placeholders:

- `{DATA_DIR}`
- `{id}`
- `{name}`
- `{team}`
- `{priority}`
- Any tag key placeholder like `{env}` or `{service}` – any tag present on the monitor

Notes:

- Names and tag values are sanitized (non-alphanumerics → `-`)
- Missing values render as `none`

## Environment

- `DATA_DIR` – base folder for data files (default: `data`)
- `MONITORS_PATH_TEMPLATE` – monitor path pattern (default: `{DATA_DIR}/monitors/{id}.json`)

## See also

- General docs: [docs/README.md](./README.md)
- Dashboards: [docs/dashboards.md](./dashboards.md)
