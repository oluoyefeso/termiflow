# Changelog

All notable changes to termiflow are documented here.

## [0.3.9.0] - 2026-03-28 — Concurrent Subscription Refresh

### Changed
- **Concurrent subscription refresh**: TUI and CLI now refresh up to 2 subscriptions concurrently instead of sequentially. With 5 subscriptions, wall-clock time is roughly halved. Per-topic progress messages still appear in real-time.
- **CLI refresh concurrency reduced from 5 to 2**: Bounds burst LLM calls at ~20 max (2 subs × 5 articles × 2 calls) to stay within managed API rate limits.

### Added
- **`RefreshAllSubscriptionsConcurrent()`**: New scheduler method with bounded concurrency via channel semaphore, progress callback for UI updates, panic recovery in goroutines, and `maxConcurrent <= 0` guard against deadlock.

## [0.3.8.0] - 2026-03-28 — Styling Refresh

### Changed
- **Color palette aligned with termiflow.com**: Secondary text now uses warm tan (`#d7c3ae`) instead of cool gray. Muted text, cyan, and success green tuned to match website hex values.
- **`>>` section headers**: LabeledRule and section headers use `>> LABEL ═══` format in amber instead of `── Label ──` in muted gray. New `SectionHeader` function for `>> LABEL // CATEGORY: VALUE` format.
- **Left border accents on feed items**: Unread items show amber `│` left border, read items show muted border. Replaces the old dot-based read indicators.
- **Uppercase formatting**: Topic names, article titles, metadata labels, and tags now render in uppercase for the intelligence terminal aesthetic.
- **Italic summaries**: Feed item summaries and detail view content use italic warm muted styling for clear visual hierarchy.
- **Structured metadata**: Source and timestamp display as `SOURCE: NAME    TIMESTAMP: TIME` in uppercase instead of `source · time`.
- **Tags without brackets**: Tags render as bare uppercase words with double spacing instead of `[tag]` format.
- **Dash separators between feed items**: Items separated by `───` dash rules instead of blank lines.
- **Warm muted secondary text**: Frequency, last-fetch time, status bar descriptions, and position indicators use warm tan instead of cool gray.

## [0.3.7.0] - 2026-03-28 — Custom Sources (RSS/Blog Subscriptions)

### Added
- **Custom source subscriptions**: Follow specific blogs and RSS feeds with AI-curated summaries. `termiflow source add <url>` runs RSS autodiscovery, resolves the feed URL, and creates a subscription that flows through the same LLM curation pipeline as topic-based search.
- **RSS autodiscovery** (`internal/sources/discover.go`): Given any blog URL, detects RSS/Atom feeds via Content-Type, `<link rel="alternate">` tags, or common path probing (`/feed`, `/atom.xml`, `/rss`). Falls back to web scraping if no feed found. 15-second timeout budget.
- **`termiflow source` CLI commands**: `source add <url>` (with `--context` and frequency flags), `source list`, `source remove <name-or-url>`.
- **Scheduler source-type branching**: `RefreshSubscription()` now branches on `source_type`: "feed" fetches RSS via gofeed, "scrape" extracts content via the web scraper, "topic" uses the existing Tavily search flow. HTML content is stripped and truncated (2000 runes, UTF-8 safe) before LLM processing.
- **Data model**: 4 new columns on `subscriptions` table (`source_url`, `source_type`, `display_name`, `context`) + unique index on `source_url`. Idempotent migration (handles existing databases).
- **TUI source display**: Source subscriptions appear in the topics browser with `[feed]` or `[scrape]` tag. Display name from feed metadata instead of raw URL.
- **Sync integration**: `serverSubscription` struct extended with source fields for managed-mode push/pull.
- **10 autodiscovery tests** + 12 scheduler/model tests covering HTML stripping, dedup, scoring topic resolution, and source type detection.

### Changed
- **Header endpoint label**: Managed-mode TUI header now shows "termiflow cloud" instead of the raw API URL.

