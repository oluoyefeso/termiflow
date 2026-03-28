# TODOS

## Post-MVP (UX improvements)

### SSE streaming pass-through in backend proxy
**What:** Backend proxy forwards Anthropic SSE chunks verbatim to the CLI instead of buffering.
**Why:** The `ask` command currently outputs all text at once in managed mode. Token-by-token streaming significantly improves perceived responsiveness.
**Pros:** Better UX. CLI already handles streaming (see ask.go).
**Cons:** Requires `http.Flusher` assertion in `proxy_llm.go`. Need to test SSE chunking boundary cases. Non-trivial Go HTTP pattern.
**Context:** `internal/api/proxy_llm.go`. Use `w.(http.Flusher)` — assert and call `flusher.Flush()` after each SSE chunk. `ManagedProvider.Stream()` currently calls `Complete()` internally (buffered) — update it to parse SSE events once backend supports it. Add explicit integration test with a slow mock Anthropic server.
**Depends on:** Nothing blocking — independent of rate limiting.

---

## Done

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
