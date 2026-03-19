// Package main is the entry point for the clawmon TUI dashboard.
package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/PeterHiroshi/clawmon/tui/internal/app"
	"github.com/PeterHiroshi/clawmon/tui/internal/client"
	"github.com/PeterHiroshi/clawmon/tui/internal/config"
	"github.com/PeterHiroshi/clawmon/tui/internal/views"
)

func main() {
	daemonURL := flag.String("daemon-url", "", "clawmon daemon API base URL")
	flag.Parse()

	// Resolve daemon URL: CLI flag > config file > default
	resolvedURL := resolveDaemonURL(*daemonURL)

	fmt.Fprintf(os.Stderr, "clawmon-tui: connecting to %s\n", resolvedURL)

	c := client.NewHTTPClient(resolvedURL)
	m := app.NewModel(c)

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// resolveDaemonURL determines the daemon URL from CLI flag, config file, or default.
func resolveDaemonURL(cliURL string) string {
	if cliURL != "" {
		return cliURL
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to load config: %v\n", err)
	}

	if cfg != nil {
		if url := cfg.GetDaemonURL(); url != "" {
			return url
		}
	}

	return views.DefaultDaemonURL
}
