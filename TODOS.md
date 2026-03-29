# TODOS

## Post-MVP (UX improvements)

---

## Done

### User-selectable terminal theme system
**Shipped in:** v0.3.12.0
**What:** 3 built-in themes (amber, light, dracula) via `termiflow config set general.theme <name>`. Theme struct (15 fields) as single source of truth, LoadTheme() recomputes all CLI + TUI styles, themed glamour rendering, NO_COLOR env var support. 13 new tests.

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
