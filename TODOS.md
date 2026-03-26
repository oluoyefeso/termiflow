# TODOS

## High Priority (ship before public key issuance)

### Rate limiting per API key
**What:** Per-key rate limiting in the managed backend (100 req/hr sliding window).
**Why:** Prevents a single leaked `tf_xxx` key from running up the Anthropic bill. Without it, one bad actor can drain the server-side Anthropic quota.
**Pros:** Cheap to implement; protects the operator budget before any public-facing use.
**Cons:** In-memory only (resets on restart) — acceptable for single-server MVP, not for multi-instance.
**Context:** Backend proxy in `internal/api/proxy_llm.go`. Middleware chain: auth → rate_limit → proxy. Use `sync.Map` + sliding window counter per key. No Redis needed. Add `X-RateLimit-Remaining` response header for transparency.
**Depends on:** Managed backend shipping first.

---

## Post-MVP (UX improvements)

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
