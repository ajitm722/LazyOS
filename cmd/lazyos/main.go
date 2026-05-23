// Package main is the binary entry point for lazyos. It sets up OS signal
// handling and delegates to the Cobra-based Execute function.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

// main creates a context that is cancelled on SIGINT or SIGTERM, then hands
// control to the Cobra command tree defined in root.go.
func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := Execute(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
