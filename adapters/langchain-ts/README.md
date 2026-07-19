# VAJRA LangChain TypeScript Integration

Ephemeral credential injection for LangChain.js agents.

## Install

```bash
npm install vajra-langchain
```

## Usage

```typescript
import { ChatOpenAI } from "@langchain/openai";
import { AgentExecutor, createOpenAIToolsAgent } from "@langchain/core/agents";
import { DynamicTool } from "@langchain/core/tools";
import { VajraToolInterceptor } from "vajra-langchain";

const vajra = new VajraToolInterceptor({
  endpoint: "http://localhost:9735",
  agentId: "my-agent",
  agentSecret: process.env.VAJRA_AGENT_SECRET,
});

const tools = [
  new DynamicTool({
    name: "db_query",
    description: "Query the database",
    func: async (query) => {
      // Credentials are injected transparently
      return await executeQuery(query);
    },
  }),
];

const interceptedTools = tools.map((t) => vajra.intercept(t));
const executor = new AgentExecutor({
  agent: await createOpenAIToolsAgent({
    llm: new ChatOpenAI({ model: "gpt-4o" }),
    tools: interceptedTools,
    prompt,
  }),
  tools: interceptedTools,
});

const result = await executor.invoke({ input: "List all users" });
```

## Environment Variables

- `VAJRA_ENDPOINT` — VAJRA daemon URL
- `VAJRA_AGENT_ID` — Agent identity
- `VAJRA_AGENT_SECRET` — Agent secret key
