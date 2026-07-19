# 🗡️ VAJRA — MCP Ephemeral Identity Boilerplate

**Tagline:** *Open-source ephemeral identity broker for AI agents — deploy as a single Go binary, zero runtime dependencies.*

**Target Buyer:** Developers building AI agents who need credential security without building from scratch.

**Distribution:** Gumroad ($79 starter / $149 pro) + GitHub (MIT core)

---

## 1. Executive Summary

As autonomous AI agents gain API and database access, static environment secrets (`.env` files, hardcoded keys, long-lived service accounts) are the #1 attack vector under prompt injection. Every industry report — OWASP, NIST, CISA, Cloud Security Alliance — now mandates **ephemeral, just-in-time, scope-narrowed credentials** as the only durable defense.

VAJRA is a **boilerplate** (open-source starter kit) that gives developers a production-ready MCP identity broker they can deploy in minutes. Not a SaaS platform, not a Docker-heavy enterprise suite — a single Go binary that any developer can drop into their agent stack.

## 2. What Changed (v1 → v2)

| v1 (Enterprise SaaS) | v2 (Boilerplate) |
|----------------------|-------------------|
| $1K–$4K/month per cluster | $79–$149 one-time purchase |
| SOC2 / HIPAA compliance needed | No compliance paperwork — it's code |
| 6-18 month enterprise sales cycle | 24-hour impulse buy by developers |
| Docker / Kubernetes required | Single Go binary, zero deps |
| Full web dashboard | Terminal UI (Bubble Tea) |
| VAJRA Vault + VAJRA Cloak included | CUT from v1 — focus on credential brokering |

### Why the Pivot?

The original idea.md described an enterprise security product with 11 well-funded competitors (CyberArk, Akeyless, HashiCorp, NewCore, Alter, Aembit, OneCLI, etc.). For a solo developer on a Dell Latitude 3460, going head-to-head with VC-backed teams is a guaranteed loss.

The **boilerplate** angle flips the game:
- **Zero direct competitors** on Gumroad or CodeCudos for AI agent identity
- **Solo dev advantage** — I build for other solo devs, not procurement departments
- **$50M+ boilerplate market** growing annually, and this niche is empty

## 3. Architecture

```
[ Agent (LangChain/CrewAI/MCP SDK) ]
        │
        ▼
┌───────────────────────────────┐
│        VAJRA Core             │
│  (Go binary, runs as daemon)  │
│                               │
│  ┌─────────────────────┐      │
│  │  MCP Proxy (stdio   │      │
│  │  + HTTP SSE)        │      │
│  └────────┬────────────┘      │
│           ▼                   │
│  ┌─────────────────────┐      │
│  │  Credential Ring     │      │
│  │  Buffer (TTL <5s)   │      │
│  └────────┬────────────┘      │
│           ▼                   │
│  ┌─────────────────────┐      │
│  │  OPA/Rego Policy    │      │
│  │  Engine (ABAC)      │      │
│  └────────┬────────────┘      │
│           ▼                   │
│  ┌─────────────────────┐      │
│  │  SQLite Audit Log   │      │
│  └─────────────────────┘      │
└───────────────┬───────────────┘
                │
                ▼
     [ Database / API / SaaS ]
```

## 4. Core Modules (v1 Scope)

| Module | What It Does | Status |
|--------|-------------|--------|
| **VAJRA Mint** | MCP proxy intercept + credential ring buffer + OAuth 2.1 Device Flow | ✅ v1 CORE |
| **OPA Policy Engine** | ABAC rule evaluation at request time (row-level, column-level, API-scoped) | ✅ v1 CORE |
| **LangChain Adapter** | Python + TypeScript callback handler for LangChain agents | ✅ v1 CORE |
| **CrewAI Adapter** | Tool wrapper for CrewAI agent crews | ✅ v1 STARTER |
| **TUI Dashboard** | Bubble Tea terminal UI for audit stream + credential monitoring | ✅ v1 PRO |
| **CLI Tool** | `vajra mint`, `vajra ps`, `vajra policy`, `vajra revoke` | ✅ v1 CORE |
| **VAJRA Cloak** | PII redaction on outbound payloads | ❌ Deferred to v2 |
| **VAJRA Vault** | In-memory state encryption | ❌ Deferred to v2 |

## 5. Tech Stack

| Component | Choice | Rationale |
|-----------|--------|-----------|
| Language | Go 1.24+ | Single binary, fast compile, excellent stdlib, no runtime |
| MCP Transport | stdio + HTTP SSE | MCP 2026-07-28 compliant |
| Policy Engine | OPA/Rego (embedded via github.com/open-policy-agent/opa) | Industry standard, not a custom DSL |
| Credential Store | In-memory ring buffer | Sub-5s TTL, no persistence needed |
| Audit Persistence | SQLite (modernc.org/sqlite) | Zero-dependency embedded DB |
| TUI | charmbracelet/bubbletea | Terminal UI, no browser needed |
| CLI | cobra + pterm | Developer-friendly command-line interface |

## 6. Distribution & Pricing

| Tier | Price | Channel | Includes |
|------|-------|---------|----------|
| **Core** | FREE | GitHub (MIT) | MCP proxy + credential ring buffer + CLI |
| **Starter** | $79 | Gumroad | Core + OPA/Rego policies + LangChain adapters |
| **Pro** | $149 | Gumroad | Starter + CrewAI adapter + TUI dashboard + enterprise SSO |

All tiers: single static Go binary. `sudo apt install vajra && ./vajra serve`. That's it.

## 7. Market Validation

| Signal | Source |
|--------|--------|
| 74% of orgs deploy AI agents that need credentials | SANS 2026 NHI Survey |
| 76% report NHI growth; 82:1 machine-to-human identity ratio | CyberArk 2025/2026 |
| 40% of enterprise apps will embed task-specific AI agents by EOY 2026 | Gartner |
| AI agent security startups: $3.6B raised across top 10 | CB Insights (Mar 2026) |
| SaaS boilerplate market: $50M+/year, growing | Industry estimates |
| Zero dedicated AI agent identity boilerplates on Gumroad/CodeCudos | Direct research |

## 8. Commercial Strategy

- **Build audience first** (3 Dev.to posts, HN launch, Twitter/X build-in-public)
- **Ship MVP in 4 weeks** — MCP proxy, credential buffer, OPA policies, LangChain adapter
- **Launch on Gumroad + GitHub** — free MIT core draws users, paid tiers convert power users
- **Revenue target:** $15K–$40K year one (200 starter × $79 + 100 pro × $149)
- **No outside investment** — this is a solo-dev lifestyle business, not a VC rocket ship

## 9. Acquisition / Exit Potential

Acquisition is not the goal, but if the project gains traction, natural acquirers would be companies that need developer mindshare in the MCP/agent identity space:
- **HashiCorp** — Vault already has AI agent features, a developer-friendly boilerplate extends reach
- **CodeCudos / Gumroad** — platform-native acquisition to own the boilerplate category
- **LangChain / CrewAI** — framework-native security templates

But the primary goal: **sustainable solo-dev revenue building useful software.**
