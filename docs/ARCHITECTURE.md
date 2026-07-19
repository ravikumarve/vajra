# 🗡️ VAJRA — Architecture

## Overview

VAJRA is a **single-process Go binary** that sits between AI agents and their target
systems (databases, APIs, SaaS tools). It intercepts tool calls, mints ephemeral
credentials scoped to the minimum privilege needed, and self-destructs them after
execution. No Docker. No Postgres. No cloud dependencies.

---

## Architecture Diagram

```
┌──────────────────────────────────────────────────────────────┐
│                     AI Agent Process                          │
│  (LangChain / CrewAI / MCP SDK / Custom)                     │
│                                                              │
│  ┌─────────────┐   MCP Request    ┌───────────────────┐      │
│  │ Agent Logic ├──────────────────►│  VAJRA Sidecar    │      │
│  │             │◄──────────────────┤  (localhost:9735) │      │
│  └─────────────┘   Scoped Reply   └─────────┬──────────┘      │
└──────────────────────────────────────────────┼────────────────┘
                                               │
                                               ▼
              ┌──────────────────────────────────────────────────┐
              │               VAJRA Core (Go binary)              │
              │                                                  │
              │  ┌────────────────┐    ┌────────────────────┐    │
              │  │ MCP Transport  │    │ OAuth 2.1 Device   │    │
              │  │ (stdio / SSE)  │    │ Flow Handler       │    │
              │  └───────┬────────┘    └─────────┬──────────┘    │
              │          │                       │               │
              │          ▼                       ▼               │
              │  ┌─────────────────────────────────────────┐     │
              │  │        Credential Ring Buffer            │     │
              │  │  (in-memory, sub-5s TTL, fixed size)    │     │
              │  │  ┌─────┐ ┌─────┐ ┌─────┐ ┌─────┐       │     │
              │  │  │tok1 │ │tok2 │ │tok3 │ │...  │       │     │
              │  │  └─────┘ └─────┘ └─────┘ └─────┘       │     │
              │  └──────────────────┬──────────────────────┘     │
              │                     │                            │
              │                     ▼                            │
              │  ┌─────────────────────────────────────────┐     │
              │  │      OPA/Rego Policy Engine              │     │
              │  │  ┌─────────┐ ┌──────────┐ ┌─────────┐  │     │
              │  │  │ RBAC    │ │ ABAC     │ │ Row/Col │  │     │
              │  │  │ Rules   │ │Context   │ │Scoping  │  │     │
              │  │  └─────────┘ └──────────┘ └─────────┘  │     │
              │  └──────────────────┬──────────────────────┘     │
              │                     │                            │
              │                     ▼                            │
              │  ┌─────────────────────────────────────────┐     │
              │  │      SQLite Audit Log                    │     │
              │  │  (append-only, auto-rotate, JSON rows)   │     │
              │  └──────────────────────────────────────────┘     │
              │                                                  │
              │  ┌──────────────────────────────────────────┐     │
              │  │      Bubble Tea TUI (optional)             │     │
              │  │  Live audit stream + policy monitor       │     │
              │  └──────────────────────────────────────────┘     │
              └──────────────────┬───────────────────────────────┘
                                 │
                                 ▼
              ┌──────────────────────────────────────────────────┐
              │              Target Systems                       │
              │                                                  │
              │  ┌────────┐ ┌──────────┐ ┌────────┐ ┌────────┐  │
              │  │Postgres│ │  MySQL   │ │REST API│ │ SaaS   │  │
              │  │        │ │          │ │        │ │ (Hub-  │  │
              │  │        │ │          │ │        │ │ spot)  │  │
              │  └────────┘ └──────────┘ └────────┘ └────────┘  │
              └──────────────────────────────────────────────────┘
```

---

## Request Lifecycle

```
Step 1: Agent sends tool request to VAJRA via MCP
         {
           "tool": "db_query",
           "args": { "table": "users", "columns": ["email", "name"] }
         }

Step 2: VAJRA authenticates agent identity
         - OAuth 2.1 Device Flow for human-linked agents
         - mTLS certificate for headless agents
         - Falls to deny if neither presented

Step 3: OPA/Rego evaluates policy against request
         - "Can this agent SELECT from users?"
         - "Which columns is it allowed to read?"
         - "Any row-level filters required (e.g. WHERE org_id = agent.org)?"
         - "What TTL is appropriate for this operation?"

Step 4: VAJRA mints scoped credential
         - If target is DB: CREATE USER + GRANT SELECT ON users(email,name)
           with TTL = 5s
         - If target is API: signed JWT with scoped claims, 5s expiry
         - Credential stored in ring buffer, keyed by request ID

Step 5: VAJRA proxies the request with injected credential
         - Intercepts the original MCP tool call
         - Swaps the credential placeholder for the real ephemeral credential
         - Forwards to target system

Step 6: Response returned, credential self-destructs
         - Ring buffer evicts after TTL expiry or explicit revoke
         - Full audit record written to SQLite
         - Agent never saw the real credential
```

