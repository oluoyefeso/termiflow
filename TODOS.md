# TODOS

## Phase 1 — Engagement Core

### Morning briefing on dashboard
**What:** When TUI launches, show a personalized "since you were last here" briefing instead of the raw subscription list. Top 3-5 articles across all topics since last session, one-line summaries, sorted by relevance. Quick-scan format: 30 seconds to know what happened overnight.
**Why:** The dashboard is currently a static subscription list. It gives you no reason to launch termiflow over just... not launching it. A morning briefing makes the first 5 seconds feel like value, not navigation.
**Pros:** Turns termiflow into a daily habit. The dashboard becomes the product, not a menu. Differentiates from every RSS reader.
**Cons:** Needs "last seen" timestamp tracking. Needs a scoring pass to pick the top articles (not just chronological). Medium complexity.
**Context:** Dashboard is `internal/tui/dashboard.go`. Would add a "briefing mode" that shows on first render if unread articles exist, then collapses to normal view on any keypress. Articles already have scores from the curation pipeline.
**Depends on:** Nothing — can ship standalone.
**Priority:** P1

### Read/skip tracking
**What:** Track which articles users actually open (enter → detail view) vs scroll past. Store as lightweight events in SQLite: `(article_id, action, timestamp)` where action is `read`, `skipped`, or `dismissed`.
**Why:** This is the foundation of personalization. Without knowing what users engage with, the feed is just keyword matching. With it, we can learn what they actually care about.
**Pros:** Tiny data footprint. No new dependencies. Enables the entire personalization track.
**Cons:** Schema migration. Privacy consideration for managed users (events stay local-only, never synced to server).
**Context:** `internal/db/` for schema. Hook into `OpenDetailMsg` in `internal/tui/feed.go` for reads. Skips inferred from scrolling past items without opening.
**Depends on:** Nothing — can ship standalone.
**Priority:** P1

### Session streak indicator
**What:** Track consecutive days the user launched termiflow. Show quietly in the TUI header: "day 12" next to the version string. No gamification, no badges, no congratulations. Just a number.
**Why:** Developers hate gamification but respond to personal accountability. A quiet streak counter creates a gentle pull to not break the chain. It's the "GitHub contribution graph" effect — you notice when it resets.
**Pros:** Trivial to implement (one timestamp in SQLite). Subtle but effective retention signal.
**Cons:** Could feel gimmicky if overdone. Keep it to literally just the number.
**Context:** Add `last_session_date` and `streak_days` to a new `user_stats` table. Update on each TUI launch in `internal/tui/app.go`. Display in `internal/tui/layout.go` header.
**Depends on:** Nothing.
**Priority:** P2

---

## Phase 1 — Personalization Engine

### Topic affinity model
**What:** Build a per-user interest profile from read/skip data. Not just "subscribed to rust" but "reads 80% of rust-performance articles, 10% of rust-syntax articles, and 60% of anything mentioning async." Represent as a weighted vector of extracted tags from engaged content.
**Why:** This is what makes the feed get smarter over time. Week 1 it's keyword matching. Week 4 it knows your taste. That's the retention flywheel — the longer you use it, the harder it is to leave because no other tool has your profile.
**Pros:** Uses tag data already extracted by the curation pipeline. Scoring becomes `affinity_score * relevance_score` instead of just `relevance_score`. Massive quality improvement with modest code.
**Cons:** Needs enough data to be useful (cold start problem). Need a minimum of ~20-30 read events before the model adds signal over random. Show "learning your preferences" during cold start period.
**Context:** New `internal/intelligence/affinity.go` (or in the engine). Reads from the tracking events table. Outputs a tag → weight map. Feed scoring multiplies article tags against this map.
**Depends on:** Read/skip tracking.
**Priority:** P1

### Source quality scoring
**What:** Score content sources (domains) per-user based on engagement rates. If a user reads 4 out of 5 HN articles but 1 out of 10 from dev.to, HN should rank higher in their feed. Stored as `(user, domain, read_count, skip_count, quality_score)`.
**Why:** Not all sources are equal for all users. A senior systems engineer gets different value from lwn.net vs freecodecamp than a bootcamp grad. Source scoring is the second dimension of personalization (topic affinity is the first).
**Pros:** Stacks with topic affinity for compound scoring. Cheap to compute.
**Cons:** Needs enough per-source data. Sources with < 5 interactions use a global default.
**Depends on:** Read/skip tracking.
**Priority:** P2