### Fixed
- **UTF-8 safe content truncation**: RSS content truncation uses rune slicing instead of byte slicing to prevent splitting multi-byte characters.
- **Feed detection false positives**: `looksLikeFeed()` now requires `<feed>` or `<feed ` (with delimiter) instead of substring match, preventing HTML class names like `<div class="feed-widget">` from being detected as Atom feeds.
- **Source remove safety**: `source remove` uses exact display_name or domain match instead of substring match, preventing accidental deletion of wrong subscriptions.
- **Source deletion sync**: Removing a source subscription now pushes the deletion to the managed-mode server, preventing resurrection on next sync pull.

## [0.3.6.0] - 2026-03-28 — Server-Side Sync Infrastructure

### Added
- **Shared `ManagedClient`** (`internal/providers/managed_client.go`): Extracts auth headers, version header, retry logic, and JSON helpers into one shared HTTP client. Used by LLM proxy, search proxy, and sync. Eliminates DRY violation across three managed-mode HTTP clients.
- **Sync package** (`internal/sync/`): Server-side data sync for managed-mode users. Auto-syncs subscriptions, feed items, and read state to `api.termiflow.com` so switching devices preserves all data.
  - Full-state subscription sync (pull all, diff locally, union merge)
  - Incremental feed item sync (push after curation, pull since last sync)
  - Offline read-state buffer (`pending_read_sync` table) replays on reconnect
  - 30-minute staleness check on every CLI command
- **Sync hooks**: Subscribe, unsubscribe, and feed refresh automatically push to server. Root command pulls on startup. All gated by `IsManagedMode()`, zero impact on self-hosted users.
- **`db.IsOpen()`**: Nil-safe check for DB initialization state.
- **12 sync tests** covering nil safety, staleness logic, server call verification, and network error resilience.

### Changed
- **`ManagedProvider`** refactored to use shared `ManagedClient` instead of owning its own `http.Client` + auth headers.
- **`ManagedSearchProvider`** refactored to use shared `ManagedClient`. Removes `llm` package import (was only needed for `CLIVersion`).
- **`CLIVersion`** moved from `llm` package to `providers` package (shared concern).

## [0.3.5.0] - 2026-03-28 — API Health Indicator

### Added
- **Health status indicator for managed-mode users**: Green/red/gray dot in the TUI header and CLI output (`termiflow status`, `termiflow feed`) shows whether `api.termiflow.com` is healthy, degraded, or unreachable.
- **`CheckHealth()`**: 3-second timeout HTTP GET to `/health` endpoint with response body size limit (4KB). Returns `"ok"` or `"degraded"`.
- **`CheckHealthCached()`**: Stale-while-revalidate cache in `~/.cache/termiflow/`. Always returns cached value immediately; refreshes in background goroutine when stale (>5 min). Only blocks on first-ever call (no cache file). Prevents CLI latency.
- **TUI health tick**: Independent 5-minute timer re-checks API health in the background. Separate from the 30-minute auto-refresh timer.
- **`--json` support**: `termiflow status --json` includes `api_status` field for programmatic health checks.
- **12 unit tests** in `health_test.go` covering happy path, degraded, timeout, non-200, malformed JSON, empty status, and all cache states (fresh, stale, missing, corrupt).

### Fixed
- **TUI shares disk cache with CLI**: `checkAPIHealthCmd` uses `CheckHealthCached` so TUI and CLI see consistent health status from the same cache file.
- **Cache key normalization**: Empty `baseURL` is resolved to the default before hashing, preventing separate cache entries for the same effective endpoint.

## [0.3.4.0] - 2026-03-28 — Rich ASCII Header + Status Card Fix

### Added
- **ASCII art header on every screen**: Box-drawing style "TERMIFLOW" logo with mode, endpoint, subscription count, unread badge, refresh status, and version displayed alongside. Replaces the sparse one-line header.
- **Narrow terminal fallback**: Terminals under ~50 columns get a compact text-only header instead of the ASCII logo to prevent layout corruption.
- **`HeaderInfo` struct**: Config values (mode, endpoint, version) are read once at TUI init instead of on every render frame, eliminating unnecessary config reads on the hot path.

### Fixed
- **Status card right-border clipping**: Long file paths (database, config) that exceeded the card's inner width now truncate with `...` instead of pushing the right border off-screen.
- **Card truncation guard**: `RenderCard` no longer attempts truncation when `contentArea < 3`, preventing garbled output on very small terminals.
- **Logo width measurement**: Now computes max visual width across all 3 logo lines instead of assuming line 0 is widest.