---

## Module Responsibilities

### 1. MCP Transport Layer (`internal/transport/`)

| Component | File | Responsibility |
|-----------|------|----------------|
| stdio server | `stdio.go` | For local agent processes (pipe-based MCP) |
| HTTP SSE server | `sse.go` | For remote agents or multi-process setups |
| Protocol parser | `mcp.go` | JSON-RPC message framing, MCP 2026-07-28 compliance |

### 2. Identity & Auth (`internal/auth/`)

| Component | File | Responsibility |
|-----------|------|----------------|
| OAuth 2.1 Device Flow | `device.go` | Browserless auth for agent identity proof |
| mTLS handler | `mtls.go` | Certificate-based identity for headless agents |
| Agent registry | `registry.go` | Maps agent IDs to policies, stored in SQLite |
| Token exchange (RFC 8693) | `exchange.go` | Delegation chains: human → agent → VAJRA |

### 3. Credential Engine (`internal/credential/`)

| Component | File | Responsibility |
|-----------|------|----------------|
| Ring buffer | `ring.go` | O(1) insert/evict, fixed capacity, TTL sweep goroutine |
| Token mint | `mint.go` | Database user provisioning, JWT issuance, API key generation |
| Token revoke | `revoke.go` | Immediate invalidation + buffer eviction |

### 4. Policy Engine (`internal/policy/`)

| Component | File | Responsibility |
|-----------|------|----------------|
| OPA runtime | `opa.go` | Embedded Rego interpreter (github.com/open-policy-agent/opa) |
| Rule loader | `rules.go` | Loads `.rego` files from `~/.vajra/policies/` |
| Built-in predicates | `predicates.go` | Time-of-day, source IP, agent identity context |
| Decision cache | `cache.go` | Short-lived cache for repeated policy evaluations |

### 5. Audit & Observability (`internal/audit/`)

| Component | File | Responsibility |
|-----------|------|----------------|
| SQLite writer | `sqlite.go` | Append-only log, auto-rotate at 100MB |
| JSON stream | `stream.go` | stdout JSON lines for log aggregators (Vector, Loki, etc.) |
| OTEL exporter | `otel.go` | OpenTelemetry-compatible traces (optional, Pro tier) |

### 6. CLI & TUI (`cmd/vajra/` and `internal/tui/`)

| Component | File | Responsibility |
|-----------|------|----------------|
| Root command | `main.go` | cobra CLI entry point |
| `vajra serve` | `serve.go` | Start the daemon |
| `vajra mint` | `mint.go` | Manual credential request (debugging) |
| `vajra ps` | `ps.go` | Live audit stream + active credentials |
| `vajra policy` | `policy.go` | List/add/test Rego policies |
| `vajra revoke` | `revoke.go` | Immediately invalidate a credential |
| TUI dashboard | `dashboard.go` | Bubble Tea real-time terminal UI |

---

## Data Flow Diagram

```
Agent                    VAJRA                        Target
  │                        │                            │
  │── MCP tool call ──────►│                            │
  │                        │                            │
  │                        ├── Verify agent identity    │
  │                        ├── Evaluate OPA policy      │
  │                        ├── Mint ephemeral credential │
  │                        ├── Swap credential in call   │
  │                        │── proxied request ────────►│
  │                        │                            │
  │                        │◄──── response ────────────│
  │                        │                            │
  │                        ├── Evict credential         │
  │                        ├── Write audit log          │
  │◄── scoped response ────┤                            │
  │                        │                            │
```

---

## Tech Stack Rationale

| Decision | Why |
|----------|-----|
| **Go over Rust** | Faster compile times, excellent stdlib, lower mental overhead for a solo dev. Go's net/http and crypto/tls are production-grade with zero deps. |
| **Single binary** | Buyers run ONE file. No Python runtime, no Node.js, no JVM, no Docker. `./vajra serve` and done. |
| **SQLite over Postgres** | Zero dependencies. The binary embeds the database. modernc.org/sqlite means no CGO, pure Go. |
| **OPA/Rego over custom DSL** | Industry standard policy engine. Buyers who know OPA can bring their existing policies. Those who don't can learn from OPA's excellent docs — not from my half-baked custom language. |
| **In-memory ring buffer** | Credentials live for <5 seconds. Persisting them to disk creates forensic liability. A fixed-size ring buffer with a TTL sweep goroutine is simpler and more secure. |
| **Bubble Tea over React** | A web dashboard requires a browser, a port, CORS config, and a frontend build. A TUI works over SSH, in tmux, in CI logs. It's also vastly faster to build. |
| **MCP 2026-07-28** | The spec adds stateless transports and incremental scope consent. Building to the latest spec means the boilerplate doesn't need a breaking rewrite in 6 months. |