### Smart feed mode ("For You")
**What:** A new feed mode that ignores subscription boundaries. Instead of "show me all rust articles" it's "show me the articles I'm most likely to care about across all topics." Scored by `affinity * source_quality * recency * relevance`. The "For You" feed.
**Why:** This is what makes termiflow feel like an algorithm that works for you instead of against you. Every social platform has this, but they optimize for engagement (time on site). We optimize for signal (did you actually learn something useful?).
**Pros:** The killer feature. The thing users tell other devs about.
**Cons:** Needs both affinity model and source scoring to be meaningful. Cold start problem. Needs a fallback to chronological until enough data exists.
**Context:** New screen or mode toggle on the dashboard. Queries across all subscriptions, applies compound scoring, shows top N.
**Depends on:** Topic affinity model, source quality scoring.
**Priority:** P2

---

## Phase 1 — Engine Extraction (termiflow-engine)

### Extract intelligence layer as Go library
**What:** Move the curation pipeline (curator, scorer, summarizer, tagger, asker) into `github.com/oluoyefeso/termiflow-engine` as a standalone Go module. Clean interfaces (`LLMProvider`, `SearchProvider`), zero concrete provider dependencies, mock providers for testing.
**Why:** The curation pipeline is the most valuable and reusable part of termiflow. As a library, anyone building a content product in Go can import it. As internal code, it's locked inside the CLI where nobody else can use it.
**Pros:** Forces clean interfaces (makes the CLI better). Marketing asset. Contributor funnel. First-mover in Go for content intelligence libraries.
**Cons:** Two repos to maintain. Release coordination between engine and CLI. Must keep the library surface small.
**Context:** Full design doc at `~/.gstack/projects/termiflow/mac-unknown-design-20260326-145257.md`. CEO + Eng reviewed and approved. Phase 1 of the 3-phase migration plan.
**Depends on:** Nothing — can start independently.
**Priority:** P2

### Add learning layer to engine
**What:** The engine's identity isn't "curation pipeline" — it's "personal content intelligence." Add the affinity model and source scoring as engine-level APIs: `engine.UpdateProfile(events []ReadEvent)`, `engine.ScoreForUser(profile *UserProfile, articles []FeedItem) []ScoredItem`. The engine learns, not just curates.
**Why:** This is what makes termiflow-engine worth importing. A curation pipeline is a commodity (call an LLM, parse the response). A learning content intelligence library that gets smarter per-user is something nobody has in Go.
**Pros:** Differentiates the library. Makes the "For You" feed a one-liner for any app using the engine.
**Cons:** Depends on the affinity model being validated in the CLI first. Don't abstract before proving the approach works.
**Context:** Build and validate in the CLI (`internal/intelligence/`) first. Extract to engine after it's proven. Don't build the library API speculatively.
**Depends on:** Topic affinity model (validated in CLI), engine extraction.
**Priority:** P3

### Social content sources
**What:** Add content source adapters for HN (Algolia API), Reddit (JSON API), and Twitter/X (via Tavily or scraping). Instead of only searching web articles, the engine can surface discussions, threads, and comments about your topics.
**Why:** Articles are one content type. Conversations are another. "Rust async was trending on HN yesterday, 47 comments" is a different kind of signal that articles miss. Devs care about what other devs are saying, not just what bloggers publish.
**Pros:** Richer content mix. Catches breaking discussions that articles lag behind by hours/days.
**Cons:** API rate limits (especially Twitter). Content quality varies wildly. Need good dedup (article + HN discussion of that article = one item, not two).
**Context:** New `SearchProvider` implementations in the engine or CLI's `internal/providers/search/`. HN has a free Algolia API. Reddit has a JSON API. Twitter is the hardest.
**Depends on:** Engine extraction (ideally), or can be built directly in CLI.
**Priority:** P3

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
