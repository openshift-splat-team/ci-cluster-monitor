<!--
 Copyright 2025.
 SPDX-License-Identifier: Apache-2.0
 -->

# CI Cluster Monitor

Monitors open PRs across `nutanix-cloud-native` repositories, tracks their age and CI status, and classifies them by severity.

## Prerequisites

- Go 1.26+
- `GITHUB_TOKEN` with repo read access

## Build and Run

```bash
make build

GITHUB_TOKEN=<token> ./bin/pr-monitor --config config/repos.yaml
```

Use `--format json` for JSON output.

## Configuration

See [`config/repos.yaml`](config/repos.yaml) for monitored repositories, age thresholds, and filters.

## License

Apache License 2.0 — see [LICENSE](LICENSE).
