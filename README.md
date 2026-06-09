# LazyOS

Visualize, query, and audit your entire operating system and cloud infrastructure directly from the terminal.

LazyOS is a terminal-based interactive client that bridges the analytical power of osquery with a modern Text User Interface (TUI). Built with the Go Bubble Tea framework, it replaces complex schemas, verbose CLI flags, and manual filtering with an intuitive three-pane workspace: a discoverable table browser, a multi-line SQL editor, and a dynamic results inspector. Query kernel tables (processes, users, network), cloud resources (EC2, S3, IAM), or both simultaneously — with results cached in an embedded SQLite store for instant re-querying.

### Kernel Tables

![Kernel demo](assets/demo.gif)
*Browse kernel tables (processes, users), execute queries with automatic SQLite caching, toggle between line and table modes, and edit SQL in INSERT mode — all with vim-style modal keybindings.*

### Cloud Infrastructure (AWS)

![AWS demo](assets/demo2.gif)
*Cycle to the AWS backend with `B`, search for tables by name, autofill `SELECT` queries, and inspect cloud resources with the same TUI — EC2, S3, IAM, RDS, ECS, EKS, and more.*

---

## Prerequisites

- [Go](https://go.dev/) 1.21 or later
- [osquery](https://github.com/osquery/osquery) — an active `osqueryd` daemon providing an extension socket (typically `/var/osquery/osquery.em` or `/tmp/osquery.em`)
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — Go TUI framework (fetched via `go mod`)
- [SQLite](https://sqlite.org/) — embedded query cache and local store
- [osquery-go](https://github.com/osquery/osquery-go) — Go extension SDK for Thrift RPC
- [Thrift](https://thrift.apache.org/) — RPC protocol between the osquery daemon and the extension
- [AWS](https://aws.amazon.com/) — optional cloud tables (EC2, S3, IAM, RDS, etc.)
- [MCP](https://modelcontextprotocol.io/) — optional server for AI agent access
- [`jq`](https://jqlang.github.io/jq/) — required by `make watch-logs` for JSON log formatting

---

## Documentation

Setup, build, test, and operational guides are all available at:

**[ajitm722.github.io/LazyOS](https://ajitm722.github.io/LazyOS)**

---

## License

[MIT](https://github.com/ajitm722/LazyOS/blob/main/LICENSE)
