# 🗡️ VAJRA — Integration Guide

How to plug VAJRA into your AI agent stack.

---

## Table of Contents

- [Architecture Pattern](#architecture-pattern)
- [LangChain (Python)](#langchain-python)
- [LangChain (TypeScript)](#langchain-typescript)
- [CrewAI](#crewai)
- [Raw MCP SDK (Python)](#raw-mcp-sdk-python)
- [Raw MCP SDK (TypeScript)](#raw-mcp-sdk-typescript)
- [curl / HTTP Demo](#curl--http-demo)
- [Agent Adapter Configuration](#agent-adapter-configuration)
- [Troubleshooting](#troubleshooting)

---

## Architecture Pattern

All integrations follow the same pattern:

```
Agent Framework ──► VAJRA Adapter ──► VAJRA Core ──► Target System
                       │
                       ├── Wraps tool call in MCP request
                       ├── VAJRA intercepts, mints ephemeral credential
                       ├── Proxies to target with scoped access
                       └── Returns response, auto-revokes credential
```

The adapter is a thin wrapper. It adds ~5 lines of code to your existing agent setup.

---

## LangChain (Python)

### Installation

```bash
pip install vajra-langchain
```

### Basic Usage

```python
from langchain.agents import AgentExecutor, create_openai_tools_agent
from langchain_openai import ChatOpenAI
from vajra_langchain import VajraToolInterceptor

# 1. Create your tools as usual
from langchain_community.tools import PostgresTool

tools = [
    PostgresTool(
        connection_string="placeholder://localhost:5432/mydb",
        # VAJRA will replace the placeholder with a real ephemeral credential
    ),
]

# 2. Wrap with VAJRA interceptor (this is the only change)
vajra = VajraToolInterceptor(
    endpoint="http://localhost:9735",  # VAJRA core
    agent_id="my-agent-001",
    agent_secret=os.getenv("VAJRA_AGENT_SECRET"),
)

intercepted_tools = [vajra.intercept(t) for t in tools]

# 3. Build agent as usual
llm = ChatOpenAI(model="gpt-4o")
agent = create_openai_tools_agent(llm, intercepted_tools, prompt)
executor = AgentExecutor(agent=agent, tools=intercepted_tools)

# 4. Run — each tool call gets a fresh ephemeral credential
result = executor.invoke({"input": "List all users in the pro plan"})
```

### Configuration via Environment

```bash
# Required (values below are placeholders — mint a real secret per agent:
# `vajra mint --target self --scope agent --ttl 8760h`, then store it in
# your agent host env or secret manager, never in git)
VAJRA_ENDPOINT="http://localhost:9735"
VAJRA_AGENT_ID="my-agent-001"
VAJRA_AGENT_SECRET="<minted-per-agent>"

# Optional
VAJRA_DEFAULT_TTL="5s"           # Default credential lifetime
VAJRA_POLICY_TAGS="read-only"    # Comma-separated policy tags
```

### Advanced: Custom Policies Per Tool

```python
# Database tool with row-level scoping
db_tool = PostgresTool(
    connection_string="placeholder://localhost:5432/mydb",
)

vajra_db = VajraToolInterceptor(
    endpoint="http://localhost:9735",
    agent_id="my-agent-001",
    agent_secret=os.getenv("VAJRA_AGENT_SECRET"),
    # Policy overrides for this specific tool
    policy_overrides={
        "db_scope": {
            "allowed_tables": ["users", "organizations"],
            "allowed_columns": ["email", "name", "plan"],
            "row_filter": "org_id = {{ agent.org_id }}",
        },
        "ttl": "3s",  # Shorter TTL for sensitive data
    },
)

# API tool with endpoint scoping
api_tool = RequestsTool(
    base_url="https://api.example.com",
)

vajra_api = VajraToolInterceptor(
    endpoint="http://localhost:9735",
    agent_id="my-agent-001",
    agent_secret=os.getenv("VAJRA_AGENT_SECRET"),
    policy_overrides={
        "api_scope": {
            "allowed_methods": ["GET"],
            "allowed_paths": ["/v2/users", "/v2/organizations/*"],
            "rate_limit": "100/min",
        },
    },
)
```

---

## LangChain (TypeScript)

### Installation

```bash
npm install vajra-langchain
```

### Basic Usage

```typescript
import { ChatOpenAI } from "@langchain/openai";
import { AgentExecutor, createOpenAIToolsAgent } from "@langchain/core/agents";
import { DynamicTool } from "@langchain/core/tools";
import { VajraToolInterceptor } from "vajra-langchain";

// 1. Create VAJRA interceptor
const vajra = new VajraToolInterceptor({
  endpoint: "http://localhost:9735",
  agentId: "my-agent-001",
  agentSecret: process.env.VAJRA_AGENT_SECRET,
});

// 2. Define tools with placeholder credentials
const tools = [
  new DynamicTool({
    name: "db_query",
    description: "Query the PostgreSQL database",
    func: async (query) => {
      // This function is wrapped by VAJRA — credentials are
      // injected at the network layer before execution
      return await executeQuery(query);
    },
  }),
];

// 3. Wrap tools with VAJRA
const interceptedTools = tools.map((t) => vajra.intercept(t));

// 4. Build and run agent
const llm = new ChatOpenAI({ model: "gpt-4o" });
const agent = await createOpenAIToolsAgent({
  llm,
  tools: interceptedTools,
  prompt,
});
const executor = new AgentExecutor({
  agent,
  tools: interceptedTools,
});

const result = await executor.invoke({
  input: "List all users in the pro plan",
});
```

### Configuration

```typescript
const vajra = new VajraToolInterceptor({
  endpoint: process.env.VAJRA_ENDPOINT || "http://localhost:9735",
  agentId: process.env.VAJRA_AGENT_ID,
  agentSecret: process.env.VAJRA_AGENT_SECRET,
  defaultTtl: "5s",
  policyTags: process.env.VAJRA_POLICY_TAGS?.split(","),
});
```

---

## CrewAI

### Installation

```bash
pip install vajra-crewai
```

### Basic Usage

```python
from crewai import Agent, Task, Crew
from crewai_tools import tool
from vajra_crewai import VajraToolWrapper

# 1. Create your tool
@tool("Database Query")
def query_database(sql: str) -> str:
    """Execute a SQL query on the customer database."""
    # VAJRA replaces the placeholder connection
    return execute_with_vajra(sql)

# 2. Wrap with VAJRA
vajra = VajraToolWrapper(
    endpoint="http://localhost:9735",
    agent_id="data-analyst-001",
    agent_secret=os.getenv("VAJRA_AGENT_SECRET"),
    policy_tags=["read-only", "analytics"],
)

db_tool = vajra.wrap(query_database)

# 3. Create CrewAI agent with wrapped tool
analyst = Agent(
    role="Data Analyst",
    goal="Answer business questions from the database",
    backstory="Senior analyst with SQL expertise",
    tools=[db_tool],
    allow_delegation=False,
)

# 4. Run the crew
task = Task(
    description="Find the top 5 customers by MRR",
    expected_output="List of customer names and MRR values",
    agent=analyst,
)

crew = Crew(agents=[analyst], tasks=[task])
result = crew.kickoff()
```

### Multi-Agent Crew

```python
# Each agent gets its own scoped VAJRA wrapper
analyst_tools = VajraToolWrapper(
    endpoint="http://localhost:9735",
    agent_id="analyst-001",
    agent_secret=os.getenv("VAJRA_ANALYST_SECRET"),
    policy_tags=["read-only", "analytics"],
)

writer_tools = VajraToolWrapper(
    endpoint="http://localhost:9735",
    agent_id="writer-001",
    agent_secret=os.getenv("VAJRA_WRITER_SECRET"),
    policy_tags=["read-only", "summary"],
)

analyst = Agent(
    role="Data Analyst",
    tools=[analyst_tools.wrap(query_database)],
)

writer = Agent(
    role="Report Writer",
    tools=[writer_tools.wrap(get_summary)],
)
# Writer can access summaries but NOT raw queries
```

---

## Raw MCP SDK (Python)

### Installation

```bash
pip install vajra-mcp
```

### Integration

```python
from vajra_mcp import VajraMCPClient

# 1. Connect to VAJRA via MCP
client = VajraMCPClient(
    server_url="http://localhost:9735/mcp",
    agent_id="my-agent",
    agent_secret=os.getenv("VAJRA_AGENT_SECRET"),
)

# 2. List available resources (scoped by policy)
resources = client.list_resources()
# => [
#   { "name": "users", "type": "db:table", "columns": ["email", "name"] },
#   { "name": "organizations", "type": "db:table", "columns": ["id", "name"] },
# ]

# 3. Call a tool — VAJRA mints credential transparently
result = client.call_tool(
    name="db_query",
    arguments={
        "table": "users",
        "columns": ["email", "name"],
        "filter": {"plan": "pro"},
    },
)

# 4. Done. Credential is already revoked.
print(result)
```

### Async Support

```python
import asyncio
from vajra_mcp import VajraMCPClient

async def main():
    async with VajraMCPClient(
        server_url="http://localhost:9735/mcp",
        agent_id="my-agent",
        agent_secret=os.getenv("VAJRA_AGENT_SECRET"),
    ) as client:
        result = await client.call_tool_async(
            name="db_query",
            arguments={
                "table": "users",
                "columns": ["email", "name"],
            },
        )
        print(result)

asyncio.run(main())
```

---

## Raw MCP SDK (TypeScript)

### Installation

```bash
npm install vajra-mcp
```

### Integration

```typescript
import { VajraMCPClient } from "vajra-mcp";

const client = new VajraMCPClient({
  serverUrl: "http://localhost:9735/mcp",
  agentId: "my-agent",
  agentSecret: process.env.VAJRA_AGENT_SECRET,
});

// List available tools
const tools = await client.listTools();
console.log(tools);

// Call a database query
const result = await client.callTool({
  name: "db_query",
  arguments: {
    table: "users",
    columns: ["email", "name"],
    filter: { plan: "pro" },
  },
});

// Credential is auto-revoked after response
console.log(result);
```

---

## curl / HTTP Demo

Test VAJRA without any framework:

```bash
# 1. Start VAJRA
./vajra serve

# 2. Authenticate agent (OAuth 2.1 Device Flow)
curl -X POST http://localhost:9735/auth/device \
  -H "Content-Type: application/json" \
  -d '{"agent_id": "demo-agent"}'
# => { "device_code": "abc123", "verification_uri": "...", "interval": 5 }

# 3. Mint a scoped credential manually
curl -X POST http://localhost:9735/v1/mint \
  -H "Authorization: Bearer <agent_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "target": {
      "type": "postgres",
      "host": "localhost",
      "database": "customers"
    },
    "scope": {
      "tables": ["users"],
      "columns": ["email", "name"],
      "row_filter": "plan = 'pro'"
    },
    "ttl": "5s"
  }'
# => { "credential": "postgres://user_abc:tok_xyz@localhost:5432/customers",
#      "expires_at": "2026-07-17T18:00:05Z" }

# 4. Use the credential (works for 5 seconds)
psql "postgres://user_abc:tok_xyz@localhost:5432/customers" \
  -c "SELECT email, name FROM users WHERE plan = 'pro';"

# 5. Check audit log
curl http://localhost:9735/v1/audit?tail=10
# => JSON array of recent credential events
```

---

## Agent Adapter Configuration

### Shared Configuration (YAML)

```yaml
# ~/.vajra/config.yaml (on VAJRA host)

agents:
  - id: "my-agent-001"
    name: "Customer Support Agent"
    secret_hash: "$2a$10$..."   # bcrypt hash of agent secret
    policies:
      - role: "support_agent"
    identity:
      type: "oauth"
      provider: "google"
      user_id: "admin@example.com"
      # Token exchange: human user delegates to agent

  - id: "analyst-001"
    name: "Data Analyst Agent"
    secret_hash: "$2a$10$..."
    policies:
      - role: "read_only_analyst"
    identity:
      type: "mtls"
      cert_fingerprint: "sha256:..."
```

### Environment Variables (on Agent host)

```bash
# Required — tells the adapter where to find VAJRA (secret minted per agent,
# see "Configuration via Environment" above — never commit real secrets)
export VAJRA_ENDPOINT="http://localhost:9735"
export VAJRA_AGENT_ID="my-agent-001"
export VAJRA_AGENT_SECRET="<minted-per-agent>"

# Optional — policy overrides
export VAJRA_DEFAULT_TTL="5s"
export VAJRA_POLICY_TAGS="read-only,production"
```

---

## Troubleshooting

### "Connection refused" when starting agent

```bash
# VAJRA is not running. Start it first:
./vajra serve

# Or check if it's on a different port:
curl http://localhost:9735/v1/health
```

### "Agent not authorized"

```bash
# Check that the agent is registered in VAJRA:
./vajra policy list
# => Lists registered agents and their policies

# Register a new agent:
./vajra policy add-agent --id "my-agent-001" --role "support_agent"
```

### "Credential expired"

```bash
# Default TTL is 5 seconds. If your agent takes longer:
# 1. Set a longer TTL in the adapter:
export VAJRA_DEFAULT_TTL="30s"

# 2. Or check for network latency between agent and target:
./vajra ps  # Shows active credentials and remaining TTL
```

### "Policy denied"

```bash
# Test a policy decision manually:
./vajra policy test --agent "my-agent-001" \
  --action "db_query" \
  --resource "users" \
  --context '{"columns": ["email", "ssn"]}'
# => DENIED: agent "my-agent-001" cannot access column "ssn"

# View active policies:
./vajra policy show --agent "my-agent-001"
```

### Audit log location

```bash
# Default: ~/.vajra/audit.db
# View recent entries:
./vajra ps --tail 20

# Raw SQLite access:
sqlite3 ~/.vajra/audit.db "SELECT * FROM audit ORDER BY id DESC LIMIT 10;"
```
