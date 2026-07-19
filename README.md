# 🗡️ VAJRA — MCP Ephemeral Identity Broker

**Your AI agents get credentials. Your secrets don't leave the vault.**

VAJRA is an open-source boilerplate that gives your AI agents **ephemeral, scoped, self-destructing credentials** for every tool call. Built as a **single Go binary** — no Docker, no Postgres, no cloud dependencies. Everything is embedded.

> v0.1.0 — All core subsystems built and wired. 28MB static binary. Zero runtime deps.

## The Problem

Every AI agent you build needs access to databases, APIs, and SaaS tools. The default approach — baking API keys into `.env` files or agent memory — is a security disaster waiting for a prompt injection to trigger it.

Static credentials mean:
- One leak = total compromise
- No per-call scope boundaries
- No audit trail for agent actions
- No way to revoke mid-session

## What VAJRA Gives You

```
[ Agent Request ]
       │
       ▼
┌──────────────────────────────────────────────────┐
│  VAJRA Core (single binary, zero deps)           │
│                                                   │
│  ┌──────────┐   ┌──────────┐   ┌──────────────┐ │
│  │Transport  │──►│Credential│──►│  Policy      │ │
│  │ stdio     │   │ Ring     │   │  OPA/Rego    │ │
│  │ HTTP SSE  │   │ Buffer   │   │  Evaluation  │ │
│  └──────────┘   │ (5s TTL) │   └──────────────┘ │
│                 └────┬─────┘                     │
│                      │                            │
│              ┌───────▼────────┐                  │
│              │  Audit Trail   │                  │
│              │  SQLite (append)│                  │
│              └────────────────┘                  │
│                                                   │
│  ┌──────────┐   ┌──────────┐   ┌──────────────┐ │
│  │  Auth    │   │  TUI     │   │  Agent SDK   │ │
│  │OAuth 2.1 │   │ Dashboard│   │  Adapters    │ │
│  │ mTLS     │   │ (Bubble  │   │ LangChain TS │ │
│  │Registry  │   │  Tea)    │   │ LangChain Py │ │
│  └──────────┘   └──────────┘   │ CrewAI       │ │
│                                └──────────────┘ │
└──────────────────────────────────────────────────┘
```

- **MCP-native** — speaks MCP 2026-07-28 over stdio and HTTP SSE
- **Ephemeral credentials** — sub-5-second TTL, zero standing privilege
- **ABAC policies** — row-level, column-level, API-scoped via OPA/Rego
- **Agent SDK adapters** — plug into LangChain (Python/TS), CrewAI, or raw MCP
- **Terminal dashboard** — real-time monitoring via Bubble Tea TUI
- **Full audit trail** — append-only SQLite log of every mint, use, and expiry
- **OAuth 2.1 Device Flow** — auth for headless agents
- **Single binary** — `./vajra`, no Docker, no Postgres, no npm install

## Quick Start

```bash
# Build from source (requires Go 1.24+)
git clone https://github.com/vajra/vajra.git
cd vajra
go build -ldflags="-s -w" -o vajra .

# Or grab a prebuilt binary (coming soon)
# wget https://github.com/vajra/vajra/releases/latest/download/vajra-linux-amd64

# Start the broker daemon
./vajra serve

# Start with SSE transport on port 9735
./vajra serve --sse :9735

# Launch the terminal dashboard (full-screen TUI)
./vajra tui

# Mint a scoped credential manually
./vajra mint --db customers --ttl 5s --select "email,name" --where "plan=pro"

# Watch live credential stream in terminal
./vajra ps

# Check and evaluate OPA/Rego policies
./vajra policy list
./vajra policy test policies/db_scope.rego

# Revoke credentials for an agent
./vajra revoke --agent agent-001
```

## What's Inside

| Package | Lines | Purpose |
|---------|-------|---------|
| `cmd/vajra/` | 7 CLI commands | `serve`, `mint`, `revoke`, `ps`, `policy`, `tui` + root |
| `internal/transport/` | MCP 2026-07-28 | stdio + HTTP SSE transports, full protocol types |
| `internal/credential/` | Ring buffer | In-memory credential store with sub-5s TTL + sweep |
| `internal/policy/` | OPA/Rego | Real policy evaluation engine via `open-policy-agent/opa` |
| `internal/auth/` | Auth subsystem | OAuth 2.1 Device Flow, mTLS, agent registry, token exchange |
| `internal/audit/` | SQLite + stream | Append-only audit log, JSON stdout stream |
| `internal/tui/` | Bubble Tea | Full-screen terminal dashboard (like htop for credentials) |
| `adapters/` | SDK wrappers | LangChain (Python pip package + TS), CrewAI wrapper |
| `policies/` | Rego files | `db_scope.rego`, `api_scope.rego` — ready to edit |
| `docs/` | Architecture + Integrations | Full system design doc, 5 integration patterns |

## Who Is This For?

| Buyer | Problem It Solves | Best Tier |
|-------|-------------------|-----------|
| Indie devs building AI agents | "Don't want to build credential infra from scratch" | Starter ($79) |
| Defense/contractors | "Must work air-gapped, no cloud dependencies" | Pro ($149) |
| AI consultancies | "Need secure agent demos for enterprise clients fast" | Pro ($149) |
| Open-source projects | "Need reference architecture for credential brokering" | Core (free MIT) |

## Pricing

| Tier | Price | What You Get |
|------|-------|-------------|
| **Core** | Free (MIT) | MCP proxy, credential ring buffer, CLI, Rego policies |
| **Starter** | $79 | Core + LangChain adapters (TS + Python) + audit persistence + SSO template |
| **Pro** | $149 | Starter + CrewAI adapter + terminal dashboard + Docker support + priority support |

All tiers ship as single Go binaries. No subscriptions. No cloud. No hidden costs.

## Why VAJRA?

- **Zero runtime deps** — literally just the binary. Not even a shell.
- **Air-gapped native** — works in disconnected environments. Defense and healthcare approved.
- **MCP 2026-07-28 spec** — built for the latest Model Context Protocol standard.
- **OPA/Rego policies** — industry standard policy language, not a custom DSL.
- **Real audit trail** — every credential mint, use, and expiry is logged immutably.
- **Terminal dashboard** — real-time monitoring without a browser (laptop-friendly).
- **Solo-dev scale** — I'm a solo developer building for other developers. Clean code, good docs, no enterprise bloat.

## Build

```bash
go build -ldflags="-s -w" -o vajra .
```

Produces a ~28MB static binary. No build-time deps beyond Go 1.24+.

## License

Core engine: **MIT License** — use it, fork it, learn from it.
Starter/Pro tiers: **Commercial license** (purchased via Gumroad).

---

*Built for developers who ship AI agents and want to keep their secrets secret.*
