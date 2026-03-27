# Changelog

All notable changes to termiflow are documented here.

## [0.2.1.0] - 2026-03-27

### Added
- **termiflow-engine integration**: Intelligence layer (curator, scorer, summarizer, tagger, asker) now uses the `termiflow-engine` v0.1.0 library instead of inline code. Cleaner separation between CLI and AI pipeline.
- **Retry logic for managed mode**: Managed LLM and search providers retry on 429/5xx with exponential backoff and jitter. Respects `Retry-After` headers.
- **Offline detection**: `feed --refresh` detects DNS failures, connection refused, and timeouts. Shows "Offline — showing cached feed" instead of crashing.
- **`ask --save` flag**: Saves question, answer, and sources to a markdown file at `~/.local/share/termiflow/saved/`. Respects `XDG_DATA_HOME`.
- **`status` command**: Shows mode (managed/self-hosted), active subscriptions with last-fetch timestamps, database size, and config path.
- **`upgrade` command**: Checks GitHub releases for newer versions and shows upgrade instructions.
- **CLI version header**: Sends `X-Termiflow-Version` header on managed API calls for server-side analytics.

### Changed
- **Engine adapter pattern**: LLM and search providers implement `engine.LLMProvider` and `engine.SearchProvider` interfaces via thin adapters, decoupling CLI providers from the engine's type system.
- **Feed item mapping**: New `feeditem_mapper.go` converts between engine `FeedItem` and CLI database model, keeping DB concerns out of the engine.

### Fixed
- **Managed provider timeout**: HTTP client now has 120s timeout. Previously had no timeout, causing `ask --no-search` to hang indefinitely.
- **`unsubscribe` exit code**: Returns exit code 1 when topic doesn't exist (was returning 0).
- **`capitalize()` unicode safety**: Uses `unicode.ToUpper` + `utf8.DecodeRuneInString` instead of ASCII byte arithmetic. Handles non-ASCII characters correctly.

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
