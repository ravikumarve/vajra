# VAJRA LangChain Integration

Ephemeral credential injection for LangChain agents.

## Install

```bash
pip install vajra-langchain
```

## Usage

```python
from langchain.agents import AgentExecutor, create_openai_tools_agent
from langchain_openai import ChatOpenAI
from langchain_community.tools import PostgresTool
from vajra_langchain import VajraToolInterceptor

# Create your tools
tools = [
    PostgresTool(
        connection_string="placeholder://localhost:5432/mydb",
    ),
]

# Wrap with VAJRA
vajra = VajraToolInterceptor(
    endpoint="http://localhost:9735",
    agent_id="my-agent",
    agent_secret=os.getenv("VAJRA_AGENT_SECRET"),
)

intercepted_tools = [vajra.intercept(t) for t in tools]

# Build agent as usual
llm = ChatOpenAI(model="gpt-4o")
agent = create_openai_tools_agent(llm, intercepted_tools, prompt)
executor = AgentExecutor(agent=agent, tools=intercepted_tools)

result = executor.invoke({"input": "List all users"})
```

## Environment Variables

- `VAJRA_ENDPOINT` — VAJRA daemon URL (default: http://localhost:9735)
- `VAJRA_AGENT_ID` — Agent identity
- `VAJRA_AGENT_SECRET` — Agent secret key
- `VAJRA_DEFAULT_TTL` — Credential TTL (default: 5s)
- `VAJRA_POLICY_TAGS` — Comma-separated policy tags