---

## Deployment Model

### Local (Single Machine) — Default

```bash
# Download and run. That's it.
./vajra serve
# Agent connects to localhost:9735
```

### Air-Gapped

```bash
# Transfer binary via USB or internal package repo
scp vajra admin@airgap-host:~/
./vajra serve --no-telemetry
# No internet access needed at runtime
```

### Sidecar (Container/VM)

```dockerfile
FROM alpine:3.20
COPY vajra /usr/local/bin/
EXPOSE 9735
ENTRYPOINT ["vajra", "serve"]
```

### System Service

```bash
sudo cp vajra /usr/local/bin/
sudo cp contrib/vajra.service /etc/systemd/system/
sudo systemctl enable --now vajra
```

---

## File Layout

```
vajra/
├── cmd/
│   └── vajra/
│       ├── main.go           # CLI entry point (cobra)
│       ├── serve.go          # `vajra serve` daemon
│       ├── mint.go           # `vajra mint` manual issuance
│       ├── ps.go             # `vajra ps` audit stream
│       ├── policy.go         # `vajra policy` management
│       └── revoke.go         # `vajra revoke` manual revoke
├── internal/
│   ├── transport/            # MCP protocol (stdio + HTTP SSE)
│   │   ├── mcp.go
│   │   ├── stdio.go
│   │   └── sse.go
│   ├── auth/                 # Agent identity & authentication
│   │   ├── device.go         # OAuth 2.1 Device Flow
│   │   ├── mtls.go           # mTLS certificate handler
│   │   ├── registry.go       # Agent identity store
│   │   └── exchange.go       # Token exchange (RFC 8693)
│   ├── credential/           # Ephemeral credential engine
│   │   ├── ring.go           # In-memory ring buffer
│   │   ├── mint.go           # Credential provisioning
│   │   └── revoke.go         # Immediate invalidation
│   ├── policy/               # OPA/Rego policy engine
│   │   ├── opa.go            # Embedded OPA runtime
│   │   ├── rules.go          # Policy file loader
│   │   ├── predicates.go     # Built-in context predicates
│   │   └── cache.go          # Decision caching
│   ├── audit/                # Audit & observability
│   │   ├── sqlite.go         # Append-only audit log
│   │   ├── stream.go         # JSON stdout stream
│   │   └── otel.go           # OpenTelemetry exporter
│   └── tui/                  # Bubble Tea terminal UI
│       ├── dashboard.go      # Main TUI component
│       ├── audit_view.go     # Audit log viewer
│       └── policy_view.go    # Policy monitor
├── adapters/                 # Agent framework integrations
│   ├── langchain-py/         # Python LangChain callback
│   ├── langchain-ts/         # TypeScript LangChain callback
│   └── crewai/               # CrewAI tool wrapper
├── policies/                 # Default Rego policy files
│   ├── db_scope.rego         # Row/column scoping templates
│   └── api_scope.rego        # API method/endpoint scoping
├── contrib/                  # Platform integration extras
│   ├── vajra.service         # systemd unit file
│   └── vajra.conf            # Default configuration
├── go.mod
├── go.sum
├── main.go                   # Build entry point
└── README.md
```

---

## Configuration

VAJRA is configured via a single YAML file at `~/.vajra/config.yaml`:

```yaml
# ~/.vajra/config.yaml

listen:
  mcp_sse: ":9735"           # MCP HTTP SSE endpoint
  mcp_stdio: true             # Enable stdio transport

storage:
  audit_db: "~/.vajra/audit.db"
  audit_rotate_mb: 100

identity:
  oauth_providers:
    - provider: "google"
      client_id: "${VAJRA_GOOGLE_CLIENT_ID}"
  mtls_ca: "~/.vajra/ca.pem"

policies:
  dir: "~/.vajra/policies/"
  default_ttl: "5s"

ring_buffer:
  capacity: 10000             # Max concurrent active credentials
  sweep_interval: "1s"        # TTL check frequency
```

---

## Security Model

| Property | Implementation |
|----------|---------------|
| Credential at rest | Never persisted. Ring buffer lives in process heap. |
| Credential in transit | TLS 1.3 between agent and VAJRA, TLS 1.3 between VAJRA and target |
| Audit integrity | Append-only SQLite, CRC-checked rows, periodic WAL checkpoint |
| Policy isolation | OPA evaluates in a sandboxed Rego interpreter |
| Agent identity proof | OAuth 2.1 Device Flow (human-attended) or mTLS (headless) |
| Memory protection | `mlock()` on ring buffer pages to prevent swap leakage |
| Minimal attack surface | Single binary, no shell exec, no plugin loading at runtime |
