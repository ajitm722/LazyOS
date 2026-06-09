// Package main is the binary entry point for lazyos-mcp. It starts an
// MCP-over-HTTP server that exposes osquery tables and query execution
// to AI agents via the Model Context Protocol.
//
// The server uses Streamable HTTP transport and is designed to work with
// Tailscale for secure cross-network access without API keys or TLS.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ajitm722/LazyOS/internal/bootstrap"
	"github.com/ajitm722/LazyOS/internal/config"
	"github.com/ajitm722/LazyOS/internal/mcp"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := execute(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func execute(ctx context.Context) error {
	var cfg config.Config

	rootCmd := &cobra.Command{
		Use:   "lazyos-mcp",
		Short: "MCP server exposing osquery data to AI agents over HTTP",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			viper.SetEnvPrefix("LAZYOS")
			viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
			viper.AutomaticEnv()
			return viper.Unmarshal(&cfg)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServer(cmd.Context(), cfg)
		},
	}

	rootCmd.Flags().String("osquery-socket", "/tmp/osquery.em", "Path to the osquery extension socket")
	rootCmd.Flags().Duration("osquery-startup-timeout", 10*time.Second, "Timeout for the initial osquery Thrift connection")
	rootCmd.Flags().Duration("osquery-query-timeout", 100*time.Second, "Timeout for individual osquery queries")
	rootCmd.Flags().StringSlice("backend", []string{"kernel"}, "Backends to enable: kernel, aws")
	rootCmd.Flags().String("host", "127.0.0.1", "Bind address for the HTTP server")
	rootCmd.Flags().Int("port", 8080, "Bind port for the HTTP server")

	if err := viper.BindPFlags(rootCmd.Flags()); err != nil {
		return err
	}

	return rootCmd.ExecuteContext(ctx)
}

func runServer(ctx context.Context, cfg config.Config) error {
	// Resolve defaults for fields we need but don't come from flags directly.
	if cfg.OsquerySocket == "" {
		cfg.OsquerySocket = "/tmp/osquery.em"
	}

	backends, _, err := bootstrap.BootstrapBackends(cfg)
	if err != nil {
		return fmt.Errorf("failed to bootstrap backends: %w", err)
	}

	host := viper.GetString("host")
	port := viper.GetInt("port")
	addr := fmt.Sprintf("%s:%d", host, port)

	srv := mcp.New(backends, "")
	return srv.ListenAndServe(ctx, addr)
}