## [0.3.3.0] - 2026-03-28 — errcheck Linter + Ask Context Injection

### Added
- **errcheck linter re-enabled**: Catches unchecked error returns in Go code. Configured with exclude-functions for common patterns (defer Close, test helpers, JSON encoding). Prevents new code from silently ignoring errors.
- **User context in CLI ask command**: The `termiflow ask` command now injects subscription state (topics, unread counts, mode) into the LLM system prompt, matching the TUI behavior. Ask "how many subscriptions do I have?" and get a real answer.
- **Shared `db.BuildUserContext()` function**: Extracted from TUI-only code to `internal/db/context.go` so both CLI and TUI ask commands use the same context builder. No duplication.

## [0.3.2.1] - 2026-03-28 — Persistent Announcements

### Changed
- **Announcements show every run until they expire**: Previously, announcements were dismissed after a single view. Now they persist on every CLI/TUI invocation until the server removes or expires them. Version update banners still show once per version.

## [0.3.2.0] - 2026-03-28 — Notification Banner Fixes

### Fixed
- **Dev builds never showed announcement banners**: `GetBanners()` returned empty when version was "dev". Dev builds now show API announcements while skipping version-comparison banners.
- **Background fetch goroutine orphaned on exit**: Process exited before `FetchAsync` completed, so the cache file was never written. Added `sync.WaitGroup` to wait for completion in `PersistentPostRun`.
- **TUI never displayed banners on first run**: Stale-while-revalidate model showed banners from previous cache, but TUI is long-running and only reads banners at startup. TUI now fetches synchronously (2s timeout) when cache is stale.
- **Incremental `&since=` wiping announcements from cache**: `fetchFromAPI` sent `since=lastFetchTime`, causing the API to return only newer announcements. Full cache was replaced with this subset, losing existing announcements. Removed `since` param; always fetch all active announcements.
- **Version string not URL-encoded in API request**: Semver `+` metadata could inject query params. Now uses `url.QueryEscape`.
- **Expired announcements shown from stale cache**: `GetBanners()` now filters out announcements past their `expires_at` timestamp.
- **Stale flag never reset after successful fetch**: `loadCacheFile()` only set `m.stale = true` on failure, never reset to `false`. Now resets at the top of each call.

## [0.3.1.1] - 2026-03-28 — Bug Fixes & Review Hardening

### Fixed
- **Feed list showing only 3 items**: Screen models were created with height=0 on navigation. Terminal dimensions are now forwarded on every screen switch.
- **Feed sort order**: TUI feed now sorts by published date (newest first) instead of relevance score. Items with missing dates sink to bottom.
- **Relevance percentage alignment**: Fixed column alignment for scores (85% vs 100%) using fixed-width formatting.
- **Spinner never restarting after idle**: Animation tick is now batched on all screen transitions, not just dashboard.
- **Per-topic refresh status dead**: `*tea.Program` is now threaded via `programHolder` so `p.Send(PerTopicRefreshMsg)` actually fires during refresh.
- **Border off-by-one in RenderCard and RenderBanner**: Top border was 1 character narrower than content and bottom borders.
- **Column width calculation**: Uses visual width (`lipgloss.Width`) instead of byte length (`len`) for topic names, fixing alignment with Unicode characters.

### Changed
- **errcheck linter**: Added to TODOS.md as P2 item to re-enable after golangci-lint v2 migration stabilizes.

## [0.3.1.0] - 2026-03-28 — Premium UX Polish (Bloomberg Terminal Aesthetic)

