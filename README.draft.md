# termiflow

Terminal-native AI intelligence engine. Subscribe to any concept. Get cross-cutting intelligence from every source. Delivered to your terminal, sharper every day.

**The feed that learns you.**

> Termiflow is not a news reader. It's an intelligence engine that maps your intent to a global
> data stream, scores every article against what you actually care about, and gets more precise
> the more you use it.

## Install

```bash
# Go install (requires Go 1.24+)
go install github.com/oluoyefeso/termiflow/cmd/termiflow@latest

# Homebrew (macOS + Linux)
brew install oluoyefeso/tap/termiflow

# Build from source
git clone https://github.com/oluoyefeso/termiflow.git
cd termiflow && make build
```

## Quick Start

```bash
# Set up your API keys (interactive)
termiflow config

# Subscribe to a concept
termiflow subscribe "things affecting silicon chip prices"

# Check your feed
termiflow feed --refresh

# Launch the full TUI
termiflow

# Ask a question with AI-powered answers
termiflow ask "what are the latest advancements in 3nm chip fabrication?"
```

## Two Modes

| Mode | Setup | How It Works |
|------|-------|-------------|
| **Managed** | Get a `tf_xxx` key from [termiflow.com](https://termiflow.com) | Zero config. We handle the AI/search APIs. |
| **Self-hosted** | Bring your own Anthropic + Tavily keys | Full data sovereignty. Nothing leaves your machine. |

Both modes use the same binary. Run `termiflow config` to choose.

## Features

### Interactive TUI
Launch with bare `termiflow`. Full-screen terminal UI with 6 screens:
- **Dashboard** — subscriptions, unread counts, status
- **Feed List** — j/k navigation, read/unread, relevance scores
- **Article Detail** — full summary, tags, source URL
- **Ask** — inline Q&A with streaming AI responses
- **Topics Browser** — subscribe/unsubscribe/edit inline
- **Status** — config, provider, DB stats

### CLI Commands
All commands also work as traditional CLI output (no TUI required):

| Command | Description |
|---------|-------------|
| `termiflow` | Launch interactive TUI |
| `termiflow feed` | Print feed to stdout |
| `termiflow feed --refresh` | Fetch new items first |
| `termiflow feed --watch` | Continuous mode (auto-refresh) |
| `termiflow ask "question"` | AI-powered Q&A with sources |
| `termiflow subscribe <topic>` | Subscribe to a topic |
| `termiflow topics` | List all topics |
| `termiflow config` | Interactive setup |
| `termiflow status` | Show config and stats |
| `termiflow changelog` | Release notes |
| `termiflow upgrade` | Check for updates |

### JSON Output
Every command supports `--json` for scripting:
```bash
termiflow feed --json | jq '.data.topics[].items[].title'
termiflow status --json | jq '.meta.version'
```

### Themes
```bash
termiflow config set theme amber     # default
termiflow config set theme dracula
termiflow config set theme light
```

### LLM Providers
| Provider | Flag | Notes |
|----------|------|-------|
| OpenAI (default) | `--provider openai` | GPT-4o |
| Anthropic | `--provider anthropic` | Claude |
| Local | `--provider local` | Ollama, llama.cpp, LM Studio |

## Architecture

```
termiflow (single Go binary)
├── cmd/termiflow/         # Entry point
├── internal/
│   ├── cli/               # Cobra command implementations
│   ├── tui/               # Bubble Tea TUI (6 screens)
│   ├── ui/                # Output formatting, themes, colors
│   ├── db/                # SQLite models + migrations
│   ├── providers/         # LLM + search backends
│   ├── scheduler/         # Subscription refresh scheduling
│   ├── notifications/     # Announcement system
│   └── sources/           # Content source definitions
├── pkg/models/            # Shared data types
└── termiflow-engine       # AI curation library (external dep)
```

All intelligence runs locally. The managed-mode backend (`termiflow-api`) is purely
a proxy — it validates your key and forwards requests. No curation, scoring, or
summarization happens server-side.

## Contributing

See [GOAL-STATE.md](GOAL-STATE.md) for the full roadmap, gap analysis, and phased tasks.

### Dev Setup
```bash
git clone https://github.com/oluoyefeso/termiflow.git
cd termiflow
make build          # Build the binary
make test           # Run tests
make build-mock     # Build without real API calls (no keys needed)
make run-mock ARGS="feed --refresh"
```

### Mock Mode
Develop and test without any API keys:
```bash
TERMIFLOW_MOCK=true go run -tags mock ./cmd/termiflow ask "rust async"
```

### Before Pushing
```bash
make test
golangci-lint run    # CI catches gofmt issues that go vet misses
```

### Key Conventions
- Bloomberg terminal aesthetic: amber primary, double-line headers, uppercase labels
- All new commands must support `--json` output
- TUI screens implement `tea.Model` (Init, Update, View)
- Provider interfaces live in termiflow-engine, implementations here

## Docker

```bash
docker build -t termiflow .
docker run -it --rm \
  -e TERMIFLOW_ANTHROPIC_API_KEY=sk-ant-... \
  -e TERMIFLOW_TAVILY_API_KEY=tvly-... \
  -v ~/.config/termiflow:/home/termiflow/.config/termiflow \
  termiflow ask "your question"
```

## Related Repos

| Repo | Description | Visibility |
|------|-------------|------------|
| **[termiflow](https://github.com/oluoyefeso/termiflow)** (this) | CLI + TUI | Public |
| **[termiflow-engine](https://github.com/oluoyefeso/termiflow-engine)** | AI curation library | Public |
| **[termiflow.com](https://termiflow.com)** | Marketing website | Public |
| **termiflow-api** | Backend proxy | Private |

## License

GPL
