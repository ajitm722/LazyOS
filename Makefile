.PHONY: help build run run-with-defaults test test-verbose test-force test-integration test-integration-verbose test-coverage test-coverage-html clean format watch-logs clean-default-logs

# Default binary name
BINARY_NAME=lazyos

OSQUERY_BIN := ./build/osquery/osqueryd

help: ## Show this help menu
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

setup-sandbox: ## Interactively prompt to download and setup the local osquery daemon
	@if [ ! -f $(OSQUERY_BIN) ]; then \
		echo "LazyOS requires a local osquery daemon to run the sandbox."; \
		read -p "Do you want to download osquery into ./build/osquery? (y/N): " confirm; \
		if [ "$$confirm" != "y" ] && [ "$$confirm" != "Y" ]; then \
			echo "Download cancelled. Sandbox setup aborted."; \
			exit 1; \
		fi; \
		echo "Downloading osquery-5.11.0..."; \
		rm -rf ./build/osquery; \
		mkdir -p ./build/osquery; \
		curl -L -s -f https://pkg.osquery.io/linux/osquery-5.11.0_1.linux_x86_64.tar.gz | tar -xz -C ./build/osquery; \
		echo "Locating and moving osqueryd binary..."; \
		find ./build/osquery -type f -name "osqueryd" -exec mv {} $(OSQUERY_BIN) \; ; \
		chmod +x $(OSQUERY_BIN); \
		echo "Download complete."; \
	else \
		echo "Sandbox dependency (osqueryd) already exists. Skipping download."; \
	fi

build: ## Build the lazyos binary
	@echo "Building $(BINARY_NAME)..."
	@go build -o $(BINARY_NAME) ./cmd/lazyos

run-sandbox: setup-sandbox build ## Run LazyOS in an isolated sandbox with an ephemeral daemon
	@echo "Launching LazyOS in sandbox..."
	@./scripts/osquery_daemon_wrapper.sh $(OSQUERY_BIN) ./$(BINARY_NAME) --osquery-socket=LAZYOS_TEST_SOCKET
	@echo "The osquery daemon has been shut down, but the build is cached in ./build/ for faster next runs. Run 'make clean' to remove it."

test-integration: setup-sandbox ## Run integration tests (summary) against a live, ephemeral osquery daemon
	@echo "Running integration tests..."
	@./scripts/osquery_daemon_wrapper.sh $(OSQUERY_BIN) gotestsum --format pkgname -- -tags=integration ./internal/daemons/osquery/...
	@echo "The osquery daemon has been shut down, but the build is cached in ./build/ for faster next runs. Run 'make clean' to remove it."

test-integration-verbose: setup-sandbox ## Run integration tests with verbose output against a live, ephemeral osquery daemon
	@echo "Running integration tests (verbose)..."
	@./scripts/osquery_daemon_wrapper.sh $(OSQUERY_BIN) gotestsum --format standard-verbose -- -tags=integration ./internal/daemons/osquery/...
	@echo "The osquery daemon has been shut down, but the build is cached in ./build/ for faster next runs. Run 'make clean' to remove it."
	
run: ## Run LazyOS interactively, prompting for configuration flags
	@echo "Configuring LazyOS (press Enter to accept defaults)..."
	@echo ""
	@read -p "  Config File [~/.config/lazyos/config.yml]: " config_path; \
	config_path=$${config_path:-}; \
	read -p "  OSQuery Socket Path [/tmp/osquery.em]: " socket; \
	socket=$${socket:-/tmp/osquery.em}; \
	read -p "  Startup Timeout [2s]: " startup_timeout; \
	startup_timeout=$${startup_timeout:-2s}; \
	read -p "  Query Timeout [10s]: " query_timeout; \
	query_timeout=$${query_timeout:-10s}; \
	read -p "  Log File [~/.local/state/lazyos/lazyos.log]: " log_file; \
	log_file=$${log_file:-}; \
	read -p "  Keep Log File? (true/false) [false]: " keep_log; \
	keep_log=$${keep_log:-false}; \
	echo ""; \
	flags=""; \
	[ -n "$$config_path" ] && flags="$$flags --config=$$config_path"; \
	flags="$$flags --osquery-socket=$$socket"; \
	flags="$$flags --osquery-startup-timeout=$$startup_timeout"; \
	flags="$$flags --osquery-query-timeout=$$query_timeout"; \
	[ -n "$$log_file" ] && flags="$$flags --log-file=$$log_file"; \
	flags="$$flags --keep-log=$$keep_log"; \
	echo "  Running: lazyos$$flags"; \
	go run ./cmd/lazyos \
		$$( [ -n "$$config_path" ] && echo "--config=$$config_path" ) \
		--osquery-socket="$$socket" \
		--osquery-startup-timeout="$$startup_timeout" \
		--osquery-query-timeout="$$query_timeout" \
		$$( [ -n "$$log_file" ] && echo "--log-file=$$log_file" ) \
		--keep-log="$$keep_log"

