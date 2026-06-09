---
sidebar_position: 3
---

# Building

LazyOS provides a `Makefile` that covers building, running, testing, log-watching, and cleanup.

## Makefile Targets

| Target | Purpose |
|---|---|
| `make run-sandbox` | **Primary entry point.** Downloads osqueryd, builds binary, and runs inside kernel sandbox |
| `make run-sandbox-all` | Run sandbox with **both** kernel and AWS cloud backends enabled |
| `./lazyos-osquery.sh` | Ephemeral evaluation mode — downloads everything to `/tmp`, runs, and wipes all traces on exit |
| `make run` | Interactive prompt for all config flags (requires an existing osquery daemon) |
| `make run-with-defaults` | Run immediately with compiled-in defaults (requires an existing osquery daemon) |
| `make watch-logs` | Tail and pretty-print the log file via `jq` |
| `make build` | Build a standalone `lazyos` binary |
| `make test` | Run all tests (summary) |
| `make test-force` | Run all tests, bypassing cache |
| `make test-coverage` | Run tests with per-package coverage report |
| `make test-coverage-html` | Generate and open HTML coverage report |
| `make test-integration` | Run integration tests (summary) against live osquery daemon |
| `make test-integration-verbose` | Run integration tests with verbose output |
| `make clean` | Remove the `lazyos` binary and the entire `./build/` directory (osqueryd is re-downloaded on next `make run-sandbox`) |
| `make clean-default-logs` | Remove `~/.local/state/lazyos/lazyos.log` |

### Running with Sandbox (Primary)

```bash
make run-sandbox
```

Downloads an isolated osqueryd (if not already cached in `./build/osquery/`), builds the `lazyos` binary, starts an ephemeral daemon, and launches the TUI with kernel tables — all in one command. The daemon is automatically cleaned up when the application exits.

To include AWS cloud infrastructure tables alongside kernel tables:

```bash
make run-sandbox-all
```

This enables both the `kernel` and `aws` backends, exposing EC2, S3, IAM, RDS, ECS, EKS, and other cloud resource tables via the sidebar. Use `B` to cycle between backends.

> **AWS requires cloudquery** — the AWS tables are provided by the [cloudquery](https://github.com/ajitm722/cloudquery) osquery extension. See the [cloudquery configuration guide](/docs/reference/osquery-interaction#cloudquery-cloud-resource-instrumentation) for setup instructions.

### Running with Existing Daemon

```bash
make run
```

Prompts for every option with defaults in brackets. Press `Enter` to accept:

```
Configuring LazyOS (press Enter to accept defaults)...

  Config File [~/.config/lazyos/config.yml]:
  OSQuery Socket Path [/tmp/osquery.em]:
  Startup Timeout [10s]:
  Query Timeout [100s]:
  Backends (comma-separated) [kernel]:
  Log File [~/.local/state/lazyos/lazyos.log]:
  Keep Log File? (true/false) [false]:

  Running: lazyos --osquery-socket=/tmp/osquery.em --osquery-startup-timeout=10s --osquery-query-timeout=100s --backend kernel --keep-log=false
```

```bash
make run-with-defaults
```

Skips prompts entirely.

```bash
make build
```

Produces a `lazyos` binary in the project root. Run it manually:

```bash
./lazyos --osquery-socket /var/osquery/osquery.em --osquery-startup-timeout 5s --osquery-query-timeout 30s --backend kernel
```

### Watching Logs

```bash
make watch-logs
```

Prompts for a log path (default `~/.local/state/lazyos/lazyos.log`). Press `Enter` to accept and begin tailing through `jq`:

```
Watching /home/user/.local/state/lazyos/lazyos.log — press Ctrl+C to stop.
```

Each JSON log line from the app is pretty-printed by `jq` in real time.

### Cleaning Up

```bash
make clean
```

Removes the `lazyos` binary and the entire `./build/` directory:

```
Cleaning up build artifacts...
Removed ./build directory.
```

The next `make run-sandbox` will prompt to re-download osqueryd.

```bash
make clean-default-logs
```

Removes `~/.local/state/lazyos/lazyos.log` if it exists.