### Added
- **Persistent header**: Fixed header across all screens with breadcrumb navigation (TERMIFLOW > SCREEN > Context), unread badge, connection health dot, and last refresh time.
- **Bordered announcement cards**: Notification banners render as bordered cards with type labels (UPDATE, WARNING, BREAKING). Breaking announcements use double borders.
- **Dashboard command center**: Per-topic activity bars, inline frequency and last-fetch time, activity footer with refresh status and next auto-refresh timer.
- **Per-topic refresh feedback**: Each subscription shows inline status during refresh (spinning, success with count, error) instead of one global spinner.
- **Feed relevance micro-bars**: Visual 4-char bars (████, ██▓░) for relevance scores. Selected items get a subtle dark background.
- **Article detail polish**: Horizontal rules between sections, #tag style (replaces [tag]), dotted separators, 72-char content width cap for readability.
- **Ask screen polish**: Labeled source section (── Sources ──), column-aligned source list, animated search dots, 3-second save flash that fades to muted.
- **Inverted key status bar**: Keys rendered with amber background/black text for a keyboard-key feel, ═ separator matching the header.
- **Help modal overlay**: Help renders as a centered bordered card over the current screen, not a full-screen replacement.
- **Loading animations**: Animated spinner (▁▂▃▄) for loading states across all screens.
- **Bordered welcome card**: Zero-subscription state shows a bordered GET STARTED card pointing to the TUI topics browser.
- **Status info cards**: Status screen renders as bordered section cards (CONNECTION, DATA, SYSTEM).
- **Shared layout system**: New `layout.go` with centralized rendering helpers (RenderHeader, RenderBanner, RenderCard, RelevanceBar, ActivityBar, etc.). All screens use ContentView/StatusHints/Breadcrumb interface.

### Changed
- **Column alignment**: Strict fixed-width columns across all screens. Topic names, unread counts, frequencies, and timestamps snap to consistent positions.
- **Topics browser alignment**: Subscribed and available sections use column-aligned metadata.
- **Screen architecture**: AppModel now owns persistent chrome (header + footer). Screens only emit content via ContentView(). Eliminates 6x duplicated header rendering.

### Fixed
- **Spinner CPU waste**: Animation ticks now stop when no screen is loading, preventing 7x/sec wakeups when idle.
- **Narrow terminal panics**: Clamped negative values in title truncation, banner rendering, help overlay, and card padding to prevent strings.Repeat panics on terminals under 20 chars wide.
- **Stale refresh badges**: Per-topic refresh status is now cleared when refresh completes.
- **Detail breadcrumb**: Article detail screen now shows the topic name in the breadcrumb trail.
- **Card border alignment**: RenderCard right border now aligns correctly with top/bottom borders.

## [0.3.0.0] - 2026-03-27 — Topics Browser, Status Screen & TUI Complete (Batch 6E)

### Added
- **Topics Browser screen**: Press `t` from the dashboard. Two tabbed sections: Subscriptions (with item counts, frequency, unsubscribe via `d`, edit frequency via `e`) and Available (predefined categories, subscribe via Enter with frequency picker). Tab to switch sections.
- **Status screen**: Press `s` from the dashboard. Shows mode (managed/self-hosted), masked API key, subscription count, total items/unread, database path and size, config path. Read-only, Esc to return.
- **User context injection in Ask**: The Ask screen now includes your termiflow state (subscriptions, unread counts, frequencies, last refresh times) in the LLM system prompt. You can ask "how many subscriptions do I have?" or "when was rust-lang last refreshed?" and get accurate answers.
- **Frequency picker**: Subscribe and edit-frequency flows use a 3-option picker (hourly/daily/weekly) with j/k navigation and Enter to confirm.
- **Delete confirmation**: Unsubscribing shows a y/n confirmation dialog before deleting.

### Fixed
- **Global quit during Topics dialogs**: Pressing `q` during the frequency picker or delete confirmation no longer quits the app.
- **Subscription with DB error vanishes from UI**: Subscriptions with item-count query errors now show with 0/0 counts instead of disappearing and reappearing as "available".
- **Dashboard cursor after deletion**: Cursor is now clamped when subscriptions are deleted, preventing broken highlight state.

## [0.2.7.0] - 2026-03-27 — Ask Screen (Batch 6D)

### Added
- **Inline Ask screen**: Press `a` from the dashboard to ask any question. Sources are searched via Tavily, then the LLM streams a response token-by-token in real-time. Press `s` to save the answer to a markdown file. Press `Ctrl+C` to cancel during streaming.
- **Streaming cursor**: Shows a blinking `▍` cursor while the LLM is generating, giving clear visual feedback that the response is in progress.
- **Source display**: After the response completes, search sources are listed below the answer with domain names and titles.
- **Save to disk**: Press `s` after the answer completes to save the Q&A and sources to `~/.local/share/termiflow/saved/{timestamp}-{slug}.md` (0600 permissions).

