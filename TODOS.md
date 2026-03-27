# TODOS

## Post-MVP (UX improvements)

### CLI rate limit error handling
**What:** Parse 429 responses from the managed API and show user-friendly "Rate limited. Try again in Xm." instead of raw HTTP error.
**Why:** When users hit the 100 req/hr limit during `feed --refresh` (which makes many LLM calls), they see a cryptic "API error 429" message. The `Retry-After` header is available but the CLI doesn't use it.
**Pros:** Better UX for managed mode users hitting limits. Tells them exactly when to retry.
**Cons:** Minor effort. Need to parse Retry-After header in the managed provider's error handling path.
**Context:** `internal/providers/llm/managed.go` and `internal/providers/search/managed.go`. Check `resp.StatusCode == 429`, parse `Retry-After` header, return a typed `RateLimitError` that the CLI can detect and format nicely. The retry logic in `internal/providers/retry.go` caps at 30s backoff which is too short for the 1-hour rate limit window — should detect termiflow 429s and not retry them.
**Depends on:** Nothing blocking.

### SSE streaming pass-through in backend proxy
**What:** Backend proxy forwards Anthropic SSE chunks verbatim to the CLI instead of buffering.
**Why:** The `ask` command currently outputs all text at once in managed mode. Token-by-token streaming significantly improves perceived responsiveness.
**Pros:** Better UX. CLI already handles streaming (see ask.go).
**Cons:** Requires `http.Flusher` assertion in `proxy_llm.go`. Need to test SSE chunking boundary cases. Non-trivial Go HTTP pattern.
**Context:** `internal/api/proxy_llm.go`. Use `w.(http.Flusher)` — assert and call `flusher.Flush()` after each SSE chunk. `ManagedProvider.Stream()` currently calls `Complete()` internally (buffered) — update it to parse SSE events once backend supports it. Add explicit integration test with a slow mock Anthropic server.
**Depends on:** Nothing blocking — independent of rate limiting.

### Parallelize curator article processing
**What:** Process each article's score + summarize + tags LLM calls concurrently in `CurateResults()`.
**Why:** With 10 search results and managed mode (CLI→backend→Anthropic), 30 sequential HTTP hops take ~60s. Parallelizing to 5 concurrent articles reduces this to ~15s.
**Pros:** 4x faster feed refresh. No change to API surface.
**Cons:** Adds goroutines + sync complexity to `internal/intelligence/curator.go`. Must bound concurrency (semaphore, max 5) to avoid overwhelming the managed API.
**Context:** `internal/intelligence/curator.go`, `CurateResults()`. Use `errgroup.Group` (golang.org/x/sync/errgroup) with semaphore: `sem := semaphore.NewWeighted(5)`. Each article processes in its own goroutine. Collect results into a channel. Existing test coverage in curator should be extended with concurrency assertions.
**Depends on:** Nothing blocking.

---

## Done

### Rate limiting per API key
**Shipped in:** `d2ece1d` (implementation) + eng review fixes (middleware order, cleanup race, panic recovery, JSON 429 response, server timeout, test coverage)
**What:** Per-key rate limiting in the managed backend (100 req/hr sliding window). `sync.Map` + sliding window, auth → rate_limit → proxy middleware chain, `X-RateLimit-*` headers, 429 with `Retry-After`. 13 tests covering core logic, concurrency, middleware integration, cleanup, and edge cases.
