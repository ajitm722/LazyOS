---
sidebar_position: 1
---

# LazyOS

Visualize, query, and audit your entire operating system and cloud infrastructure directly from the terminal.

LazyOS is a terminal-based interactive client that bridges the analytical power of osquery with a modern Text User Interface (TUI). Built with the Go Bubble Tea framework, it replaces complex schemas, verbose CLI flags, and manual filtering with an intuitive three-pane workspace: a discoverable table browser, a multi-line SQL editor, and a dynamic results inspector. Query kernel tables (processes, users, network), cloud resources (EC2, S3, IAM), or both simultaneously — with results cached in an embedded SQLite store for instant re-querying.

### Kernel Tables

![Kernel demo](/img/demo.gif)
*Browse kernel tables (processes, users), execute queries with automatic SQLite caching, toggle between line and table modes, and edit SQL in INSERT mode — all with vim-style modal keybindings.*

### Cloud Infrastructure (AWS)

![AWS demo](/img/demo2.gif)
*Cycle to the AWS backend with `B`, search for tables by name, autofill `SELECT` queries, and inspect cloud resources with the same TUI — EC2, S3, IAM, RDS, ECS, EKS, and more.*

---

## Documentation Structure

| Section | Contents |
|---|---|
| **Architecture** | Domain model, caching design, system components, and internal package structure |
| **Development** | Unit tests, integration tests, Makefile targets, and build instructions |
| **Operations** | Remote deployment, MCP Server, Diagnostic Query Packs |
| **Reference** | UI key bindings, Osquery Interaction, and Sequence Flows |
| **API** | Auto-generated Go package documentation |

---

## Build and Test

LazyOS provides a `Makefile` that covers building, running, testing, log-watching, and cleanup.

| Target | Purpose |
|---|---|
| `make run-sandbox` | **Primary entry point.** Downloads osqueryd, builds binary, and runs inside kernel sandbox |
| `make run-sandbox-all` | Run sandbox with **both** kernel and AWS cloud backends enabled |
| `./lazyos-osquery.sh` | Ephemeral evaluation mode — downloads everything to `/tmp`, runs, and wipes all traces on exit |
| `make run` | Interactive prompt for all config flags (requires an existing osquery daemon) |
| `make run-with-defaults` | Run immediately with compiled-in defaults |
| `make watch-logs` | Tail and pretty-print the log file via `jq` |
| `make build` | Build a standalone `lazyos` binary |
| `make test` | Run all tests (summary) |
| `make test-force` | Run all tests, bypassing cache |
| `make test-coverage` | Run tests with per-package coverage report |
| `make test-coverage-html` | Generate and open HTML coverage report |
| `make test-integration` | Run integration tests against live osquery daemon |
| `make test-integration-verbose` | Run integration tests with verbose output |
| `make clean` | Remove the `lazyos` binary and the entire `./build/` directory |
| `make clean-default-logs` | Remove `~/.local/state/lazyos/lazyos.log` |

---

## Workflows

### Browsing Schema

Navigate through all available osquery tables. LazyOS reads the exact schema from your active osquery instance and presents it in a scrollable sidebar.

Results can be viewed in two modes, toggled with `Ctrl+N`:

| Mode | Screenshot |
|---|---|
| **Table mode** — columns for structured scanning. Truncates when too many to fit the pane width. | ![Schema browsing in table mode](/img/schema_browsing_tablemode.png) |
| **Line mode** — scrollable key-value pairs. Essential for wide schemas where table mode would omit data. | ![Schema browsing in line mode](/img/schema_browsing_linemode.png) |

### Querying Data

Enter raw SQL to fetch data from your local machine. The example below lists the most memory-intensive running processes, converting raw byte values into a human-readable format:

![Query execution](/img/query_execution_holder.png)
*Identifying the most memory-intensive processes with `resident_size` and `total_size` formatted as MB.*

### Compliance Checks

LazyOS can also be valuable as a continuous compliance verification tool. The queries below map directly to hardened security benchmarks (CIS, etc.) and can be executed on demand or integrated into audit workflows.

#### CIS Kernel Module Blacklist

CIS Benchmarks for Linux require disabling obsolete filesystem modules (`cramfs`, `freevxfs`, `jffs2`, `hfs`, `hfsplus`, `squashfs`, `udf`) and uncommon network protocols (`dccp`, `sctp`, `rds`, `tipc`) to reduce the kernel's attack surface. We can query `kernel_modules` to verify that none of these forbidden modules are loaded. If the query returns any rows, the host fails the check.

![Kernel module compliance check](/img/kernel_module_compliance.png)
*Querying `kernel_modules` for blacklisted modules — rows present means non-compliance.*

#### Publicly Exposed Services

By joining `listening_ports` with `processes` on `pid`, we can identify every service listening on `0.0.0.0` or `::` (all interfaces) — meaning it is potentially reachable from the outside network. The result includes the process name, port, bound address, and binary path, making it easy to audit unintended exposure.

![Publicly exposed services](/img/publicly_exposed_service.png)
*Services listening on all interfaces — potential network exposure surface.*

#### Suspicious Process Execution

Replaces manual `/tmp` auditing. Processes running from world-writable directories (`/tmp`, `/dev/shm`) are a common indicator of malicious binaries or misconfigured scripts. LazyOS queries `processes` with a path filter to surface any process whose binary lives in these locations — no more grepping through `ps aux` output.

![Suspicious process execution](/img/suspicious_process_execution.png)
*Processes running from writable temporary directories — potential security concern.*

### AI-Assisted Query Generation

Not sure how to write the SQL for a compliance check or an ad-hoc investigation? AI agents can integrate easily with LazyOS. Describe what you need in plain English, and the agent generates the osquery SQL on the spot.

The example below shows a natural-language request for inode usage:

| Step | Screenshot |
|---|---|
| Asking the agent about free inodes | ![Asking opencode about free inodes](/img/ask_opencode_freeinodes.png) |
| Agent's generated SQL response | ![opencode response with SQL](/img/opencode_freeinodes_resp.png) |

This workflow keeps you in the terminal — no context-switching to a browser, no digging through osquery docs. Just describe the question and execute the result.

---

## Full Documentation

For detailed information about architecture, development, operations, and API reference, visit the [Architecture Overview](/docs/architecture/overview).

Or explore the [GitHub repository](https://github.com/ajitm722/LazyOS).
