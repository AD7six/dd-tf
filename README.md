#t ./internal/datadog/dashboards -run TestComputeDashboardPath
dd-tf

[![CI](https://github.com/AD7six/dd-tf/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/AD7six/dd-tf/actions/workflows/ci.yml)
[![AIL 2](https://img.shields.io/badge/AIL-2-blue)](https://danielmiessler.com/blog/ai-influence-level-ail)

Datadog ⇄ Terraform helper CLI. Download dashboards and monitors as JSON for version control and Terraform.

Status: works — Expect sharp edges though, contributions welcome 🙂.

## Quick start

Install:

```bash
make build  # or: go install github.com/AD7six/dd-tf/cmd/dd-tf@latest
```

Configure `.env` with `DD_API_KEY` and `DD_APP_KEY` - there's a make target to help you if you need it:
```bash
make .env
```

Common commands:

```bash
# Dashboards
bin/dd-tf dashboards download --all
bin/dd-tf dashboards download --id=abc-def-ghi

# Monitors
bin/dd-tf monitors download --all
bin/dd-tf monitors download --id=1234
```

## Documentation

- Start: [docs/README.md](docs/README.md)
- Dashboards command: [docs/dashboards.md](docs/dashboards.md)
- Monitors command: [docs/monitors.md](docs/monitors.md)

## License

MIT – see `LICENSE`.
