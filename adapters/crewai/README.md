# VAJRA CrewAI Integration

Ephemeral credential injection for CrewAI agents.

## Install

```bash
pip install vajra-crewai
```

## Usage

```python
from crewai import Agent, Task, Crew
from crewai_tools import tool
from vajra_crewai import VajraToolWrapper

@tool("Database Query")
def query_database(sql: str) -> str:
    """Execute a SQL query."""
    return execute_with_vajra(sql)

vajra = VajraToolWrapper(
    endpoint="http://localhost:9735",
    agent_id="analyst-001",
    agent_secret=os.getenv("VAJRA_AGENT_SECRET"),
    policy_tags=["read-only"],
)

db_tool = vajra.wrap(query_database)

analyst = Agent(
    role="Data Analyst",
    tools=[db_tool],
)

task = Task(
    description="Find top customers by MRR",
    agent=analyst,
)

crew = Crew(agents=[analyst], tasks=[task])
result = crew.kickoff()
```

## Environment Variables

- `VAJRA_ENDPOINT` — VAJRA daemon URL
- `VAJRA_AGENT_ID` — Agent identity
- `VAJRA_AGENT_SECRET` — Agent secret key
