# Changelog

All notable changes to termiflow are documented here.

## [0.2.0.0] - 2026-03-25

### Added
- **Managed mode**: Single `TERMIFLOW_API_KEY` (`tf_xxx`) activates managed mode — CLI routes LLM and search through the termiflow backend, no Anthropic/Tavily key needed
- **Backend API service** (`cmd/termiflow-api`): Thin proxy with SQLite-backed key auth, admin endpoints for key issuance/revocation, WAL mode for concurrent reads
- **Mock mode** (`-tags mock`): Zero-network development mode. `TERMIFLOW_MOCK=true` uses prompt-aware mock LLM (distinguishes relevance/summarize/tags calls) and timestamped mock search results
- **`feed --watch` flag**: Continuous mode — stays alive and auto-refreshes on interval (default 30m, configurable via `--interval`)
- **Managed config onboarding**: `config init` detects `tf_xxx` key input and writes managed config, skipping Anthropic/Tavily prompts

### Changed
- **Bloomberg terminal UI**: Amber primary color, double-line `═══` headers, uppercase section labels with `▸`, compact `[tag]` brackets, dot dividers — denser and more information-forward
- **`ask` command**: Uses `GetSearchProvider()` factory — works in managed and mock modes without Tavily key
- **Search provider factory**: `GetSearchProvider()` centralizes mock → managed → tavily → rss selection

### Fixed
- `feed --refresh` showing "No new items" despite fetching: scheduler now returns only actually-inserted items (not all curator output)
- Mock providers returning wrong content type: MockProvider now detects call type from prompt keywords and returns appropriate response (score / summary / tags / answer)

## [0.1.0.0] - 2026-03-01

### Added
- Initial release: `ask`, `feed`, `subscribe`, `topics`, `unsubscribe`, `config` commands
- Self-hosted mode with OpenAI, Anthropic, local LLM support
- Tavily search integration and RSS feed fetching
- SQLite-backed subscription and feed item storage
- Curator pipeline: relevance scoring, summarization, tag extraction
