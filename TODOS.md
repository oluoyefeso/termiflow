# TODOS

## Post-MVP (UX improvements)

### User-selectable terminal theme system
**What:** Let users pick a color theme in settings (`termiflow config set theme <name>`) that controls the entire app's palette: TUI chrome, glamour markdown rendering, lipgloss styles.
**Why:** Currently the amber-on-dark palette is hardcoded. Users with light terminals or different preferences can't customize.
**Pros:** Ship with 2-3 built-in themes (dark-amber default, light, dracula). Glamour renderer already accepts style config.
**Cons:** Need to thread theme config through all lipgloss style definitions. Medium effort.
**Context:** `internal/ui/colors.go` (CLI palette), `internal/tui/styles.go` (TUI palette), `internal/ui/markdown.go` (glamour renderer). Glamour supports custom `ansi.StyleConfig`.
**Depends on:** Glamour integration (shipped in v0.3.11.0).
**Priority:** P3

---

## Done

### Glamour markdown rendering for ask command
**Shipped in:** v0.3.11.0
**What:** LLM responses in `ask` command now render with glamour terminal markdown (headings, bold, syntax-highlighted code blocks). CLI uses separator approach (raw stream + divider + rendered). TUI swaps to glamour on completion with cached rendering. TTY-aware, piped output stays raw.

### SSE streaming pass-through in backend proxy
**Shipped in:** v0.3.9.0 (backend) + v0.3.10.0 (CLI integration)
**What:** Backend proxy forwards Anthropic SSE chunks verbatim via http.Flusher. ManagedProvider.Stream() parses SSE events in real-time. 8 streaming tests cover happy path, retry, large events, malformed SSE, connection drops, rate limits.

### Parallelize within-article LLM calls + concurrent subscription refresh
**Shipped in:** v0.3.9.0
**What:** Two optimizations: (1) Summarize and ExtractTags run in parallel within each article goroutine (engine), (2) Subscription refresh runs up to 2 subscriptions concurrently (CLI + TUI). Existing between-article concurrency (maxConcurrency=5) was already in place.
**Impact:** ~2-4x faster feed refresh. Max burst 20 concurrent LLM calls (2 subs × 5 articles × 2 calls).

### CLI rate limit error handling
**Shipped in:** v0.3.10.1
**What:** CLI now detects `RateLimitError` from managed providers and shows user-friendly messages with actual wait times. `feed --refresh` lists skipped topics and wait duration. `ask` shows clean rate limit message instead of raw wrapped error.

### Rate limiting per API key
**Shipped in:** `d2ece1d` (implementation) + eng review fixes (middleware order, cleanup race, panic recovery, JSON 429 response, server timeout, test coverage)
**What:** Per-key rate limiting in the managed backend (100 req/hr sliding window). `sync.Map` + sliding window, auth → rate_limit → proxy middleware chain, `X-RateLimit-*` headers, 429 with `Retry-After`. 13 tests covering core logic, concurrency, middleware integration, cleanup, and edge cases.
