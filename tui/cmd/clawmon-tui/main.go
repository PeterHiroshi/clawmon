// Package main is the entry point for the clawmon TUI dashboard.
package main

import (
	"flag"
	"fmt"
	"os"
)

const (
	// DefaultDaemonURL is the default daemon API base URL.
	DefaultDaemonURL = "http://127.0.0.1:9876/api/v1"
)

func main() {
	daemonURL := flag.String("daemon-url", DefaultDaemonURL, "clawmon daemon API base URL")
	flag.Parse()

	fmt.Fprintf(os.Stderr, "clawmon-tui: connecting to %s\n", *daemonURL)

	// TODO: Initialize client, create app model, run bubbletea program
	fmt.Println("clawmon TUI — not yet implemented")
}