run-with-defaults: ## Run LazyOS with default configuration (no prompts)
	@echo "Running LazyOS with default configuration..."
	@go run ./cmd/lazyos

test: ## Run tests (summary). Logger tests are fast, isolated integration — t.TempDir() + t.Setenv() leave zero state on the host.
	@echo "Running tests..."
	@gotestsum --format pkgname ./...

test-verbose: ## Run unit tests with verbose output
	@echo "Running tests (verbose)..."
	@gotestsum --format standard-verbose ./...

test-force: ## Run unit tests, ignoring the cache
	@echo "Running tests (uncached)..."
	@gotestsum --format pkgname -- -count=1 ./...

test-coverage: ## Run tests and display coverage in CLI (only packages with test files)
	@echo "Running tests with coverage..."
	@echo ""
	@echo "  Included (unit tests exist):"
	@echo "    - internal/daemons"
	@echo "    - internal/logger         (fast, isolated integration — t.TempDir() / t.Setenv())"
	@echo "    - internal/tui"
	@echo "    - internal/tui/views/querybar"
	@echo "    - internal/tui/views/results"
	@echo "    - internal/tui/views/sidebar"
	@echo ""
	@echo "  Omitted (no unit tests):"
	@echo "    - cmd/lazyos              (entry point, no logic to test)"
	@echo "    - internal/config         (types-only package, no logic)"
	@echo "    - internal/daemons/mock   (test helpers consumed by other tests)"
	@echo "    - internal/daemons/osquery    (requires live osquery socket; run via 'make test-integration')"
	@echo ""
	@gotestsum --format pkgname -- -cover $$(go list -f '{{if .TestGoFiles}}{{.ImportPath}}{{end}}' ./...)

test-coverage-html: ## Run tests and open HTML coverage report in browser (only packages with test files)
	@echo "Generating HTML coverage report..."
	@echo ""
	@echo "  Included (unit tests exist):"
	@echo "    - internal/daemons"
	@echo "    - internal/logger         (fast, isolated integration — t.TempDir() / t.Setenv())"
	@echo "    - internal/tui"
	@echo "    - internal/tui/views/querybar"
	@echo "    - internal/tui/views/results"
	@echo "    - internal/tui/views/sidebar"
	@echo ""
	@echo "  Omitted (no unit tests):"
	@echo "    - cmd/lazyos              (entry point, no logic to test)"
	@echo "    - internal/config         (types-only package, no logic)"
	@echo "    - internal/daemons/mock   (test helpers consumed by other tests)"
	@echo "    - internal/daemons/osquery    (requires live osquery socket; run via 'make test-integration')"
	@echo ""
	@gotestsum --format pkgname -- -coverprofile=coverage.out $$(go list -f '{{if .TestGoFiles}}{{.ImportPath}}{{end}}' ./...)
	@go tool cover -html=coverage.out

clean: ## Remove build artifacts and the downloaded osquery daemon
	@echo "Cleaning up build artifacts..."
	@rm -f $(BINARY_NAME)
	@if [ -d "./build" ]; then \
		rm -rf ./build; \
		echo "Removed ./build directory."; \
	fi

watch-logs: ## Tail and pretty-print a lazyos log file (prompts for path, defaults ~/.local/state/lazyos/lazyos.log)
	@echo "Configuring LazyOS (press Enter to accept defaults)..."; \
	echo "  NOTE: This path must match --log-file used in 'make run'.";\
	echo ""; \
	read -p "  Log File [~/.local/state/lazyos/lazyos.log]: " log_file; \
	log_file=$${log_file:-~/.local/state/lazyos/lazyos.log}; \
	echo "Watching $$log_file — press Ctrl+C to stop."; \
	mkdir -p "$$(dirname "$$log_file")"; \
	touch "$$log_file"; \
	trap 'echo "Log viewer closed."; exit 0' INT; \
	tail -f "$$log_file" | jq

clean-default-logs: ## Remove the default log file (~/.local/state/lazyos/lazyos.log)
	@log_file=~/.local/state/lazyos/lazyos.log; \
	if [ -f "$$log_file" ]; then \
		rm -f "$$log_file"; \
		echo "Removed $$log_file"; \
	else \
		echo "No log file found at $$log_file"; \
	fi

format: ## Format the code using go fmt
	@echo "Formatting code..."
	@go fmt ./...
