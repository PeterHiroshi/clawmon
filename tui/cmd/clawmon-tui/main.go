// Package main is the entry point for the clawmon TUI dashboard.
package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/PeterHiroshi/clawmon/tui/internal/app"
	"github.com/PeterHiroshi/clawmon/tui/internal/client"
	"github.com/PeterHiroshi/clawmon/tui/internal/views"
)

func main() {
	daemonURL := flag.String("daemon-url", views.DefaultDaemonURL, "clawmon daemon API base URL")
	flag.Parse()

	fmt.Fprintf(os.Stderr, "clawmon-tui: connecting to %s\n", *daemonURL)

	c := client.NewHTTPClient(*daemonURL)
	m := app.NewModel(c)

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
