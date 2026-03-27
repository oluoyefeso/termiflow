# termiflow (self-hosted)

Terminal-native AI intelligence tool for developers. Subscribe to curated topic feeds and ask questions — all from the command line.

**Information comes to you where you already are — the terminal. No browser switching, no context loss, no noise. Just signal.**

> This is the **self-hosted edition**. Bring your own Anthropic and Tavily API keys.
> Want a managed key with no setup? Use [termiflow-cli](https://github.com/oluoyefeso/termiflow-cli) instead.

## Repos

| Repo | Description | Visibility |
|------|-------------|------------|
| **[termiflow](https://github.com/oluoyefeso/termiflow)** (this) | Self-hosted CLI, bring your own API keys | Public |
| **[termiflow-cli](https://github.com/oluoyefeso/termiflow-cli)** | Managed CLI, `tf_xxx` key from termiflow.com | Public |
| **termiflow-api** | Backend proxy (key management + LLM/search) | Private |

## Quick Install

```bash
# Using Go
go install github.com/oluoyefeso/termiflow/cmd/termiflow@latest

# Or build from source
git clone https://github.com/oluoyefeso/termiflow.git
cd termiflow
make build
```

## Quick Start

```bash
# Initial setup — enter your Anthropic, OpenAI, and/or Tavily keys
termiflow config

# Ask a question
termiflow ask "what are the latest advancements in 3nm chip fabrication?"

# Subscribe to topics
termiflow subscribe silicon-chips
termiflow subscribe "RISC-V in automotive" --weekly

# Check your feed
termiflow feed
termiflow feed --refresh        # fetch new items first
termiflow feed --watch          # stay alive, auto-refresh every 30m
```

## Features

### Ask Questions with AI-Powered Answers

```bash
termiflow ask "explain rust's borrow checker"
termiflow ask "compare TSMC N3 vs Intel 4" --sources 5
termiflow ask "what is WebGPU?" --provider local
```

### Subscribe to Curated Topic Updates

```bash
# Predefined categories
termiflow subscribe silicon-chips
termiflow subscribe rust-lang
termiflow subscribe llm-inference

# Custom topics
termiflow subscribe "quantum error correction" --daily
termiflow subscribe "RISC-V adoption" --weekly
```

### View Your Personalized Feed

```bash
termiflow feed                        # All unread items
termiflow feed --topic silicon-chips  # Filter by topic
termiflow feed --today                # Today's items
termiflow feed --refresh              # Fetch new items first
termiflow feed --watch                # Continuous mode (auto-refresh)
termiflow feed --watch --interval 5m  # Custom refresh interval
```

### Manage Subscriptions

```bash
termiflow topics                      # List all topics
termiflow topics --subscribed         # Your subscriptions
termiflow unsubscribe silicon-chips   # Remove subscription
```

### JSON Output for Scripting

```bash
termiflow feed --json | jq '.data.topics[].items[].title'
termiflow status --json | jq '.meta.version'
termiflow ask "what is WASM?" --json --no-search
termiflow changelog --json
```

### Release Notes

```bash
termiflow changelog                   # Latest 3 releases
termiflow changelog --all             # Full release history
```

## Configuration

Config file location: `~/.config/termiflow/config.toml`

```bash
# Interactive setup
termiflow config

# Set individual values
termiflow config set providers.openai.api_key YOUR_KEY
```

### Environment Variables

```bash
export TERMIFLOW_OPENAI_API_KEY=sk-...
export TERMIFLOW_ANTHROPIC_API_KEY=sk-ant-...
export TERMIFLOW_TAVILY_API_KEY=tvly-...
```

## LLM Providers

| Provider | Flag | Notes |
|----------|------|-------|
| OpenAI (default) | `--provider openai` | GPT-4o |
| Anthropic | `--provider anthropic` | Claude models |
| Local | `--provider local` | Ollama, llama.cpp, LM Studio |

## Predefined Topics

| Topic | Description |
|-------|-------------|
| `silicon-chips` | Chip fabrication, lithography, semiconductor industry |
| `rust-lang` | Rust language updates, crates, ecosystem |
| `llm-inference` | LLM optimization, inference, AI deployment |
| `webgpu` | WebGPU, browser graphics, GPU compute |
| `systems-programming` | OS development, compilers, low-level |
| `kubernetes` | K8s, containers, cloud-native |

## Development

```bash
# Build CLI
make build

# Build with mock providers (no real API calls)
make build-mock

# Run tests
make test

# Run with mock mode (zero network, no API keys needed)
make run-mock ARGS="feed --refresh"
TERMIFLOW_MOCK=true go run -tags mock ./cmd/termiflow ask "rust async"
```

## Docker

```bash
docker build -t termiflow .
docker run -it --rm \
  -e TERMIFLOW_ANTHROPIC_API_KEY=sk-ant-... \
  -e TERMIFLOW_TAVILY_API_KEY=tvly-... \
  -v ~/.config/termiflow:/home/termiflow/.config/termiflow \
  termiflow ask "your question"
```

## License

GPL License - see [LICENSE](LICENSE) for details.
