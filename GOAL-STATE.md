# termiflow CLI — Goal State & Roadmap

Last updated: 2026-03-31

---

## What This Repo Is

The Termiflow CLI. A unified Go binary supporting both self-hosted (bring your own keys)
and managed (`tf_xxx` API key) modes. Terminal-native AI intelligence tool: subscribe to
concepts, get cross-cutting intelligence, ask questions — all from the command line.

**GitHub:** https://github.com/oluoyefeso/termiflow
**Visibility:** Public (open source, GPL)

---

## Goal State (the "done" picture)

The goal state is defined by the landing page vision on `feat/landing-page-vision-restructure`
of termiflow.com. The CLI/TUI should deliver what the landing page promises.

### What the Landing Page Promises vs What Exists Today

```
LANDING PAGE PROMISE                    CLI TODAY               GAP
──────────────────────────────────────────────────────────────────────
"The feed that learns you"              Static relevance scores  PERSONALIZATION ENGINE
Morning briefing ("since you            No briefing feature      BRIEFING GENERATOR
  were last here" summary)
Accuracy % + profile week               Not tracked              AFFINITY TRACKING
MATCHED annotations ("why this          No match explanations    CONCEPT MATCHING
  appeared in your feed")
Learned interests (auto-detected)       Only explicit subs       IMPLICIT INTEREST TRACKING
Relevance score bars (████ 94%)         Numeric scores exist     UI ENHANCEMENT (small)
Cross-source intelligence               RSS + Tavily search      MOSTLY DONE (source diversity)
Subscribe to concepts, not keywords     Keywords today           CONCEPT MAPPING (engine)
Week 1 broad → Week 4 precise           No evolution over time   TEMPORAL LEARNING
Ask your feed (query history)           Ask works, no history    QUERY HISTORY
Theme system (amber/dracula/light)      On feat/theme-system     READY TO MERGE
```

### Architecture Diagram (Goal State)

```
                          ┌─────────────────────────────┐
                          │      termiflow CLI/TUI       │
                          │                             │
                          │  ┌───────────┐ ┌─────────┐ │
                          │  │ Dashboard │ │  Ask    │ │
                          │  │ + Briefing│ │ Screen  │ │
                          │  └─────┬─────┘ └────┬────┘ │
                          │        │             │      │
                          │  ┌─────▼─────────────▼────┐ │
                          │  │   Personalization       │ │
                          │  │   Engine (from lib)     │ │
                          │  │  - topic affinity       │ │
                          │  │  - read/skip tracking   │ │
                          │  │  - concept matching     │ │
                          │  │  - temporal learning    │ │
                          │  └─────┬──────────────────┘ │
                          │        │                    │
                          │  ┌─────▼──────┐            │
                          │  │ Curation   │            │
                          │  │ Pipeline   │            │
                          │  │ (engine)   │            │
                          │  └─────┬──────┘            │
                          │        │                    │
                          │  ┌─────▼──────┐            │
                          │  │  SQLite    │            │
                          │  │  + affinity│            │
                          │  │  tables    │            │
                          │  └────────────┘            │
                          └──────────┬──────────────────┘
                                     │
                    ┌────────────────┼────────────────┐
                    ▼                ▼                ▼
              Anthropic API    Tavily API      RSS/Web Sources
              (self-hosted)    (self-hosted)
                    OR               OR
              termiflow-api    termiflow-api
              (managed mode)   (managed mode)
```

---

## Current State (v0.3.12.0)

### What's Done (Batches 0-6 complete)
- Full CLI: `config`, `subscribe`, `unsubscribe`, `topics`, `feed`, `ask`, `version`, `changelog`, `status`, `upgrade`
- Full TUI (Bubble Tea): 6 screens — Dashboard, Feed List, Article Detail, Ask, Topics Browser, Status
- Multiple LLM backends: OpenAI, Anthropic, Local (Ollama/llama.cpp/LM Studio)
- Search backends: Tavily, RSS, web scraper
- SQLite local database
- Bloomberg-style terminal UI
- `--watch` mode, `--json` output, `ask --save`
- Notification/announcement system
- CI/CD: GitHub Actions, Goreleaser, Homebrew tap
- Mock build tag for dev without API keys
- Theme system (on `feat/theme-system` branch, ready to merge)

### Active Branches
| Branch | Status | Description |
|--------|--------|-------------|
| `feat/theme-system` | Ready to merge | User-selectable themes (amber/dracula/light) |
| `feat/engine-integration` | In progress | Wire termiflow-engine into CLI |
| `feat/custom-sources` | In progress | User-defined RSS/web sources |

