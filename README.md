# termiflow

Terminal-native AI intelligence tool that lets developers ask questions and subscribe to curated topic updates, all from the command line.

**Information comes to you where you already are — the terminal. No browser switching, no context loss, no noise. Just signal.**

## Quick Install

```bash
# Using Go
go install github.com/termiflow/termiflow/cmd/termiflow@latest

# Or build from source
git clone https://github.com/termiflow/termiflow.git
cd termiflow
make build
```

## Quick Start

```bash
# Initial setup
termiflow config init

# Ask a question
termiflow ask "what are the latest advancements in 3nm chip fabrication?"

# Subscribe to topics
termiflow subscribe silicon-chips
termiflow subscribe "RISC-V in automotive" --weekly

# Check your feed
termiflow feed
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
```

### Manage Subscriptions

```bash
termiflow topics                      # List all topics
termiflow topics --subscribed         # Your subscriptions
termiflow unsubscribe silicon-chips   # Remove subscription
```

## Configuration

Config file location: `~/.config/termiflow/config.toml`

```bash
# Interactive setup
termiflow config init

# View current config
termiflow config

# Edit config
termiflow config --edit

# Set individual values
termiflow config set providers.openai.api_key YOUR_KEY
```

### Environment Variables

```bash
export TERMFLOW_OPENAI_API_KEY=sk-...
export TERMFLOW_ANTHROPIC_API_KEY=sk-ant-...
export TERMFLOW_TAVILY_API_KEY=tvly-...
```

## Predefined Topics

| Topic | Description |
|-------|-------------|
| `silicon-chips` | Chip fabrication, lithography, semiconductor industry |
| `rust-lang` | Rust language updates, crates, ecosystem |
| `llm-inference` | LLM optimization, inference, AI deployment |
| `webgpu` | WebGPU, browser graphics, GPU compute |
| `systems-programming` | OS development, compilers, low-level |
| `kubernetes` | K8s, containers, cloud-native |

## LLM Providers

termiflow supports multiple LLM providers:

- **OpenAI** (default) - GPT-4o and other models
- **Anthropic** - Claude models
- **Local** - Any OpenAI-compatible server (Ollama, llama.cpp, LM Studio)

```bash
# Use specific provider
termiflow ask "question" --provider anthropic
termiflow ask "question" --provider local
```

## Docker

```bash
# Build
docker build -t termiflow .

# Run
docker run -it --rm \
  -e TERMFLOW_OPENAI_API_KEY=$OPENAI_API_KEY \
  -v ~/.config/termiflow:/home/termiflow/.config/termiflow \
  -v ~/.local/share/termiflow:/home/termiflow/.local/share/termiflow \
  termiflow ask "your question"
```

## Development

```bash
# Build
make build

# Run in development
make dev ARGS="ask 'test question'"

# Run tests
make test

# Build for all platforms
make release
```

## License

MIT License - see [LICENSE](LICENSE) for details.
