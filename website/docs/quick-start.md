---
sidebar_position: 2
---

# Quick Start

## Ephemeral Evaluation

The fastest way to try LazyOS without installing anything:

```bash
curl -sL https://raw.githubusercontent.com/ajitm722/LazyOS/main/lazyos-osquery.sh | bash
```

Downloads everything into `/tmp` — osqueryd, the LazyOS binary, and the sandbox wrapper — and opens the TUI. When you quit, every file and process is wiped. No clone, no install, no traces.

```
======================================================
 LazyOS (Ephemeral Evaluation Mode)
======================================================
This script will download everything into memory/tmp.
When you close the app, NO binaries, NO daemons, and
NO logs will be left on your machine.
======================================================
[*] Fetching osquery (5.11.0)...
[*] Fetching LazyOS binary...
[*] Fetching Daemon Wrapper...
[*] Ephemeral Sandbox Active: Zero bloat...
[*] Sandbox completely wiped. Goodbye!
```

> **Note:** Unlike `make run-sandbox`, this does **not** cache `./build/`. Every run is a fresh ephemeral session.
>
> **Note:** `lazyos-osquery.sh` currently works **only on Linux**. The download URL and binary extraction paths are Linux-specific.

## Persistent Setup

For development or repeated use with SQLite caching:

```bash
git clone https://github.com/ajitm722/LazyOS.git
cd LazyOS
make run-sandbox
```

First run:

```
LazyOS requires a local osquery daemon to run the sandbox.
Do you want to download osquery into ./build/osquery? (y/N): y
Downloading osquery-5.11.0...
Locating and moving osqueryd binary...
Download complete.
Building lazyos...
Launching LazyOS in sandbox...
Starting ephemeral osqueryd...
Socket ready.
```

Subsequent runs (osqueryd already cached):

```
Sandbox dependency (osqueryd) already exists. Skipping download.
Building lazyos...
Launching LazyOS in sandbox...
Starting ephemeral osqueryd...
Socket ready.
```

### With Cloud Backends

To include AWS cloud infrastructure tables alongside kernel tables:

```bash
make run-sandbox-all
```

This enables both the `kernel` and `aws` backends, exposing EC2, S3, IAM, RDS, ECS, EKS, and other cloud resource tables via the sidebar. Use `B` to cycle between backends.

> **AWS requires cloudquery** — the AWS tables are provided by the [cloudquery](https://github.com/ajitm722/cloudquery) osquery extension. See the [cloudquery configuration guide](/docs/reference/osquery-interaction#cloudquery-cloud-resource-instrumentation) for setup instructions.

---

## Running with Existing Daemon

If you already have an osqueryd instance running:

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

Or skip prompts entirely:

```bash
make run-with-defaults
```

Or run the binary directly:

```bash
./lazyos --osquery-socket /var/osquery/osquery.em --osquery-startup-timeout 5s --osquery-query-timeout 30s --backend kernel
```

---

## Watching Logs

```bash
make watch-logs
```

Prompts for a log path (default `~/.local/state/lazyos/lazyos.log`). Press `Enter` to accept and begin tailing through `jq`:

```
Configuring LazyOS (press Enter to accept defaults)...

  NOTE: This path must match --log-file used in 'make run'.

  Log File [~/.local/state/lazyos/lazyos.log]:

Watching /home/user/.local/state/lazyos/lazyos.log — press Ctrl+C to stop.
```

Each JSON log line from the app is pretty-printed by `jq` in real time. Press `Ctrl+C` to stop — prints `Log viewer closed.` cleanly.

---

## Cleaning Up

```bash
make clean              # Remove binary and ./build/ directory
make clean-default-logs # Remove ~/.local/state/lazyos/lazyos.log
```

The next `make run-sandbox` will prompt to re-download osqueryd.