### Fixed
- **Global quit during Ask streaming**: Pressing `q` or `Ctrl+C` during searching/streaming now cancels the request instead of quitting the entire app. Only the done phase allows global quit.
- **Stale stream chunks after cancel**: Cancelling a stream and starting a new question no longer corrupts the new answer with leftover chunks from the old stream. Phase guards reject stale messages.
- **Context leak on re-ask**: Creating a new context for a new question now properly cancels the old context first, preventing orphaned HTTP connections.
- **Unbounded scroll in Ask**: Scroll position is now clamped based on content length.
- **Dead code cleanup**: Removed unreachable `AskChunkMsg.Content` handler branch.

## [0.2.6.0] - 2026-03-27 — Live Refresh (Batch 6C)

### Added
- **Live feed refresh**: Pressing `r` in the TUI dashboard now fetches fresh articles from the API (Tavily search + LLM curation), not just reloading cached data from the database. Shows "⟳ Refreshing feeds..." during the operation and "Last refreshed Xm ago" after.
- **Auto-refresh**: Dashboard automatically refreshes feeds every 30 minutes via `tea.Tick`. Skips if already refreshing or no subscriptions exist.
- **Error reporting**: Refresh errors are surfaced to the user. Fatal errors (provider init, DB) show "Refresh failed: {reason}". Per-topic errors show "{N} topic(s) failed to refresh".
- **`scheduler.NewFromConfig()`**: Shared factory function that creates a Scheduler from config + provider name. Used by both CLI and TUI, eliminating duplicate provider setup code.

### Changed
- **CLI feed refresh refactored**: `internal/cli/feed.go` now uses `scheduler.NewFromConfig()` instead of inline LLM + search provider initialization. Same behavior, less code.
- **Double-press protection**: Pressing `r` while a refresh is already running is silently ignored.

## [0.2.5.0] - 2026-03-27 — Feed List + Article Detail (Batch 6B)

### Added
- **Feed List screen**: Browse articles for any subscription with j/k navigation, read/unread indicators (● unread, ○ read), relevance scores, and source names. Press Enter on a dashboard subscription to open its feed.
- **Article Detail screen**: Full article view with scrollable content, title, source URL, summary, tags, and relevance score. Press `o` to open in browser, `m` to mark as read, `n`/`p` to navigate between articles without returning to the list.
- **Client-side filter**: Press `/` or `f` in the feed list to filter articles by title, summary, or source. Live filtering as you type. Esc clears the filter.
- **Unread toggle**: Press `u` in the feed list to show only unread items.
- **Auto mark-as-read**: Articles are automatically marked as read in the database when opened in detail view.

### Fixed
- **Global quit during filter input**: Typing "q" in the filter text field no longer quits the app. Global keys are bypassed when the feed is capturing text input.
- **Detail back navigation**: Pressing Esc in article detail now correctly returns to the feed list (was silently dropped due to nil subscription guard).
- **URL scheme validation**: `openBrowser` now only allows http/https URLs, preventing `file://` or `javascript:` scheme injection from malicious feed data. Child processes are properly reaped.
- **UTF-8 title truncation**: Feed item titles are now truncated using rune-safe slicing instead of byte slicing, preventing garbled output for international characters.
- **Stale feed data guard**: Feed items loaded from a previous subscription are discarded if the user switches topics before the DB query returns.

## [0.2.4.0] - 2026-03-27 — TUI Scaffold (Batch 6A)

### Added
- **Bubble Tea TUI**: Running bare `termiflow` now launches a full-screen interactive TUI (alt-screen mode) instead of printing a static dashboard to stdout. Existing CLI commands (`termiflow feed`, `termiflow ask`, etc.) are unchanged.
- **Dashboard screen**: Subscription list with j/k navigation, unread counts per topic, zero-subscription getting-started guide, and status bar with keybinding hints.
- **Help overlay**: Press `?` to see all keybindings for the current screen.
- **Notification banners in TUI**: Announcement and version banners render as a top bar inside the TUI, reusing the existing `internal/notifications/` system.
- **TTY detection**: TUI only launches on interactive terminals. Pipes, `--json`, and `--quiet` fall back to the original stdout dashboard. Tests pass without a TTY.

