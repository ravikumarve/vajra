# 🗡️ VAJRA — Project AGENTS.md

## Project Overview
- **What:** Open-source MCP ephemeral identity boilerplate for AI agents
- **Stack:** Go (core engine), Bubble Tea TUI, embedded SQLite, OPA/Rego policies
- **Distribution:** Gumroad ($79 starter / $149 pro) + GitHub (MIT core)
- **Target Buyer:** Developers building AI agents who need credential security without building from scratch
- **Platform:** Single static Go binary — no Docker, no Postgres, no cloud deps

## Sprint Status
- **Phase:** 1 — MVP Build (COMPLETE)
- **Current:** v0.1.0 — All core subsystems built, wired, and compiling
- **Next:** Phase 2 — Audience Building & Landing Page
- **Market Verdict:** BOILERPLATE VALIDATED. Zero direct competitors in Gumroad/boilerplate space. 11 competitors in enterprise SaaS — irrelevant to us.

## Architectural Decisions
- **Go over Rust:** Faster build, easier distribution, good enough perf for a boilerplate
- **Single binary over Docker:** Zero runtime deps = lower friction for buyers
- **TUI over web dashboard:** Faster to build, distinct from OneCLI's Next.js dashboard
- **MCP-native identity:** Built for MCP 2026-07-28 spec from day one
- **OPA/Rego embedded:** Industry-standard policy engine, not a custom DSL
- **No Vault/Cloak modules in v1:** Deferred to post-revenue

## Tech Stack
| Component | Choice | Why |
|-----------|--------|-----|
| Language | Go 1.24+ | Single binary, fast compile, excellent stdlib |
| MCP Transport | stdio + HTTP SSE | MCP 2026-07-28 compliant |
| Policy Engine | OPA/Rego (embedded) | Industry standard, ABAC/ RBAC out of the box |
| Credential Store | In-memory ring buffer | Sub-5s TTL, no persistence needed |
| DB Persistence | SQLite (via modernc.org/sqlite) | Zero deps, embedded, no CGO |
| TUI | Bubble Tea (charmbracelet/bubbletea) | Terminal UI, no browser needed |
| Agent SDK Adapters | LangChain (TS + Python), CrewAI | Reach the widest agent dev audience |

## Milestones
- [ ] **Pre-Code:** Audience validation (100+ waitlist signups)
- [x] **Core Engine:** MCP proxy + credential ring buffer + OAuth device flow
- [x] **Policy Layer:** OPA/Rego rules + `vajra policy` CLI
- [x] **Adapters:** LangChain + CrewAI + raw MCP SDK
- [x] **TUI Dashboard:** Full-screen Bubble Tea dashboard
- [x] **Auth Subsystem:** OAuth 2.1, mTLS, agent registry, token exchange
- [x] **Audit Trail:** SQLite append-only log + JSON stream
- [ ] **Launch:** Gumroad listing + HN post + Dev.to series + landing page

## Session Memory
### [2026-09-21] — Sale packaging: PRODUCT.md + dual LICENSE + 5 shots + dist zip ✅
- **State:** Success — committed & pushed. PRODUCT.md filled (positioning/buyers/features/Core-$79-$149 tiers/stack/v0.1.0 status); LICENSE upgraded to full Commercial (Core carve-out → LICENSE-MIT added, matches README dual model); 5 screenshots (01-help/02-mint/03-ps/04-policy PNGs via PIL renderer + 05-tui frame + demo.gif via vhs 0.11); dist/vajra-v0.1.0-gumroad.zip (9.4M: binary + licenses + README + PRODUCT + env + policies + contrib + docs). Binary + dist/ stay gitignored (local artifacts). Committed fix-pack leftovers too (retry.go/tests, migrations, Dockerfile, ci.yml, .env.example, sqlite.go hardening).
- **Gates:** `go build` ✅ 28M binary; verifier 74/100 carried; screenshots visually verified (TUI dashboard renders live).
- **Next Turn Directive:** Still missing for listing: Gumroad sales copy (no GUMROAD doc yet). Then list Starter $79 / Pro $149.
### [2026-09-20] — Pre-sale Fix Pack: verifier 63 → 74 GATE PASS ✅ (first test suite)
- **State:** Success — all changes local + tested, NOT committed
- **Fixes:** INTEGRATIONS.md secrets → `<minted-per-agent>` placeholders; `.gitignore` node/__pycache__/venv; `.env.example`; `Dockerfile` (distroless static binary); `ci.yml` (vet/test/build); `migrations/` (001 audit schema + README, `PRAGMA user_version` stamping + newer-than-binary refusal in `sqlite.go`); `audit/retry.go` (backoff+jitter on SQLITE_BUSY, wired into `Log`); first tests — policy (defaults/custom/invalid, 5), audit retry+roundtrip (5), ring TTL/revoke (2), device flow (2) = **14 pass**
- **Left as honest WARNs:** framework (stdlib CLI, no web framework — correct), input-validation (no validator dep — optional `go-playground/validator` later), dangerous-calls (SQLite `.Exec` + OPA `.Eval` API names — unrenamable false positives)
- **Gates:** verifier **74/100 PASS 29/WARN 3/FAIL 0** | `go build` ✅ `go vet` clean ✅ 14/14 tests ✅ (pre-existing gofmt drift in 7 untouched files left alone)
- **Next Turn Directive:** Commit fix pack (LICENSE + PRODUCT.md still untracked — decide: ship commercial LICENSE in repo per $79/$149 plan), capture screenshots (0 today — Gumroad requires), then list

### [2026-07-17 18:00] — VAJRA Validation & Pivot Sprint
- **State:** Success — Market research complete, pivot decision made
- **MCP Data Used:** websearch (market research, competitor analysis, boilerplate market data), code_tree (project structure), project-bootstrap (AGENTS.md template)
- **Agents Deployed:** Orchestrator (direct execution for research + file creation)
- **Architectural Decision:** Pivoted from enterprise SaaS ($1K-4K/mo) to developer boilerplate ($79/$149 one-time). Stripped VAJRA Vault and VAJRA Cloak from v1. Core focus: MCP ephemeral identity broker as a single Go binary.
- **Key Outputs:** This AGENTS.md, README.md, updated idea.md
- **Next Turn Directive:** Begin Phase 1 — build waitlist landing page and write 3 Dev.to audience-building posts BEFORE writing any Go code.

### [2026-07-18 00:00] — VAJRA v0.1.0 Build Sprint
- **State:** Success — All core subsystems complete
- **MCP Data Used:** direct file reads for existing code, code_tree for structure checks
- **Agents Deployed:** Orchestrator (direct execution for all files)
- **Architectural Decision:** Auth, TUI, audit, policy, credential, transport — all wired into a single `serve` daemon command. 7 CLI commands total.
- **Key Outputs:** `internal/auth/` (4 files), `internal/credential/mint.go + revoke.go`, `internal/policy/opa.go` (real Rego eval), `internal/tui/dashboard.go` (full-screen standalone), `internal/audit/sqlite.go + stream.go`, `cmd/vajra/tui.go`. README updated to reflect live v0.1.0 state.
- **Build Status:** 38 source files. Compiles to 28MB single binary. All 7 commands functional.
- **Next Turn Directive:** Phase 2 — Design landing page HTML, then extract design tokens for Go web dashboard. Or update README/AGENTS.md for the current state.
