# clawmon

Terminal-based monitoring dashboard for [OpenClaw](https://openclaw.ai) AI agents.

Track workspace sync, task dispatch, process health, and activity timeline — all in your terminal.

## Architecture

```
┌─────────────────┐     HTTP/SSE      ┌──────────────────┐
│   Go TUI        │ ◄──────────────── │   Rust Daemon    │
│   (Bubble Tea)  │                   │   (axum + tokio) │
│                 │                   │                  │
│  ┌───────────┐  │                   │  ┌────────────┐  │
│  │ Dashboard  │  │                   │  │ Git        │  │
│  │ Tasks     │  │                   │  │ Tasks      │  │
│  │ Git       │  │                   │  │ Processes  │  │
│  │ Activity  │  │                   │  │ Env Health │  │
│  └───────────┘  │                   │  └────────────┘  │
└─────────────────┘                   └──────────────────┘
```

- **Rust daemon** — collects git status, task metadata, process health, environment state. Exposes a local HTTP + SSE API.
- **Go TUI** — connects to daemon, renders a tabbed dashboard with real-time updates.

## Status

🚧 Under active development. See [feature-list.md](feature-list.md) for progress.

## License

MIT