### Changed
- **Go version bumped to 1.24**: Required by `charmbracelet/bubbletea` v1.3.10. CI workflows (ci.yml, release.yml) updated from Go 1.22/1.23 to 1.24.
- **lipgloss upgraded to v1.1.0**: From v0.9.1. Required by bubbletea. All existing code compiles and works with the new version.

## [0.2.3.0] - 2026-03-27 — CLI Polish (Batch 3)

### Added
- **`--json` global flag**: Structured JSON output for scripting. Common envelope: `{"data": ..., "meta": {"version": ..., "timestamp": ...}}`. Supported by `feed`, `status`, `ask`, `changelog`, and the no-args dashboard. `--json` implies `--quiet` to prevent spinners corrupting output. Error envelope includes partial data via `WriteJSONError`.
- **`termiflow changelog` command**: Fetches release notes from GitHub Releases API. Shows latest 3 releases by default, `--all` for full history. Bloomberg-style rendering with colored tags (NEW=green, FIX=blue, BREAKING=red). Includes `--json` support.
- **`termiflow` no-args dashboard**: Running `termiflow` with no arguments now shows a summary dashboard with subscription unread counts and command hints instead of help text. Zero-subscription state shows a getting-started guide.
- **GitHub API ETag caching**: `changelog` and `upgrade` commands cache GitHub Releases API responses with ETag conditional requests. Avoids hitting the 60 req/hr unauthenticated rate limit.
- **Shared GitHub API helper** (`internal/cli/github.go`): Extracted from `upgrade.go`. `fetchLatestRelease()` and `fetchReleases(limit)` with ETag caching, rate limit handling, and 5MB response size limit.

### Changed
- **`upgrade` command refactored**: Now uses shared `fetchLatestRelease()` instead of inline HTTP. Same behavior, less code.
- **API key masking tightened**: Status command only shows partial key when key is longer than 20 characters (was 10). Shows fewer characters (first 4 + last 4).
- **Cache file permissions**: GitHub releases cache written with 0600 (was 0644).

## [0.2.2.1] - 2026-03-27

### Fixed
- **Stream goroutine leak**: SSE stream goroutines in both managed and Anthropic providers now check context cancellation on every channel send. Previously, if the consumer stopped reading (ctrl+c, error), the goroutine blocked forever, leaking the goroutine and HTTP connection.
- **Duplicate feed items**: Added unique index on `(subscription_id, source_url)` and changed `CreateFeedItem` to `INSERT OR IGNORE`. Parallel subscription refresh could previously insert the same article twice due to a check-then-insert race condition.
- **WAL mode verification**: `PRAGMA journal_mode=WAL` result is now verified via `QueryRow` instead of `Exec`. If WAL mode fails to activate (e.g., network filesystem), the error is caught at startup instead of silently falling back to default journaling.
- **Done chunk on connection drop**: Stream goroutines now always send a `Done` chunk via deferred send before closing the channel, even when the server drops the connection without sending `message_stop`.

## [0.2.2.0] - 2026-03-27

### Added
- **Rate limit error handling**: 429 responses return a typed `RateLimitError` with a human-friendly countdown ("Rate limited. Try again in 47m.") instead of raw JSON. No useless retries against 1-hour rate windows.
- **Real SSE streaming in managed mode**: `ask` command now streams responses in real-time instead of buffering the entire response. Uses a dedicated HTTP client with no timeout for long-running SSE connections.
- **Parallel subscription refresh**: `feed --refresh` processes all subscriptions concurrently instead of sequentially. Context cancellation on offline detection for fast failure.
- **SQLite WAL mode**: Database now uses Write-Ahead Logging and 5-second busy timeout, enabling safe concurrent reads/writes during parallel refresh.

### Changed
- **Shared retry logic**: Extracted `DoWithRetry()` helper in `retry.go`. Both managed providers use it instead of duplicated retry loops. 429s return immediately; only 503s are retried with exponential backoff.
- **Rate limit warnings during refresh**: When a subscription hits a rate limit during parallel refresh, a warning is shown after the spinner completes instead of silently failing.

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
