# Vajra

> Ephemeral, scoped, self-destructing credentials for AI agents — one line: every tool call your agents make gets a fresh credential that dies in seconds, so a leaked key is already dead.

## What it is

Vajra is a **boilerplate MCP identity broker** for developers shipping AI agents.
It sits between your agents and their target systems (databases, APIs, SaaS
tools) and mints **short-lived, scope-narrowed credentials per tool call** —
sub-5-second TTL, zero standing privilege. It ships as a **single 28MB Go
binary**: no Docker, no Postgres, no cloud dependencies, air-gap capable.

## Who it's for

| Buyer | Problem it solves |
|-------|-------------------|
| Indie devs building AI agents | Credential infra without building it from scratch |
| Defense / contractors | Must work air-gapped, no cloud dependencies |
| AI consultancies | Secure agent demos for enterprise clients, fast |
| Open-source projects | Reference architecture for credential brokering (Core, free MIT) |

## Key features

- **MCP-native** — speaks MCP 2026-07-28 over stdio and HTTP SSE
- **Ephemeral credentials** — ring buffer store, sub-5s TTL + background sweep
- **ABAC policies** — row/column/API scope via a real OPA/Rego engine (no custom DSL)
- **Agent SDK adapters** — LangChain Python + TypeScript, CrewAI, raw MCP
- **Terminal dashboard** — Bubble Tea TUI, real-time credential monitoring
- **Full audit trail** — append-only SQLite log of every mint, use, expiry, and revocation (with busy-retry + schema versioning)
- **OAuth 2.1 Device Flow + mTLS** — auth for headless agents, agent registry, token exchange
- **7 CLI commands** — `serve`, `mint`, `revoke`, `ps`, `policy`, `tui` (+ root)

## Pricing / Distribution

Gumroad one-time purchase (no subscriptions) + GitHub (MIT Core):

| Tier | Price | What you get |
|------|-------|--------------|
| **Core** | Free (MIT) | MCP proxy, credential ring buffer, CLI, Rego policies |
| **Starter** | $79 | Core + LangChain adapters (TS + Python) + audit persistence + SSO template |
| **Pro** | $149 | Starter + CrewAI adapter + terminal dashboard + Docker support + priority support |

Paid tiers ship as single Go binaries with the Commercial license (`LICENSE`).
Core engine files are MIT (`LICENSE-MIT`).

## Stack

Go 1.24+ (Cobra CLI, Bubble Tea TUI), OPA/Rego (`open-policy-agent/opa`),
SQLite (append-only audit, `PRAGMA user_version` migrations), MCP 2026-07-28
(stdio + HTTP SSE). Adapters: Python (pip) + TypeScript + CrewAI. Distroless
Dockerfile + systemd unit in `contrib/`.

## Status

v0.1.0 — built. All core subsystems wired. `go build` ✅, `go vet` clean ✅,
14/14 Go tests ✅ (policy, audit retry+roundtrip, ring TTL/revoke, device flow).
Product verifier **74/100 PASS (29 WARN / 0 FAIL)** — remaining WARNs are honest
(stdlib CLI by design; SQLite/OPA API-name false positives).
