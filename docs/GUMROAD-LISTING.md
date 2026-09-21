# VAJRA — MCP Ephemeral Identity Broker (Single Go Binary)

**Price:** $79 Starter / $149 Pro (Core free MIT)
**Platform:** Gumroad (10% + $0.50 per sale)
**License:** MIT (Core) / Commercial (Starter) / White-label (Pro)

---

## Product Description

Your AI agents need database passwords, API keys, and SaaS tokens to do anything useful. The default approach — baking static secrets into `.env` files or agent memory — is a security disaster waiting for a prompt injection to trigger it. One leak, total compromise. No per-call boundaries. No audit trail. No way to revoke mid-session.

VAJRA sits between your agents and their target systems and mints a **fresh, scoped, self-destructing credential for every tool call** — sub-5-second TTL, zero standing privilege. Built as a **single 28MB Go binary**: no Docker, no Postgres, no cloud dependencies, air-gap capable. MCP-native (2026-07-28 spec) over stdio and HTTP SSE, real OPA/Rego policy engine (not a custom DSL), append-only SQLite audit log, Bubble Tea terminal dashboard, and SDK adapters for LangChain (Python + TS) and CrewAI.

Built by a solo developer for developers who ship agents. Clean Go, real tests (14 passing), honest docs. No enterprise bloat, no custom DSLs, no "contact sales".

---

## Features

- Ephemeral credential minting (sub-5s TTL ring buffer + background sweep)
- OPA/Rego ABAC policies (row-level, column-level, API-scoped)
- MCP 2026-07-28 transports (stdio + HTTP SSE)
- 7-command CLI (`serve`, `mint`, `revoke`, `ps`, `policy`, `tui`)
- Terminal dashboard (live credential stream, activity log, uptime)
- Append-only SQLite audit trail (schema-versioned, busy-retry hardened)
- OAuth 2.1 Device Flow + mTLS + agent registry
- LangChain Python + TypeScript adapters, CrewAI wrapper
- Air-gapped operation (single static binary, zero runtime deps)
- Distroless Dockerfile + systemd unit + CI workflow included

---

## What's Included

| Starter ($79) | Pro ($149) |
|---|---|
| Full source + 28MB binary | Everything in Starter |
| MCP proxy + CLI + Rego policies | + CrewAI adapter |
| LangChain adapters (TS + Python) | + Terminal dashboard (TUI) |
| Audit persistence (SQLite) | + Docker + systemd deploy files |
| SSO template | + Priority support |
| Community support | + White-label license |

Core engine (proxy + ring buffer + CLI + policies) is free MIT on GitHub.

---

## Tech Stack

`Go 1.24+` `Cobra` `Bubble Tea` `OPA/Rego` `SQLite` `MCP 2026-07-28` `LangChain` `CrewAI` `Docker`

---

## Requirements

- **To run:** literally nothing — the 28MB static binary (Linux amd64). No shell, no runtime, no network.
- **To build from source:** Go 1.24+
- **To use adapters:** Python 3.11+ (LangChain Py) or Node 20+ (LangChain TS)
- Basic familiarity with AI agent frameworks

---

## FAQ

**Q: How is this different from a secrets manager (Vault, Akeyless)?**

A: Secrets managers store long-lived secrets your agents still have to read. VAJRA eliminates standing privilege entirely — agents never see a real secret, only a 5-second scoped credential. It complements a vault; it doesn't replace one.

**Q: What's actually MIT vs paid?**

A: Core (MCP proxy, ring buffer, CLI, Rego policies) is free MIT on GitHub — use it, fork it, learn from it. Starter adds the LangChain adapters, audit persistence, and SSO template. Pro adds CrewAI, the TUI dashboard, Docker support, white-label rights, and priority support.

**Q: Does it work air-gapped?**

A: Yes. Single static binary, embedded SQLite, local OPA engine. No phone-home, no cloud, no license server. Copy the binary to the isolated host and run.

**Q: Can I use this for client work?**

A: Pro tier includes white-label rights — rebrand it and deploy customized instances on client infrastructure (including air-gapped) as part of paid engagements. Redistribution of the raw source as a boilerplate is not permitted on any tier.

**Q: What if I find a bug?**

A: Priority support on Pro. All paid tiers get v1.x updates via Gumroad re-download. Bugs are fixed within the v1 lifecycle.

**Q: Do you offer refunds?**

A: Yes — 7-day money-back guarantee if the code doesn't work as documented. No questions asked.

**Q: Will agents slow down waiting for credentials?**

A: Minting is local and sub-millisecond. The 5-second TTL is expiry, not latency — agents fetch a credential and use it immediately.

---

## Images/Assets Needed

| Asset | Description | Spec |
|---|---|---|
| Hero screenshot | TUI dashboard with stat cards + live activity log | 1100×750 PNG ✅ `05-tui.png` |
| Feature 1 | CLI help (7 commands) | PNG ✅ `01-help.png` |
| Feature 2 | Live credential mint (scoped, 5s TTL) | PNG ✅ `02-mint.png` |
| Feature 3 | Active credentials + policy list | PNG ✅ `03-ps.png`, `04-policy.png` |
| Demo GIF | Full flow: help → mint → TUI dashboard | GIF ✅ `demo.gif` |
| Web Overview | Dashboard stat cards + live activity | PNG ✅ `06-web-overview.png` |
| Web Audit | Append-only audit trail table | PNG ✅ `07-web-audit.png` |
| Web Agents | Agent registry + active policies | PNG ✅ `08-web-agents.png` |

(All assets exist in `screenshots/` — verified renders from the v0.1.0 binary + live daemon.)

---

## Gumroad Tags

`AI agents`, `MCP`, `Model Context Protocol`, `Go boilerplate`, `credential broker`, `LangChain`, `CrewAI`, `zero trust`, `prompt injection`, `Bubble Tea`, `Rego`, `OPA`, `air-gapped`, `indie hacker`, `boilerplate`

---

## Pricing Tiers — Recommendation

### Starter — $79

For indie devs who want credential infra without building it. Adapters + audit persistence + SSO template. Below the $99 psychological line for a security tool that replaces weeks of infra work.

### Pro — $149

For consultancies and defense-adjacent teams who demo to clients and deploy air-gapped. White-label rights alone justify it — one client engagement recoups the cost 10–50x.

---

*Built for developers who ship AI agents and want to keep their secrets secret.*