---

## Gap Analysis: Current → Goal

### Critical Path (blocks the landing page promise)

```
┌─────────────────────────────────────────────────────────────┐
│                    CRITICAL PATH                             │
│                                                             │
│  1. Personalization Engine (in termiflow-engine)            │
│     └── Topic affinity tracking                             │
│     └── Read/skip/return-to pattern analysis                │
│     └── Temporal learning (Week 1 → Week 4)                 │
│     └── Concept matching (not just keywords)                │
│                                                             │
│  2. CLI Integration of Personalization                       │
│     └── Morning briefing generation                         │
│     └── MATCHED annotations in feed output + TUI            │
│     └── Learned interests display                           │
│     └── Accuracy % + profile week in header                 │
│     └── "For You" ranked feed (cross-topic)                 │
│                                                             │
│  3. Merge feat/theme-system                                 │
│     └── Already done, just needs merge                      │
│                                                             │
│  4. Inline preview on subscribe                             │
│     └── Show 3-5 items immediately after subscribing        │
└─────────────────────────────────────────────────────────────┘
```

### Non-Critical (ship anytime, independent)

- Feed health monitoring (track broken source URLs)
- Server-side sync for managed users
- Query history sync
- Homebrew tap publish (blocked on token config)

---

## Phased Roadmap

### Phase 1: Merge & Stabilize
**Goal:** Get all ready branches merged and the CLI stable at v0.4.0.

| Task | Owner | Effort | Depends On |
|------|-------|--------|------------|
| Merge `feat/theme-system` into main | Any contributor | S | Nothing |
| Merge `feat/engine-integration` (if ready) | Any contributor | M | Engine v0.1.0 |
| Run golangci-lint + fix issues | Any contributor | S | Nothing |
| Tag v0.4.0 release | Maintainer | S | Above merges |

### Phase 2: Personalization Foundation
**Goal:** Track user behavior and build the affinity model.
**Depends on:** termiflow-engine shipping personalization interfaces.

| Task | Owner | Effort | Depends On |
|------|-------|--------|------------|
| Add read/skip/dwell-time event tracking to TUI | CLI contributor | M | Nothing |
| Add `affinity_events` + `topic_affinity` SQLite tables | CLI contributor | S | Nothing |
| Wire personalization engine from termiflow-engine | CLI contributor | M | Engine Phase 2 |
| Implement accuracy % calculation | CLI contributor | S | Affinity tables |
| Add profile week counter | CLI contributor | S | Nothing |

### Phase 3: Intelligence Features
**Goal:** Deliver the landing page promises in the actual CLI.

| Task | Owner | Effort | Depends On |
|------|-------|--------|------------|
| Morning briefing generator | CLI contributor | L | Phase 2 |
| MATCHED annotations in CLI + TUI output | CLI contributor | M | Engine concept matching |
| Learned interests display in TUI | CLI contributor | S | Affinity tracking |
| "For You" ranked feed (cross-topic) | CLI contributor | M | Affinity model |
| Inline preview on `subscribe` | CLI contributor | S | Nothing |
| Relevance score bars in TUI (████ 94%) | CLI contributor | S | Nothing |

### Phase 4: Polish & Scale
**Goal:** Production-quality personalization + contributor experience.

| Task | Owner | Effort | Depends On |
|------|-------|--------|------------|
| Feed health monitoring (broken source detection) | CLI contributor | S | Nothing |
| Server-side sync (managed users) | CLI + API contributor | L | API work |
| Query history sync | CLI + API contributor | S | Server-side sync |
| Performance profiling (large feed databases) | CLI contributor | M | Nothing |
| CONTRIBUTING.md with dev setup guide | Any contributor | S | Nothing |

---

## Contributor Quick Reference

### Dev Setup
```bash
git clone https://github.com/oluoyefeso/termiflow.git
cd termiflow
make build          # Build the binary
make test           # Run tests
make build-mock     # Build without real API calls
make run-mock ARGS="feed --refresh"   # Run with mock data
```

### Key Directories
```
cmd/termiflow/         # Entry point
internal/cli/          # CLI commands (cobra)
internal/tui/          # Bubble Tea TUI screens
internal/ui/           # Output formatting, themes, colors
internal/db/           # SQLite models
internal/providers/    # LLM + search provider implementations
internal/scheduler/    # Subscription refresh scheduling
internal/notifications/ # Announcement system
pkg/models/            # Shared data types
```

### Before Pushing
```bash
make test
golangci-lint run
```
CI will catch gofmt issues that `go vet` misses. Run the linter locally.
