"""
VAJRA LangChain Integration
============================
Ephemeral credential injection for LangChain agents.

Installation:
    pip install vajra-langchain

Usage:
    from vajra_langchain import VajraToolInterceptor

    vajra = VajraToolInterceptor(
        endpoint="http://localhost:9735",
        agent_id="my-agent",
        agent_secret="sk_...",
    )
    tools = [vajra.intercept(t) for t in my_tools]
"""

import json
import logging
import os
import time
import uuid
from typing import Any, Callable, Dict, Optional
from urllib import request, error

logger = logging.getLogger("vajra")


class VajraCredential:
    """Represents an ephemeral credential issued by VAJRA."""

    def __init__(self, data: Dict[str, Any]):
        self.id = data.get("credential_id", "")
        self.target = data.get("target", "")
        self.scope = data.get("scope", "")
        self.ttl = data.get("ttl", "5s")
        self.expires_at = data.get("expires_at", "")
        self._created_at = time.time()

    @property
    def is_expired(self) -> bool:
        return time.time() - self._created_at > 5.0

    @property
    def remaining_ttl(self) -> float:
        elapsed = time.time() - self._created_at
        return max(0.0, 5.0 - elapsed)


class VajraToolInterceptor:
    """
    Wraps LangChain tools with VAJRA ephemeral credential injection.

    Each tool call is intercepted, a scoped credential is minted,
    and the credential is injected into the tool's execution context.
    """

    def __init__(
        self,
        endpoint: str = "http://localhost:9735",
        agent_id: Optional[str] = None,
        agent_secret: Optional[str] = None,
        default_ttl: str = "5s",
        policy_tags: Optional[list] = None,
        policy_overrides: Optional[Dict] = None,
    ):
        self.endpoint = endpoint.rstrip("/")
        self.agent_id = agent_id or os.getenv("VAJRA_AGENT_ID", "default-agent")
        self.agent_secret = agent_secret or os.getenv("VAJRA_AGENT_SECRET", "")
        self.default_ttl = default_ttl
        self.policy_tags = policy_tags or os.getenv("VAJRA_POLICY_TAGS", "").split(",")
        self.policy_overrides = policy_overrides or {}

    def intercept(self, tool: Any) -> Any:
        """
        Wrap a LangChain tool with VAJRA credential injection.

        Args:
            tool: A LangChain BaseTool instance.

        Returns:
            The same tool with its _run method wrapped.
        """
        original_run = tool._run

        def wrapped_run(*args, **kwargs):
            credential = self._mint_credential(tool.name, args, kwargs)
            if credential:
                logger.info(
                    f"VAJRA: Minted credential {credential.id} for {tool.name}"
                )
                # Inject credential into kwargs
                kwargs["_vajra_credential"] = credential
                kwargs["_vajra_target"] = credential.target
                kwargs["_vajra_scope"] = credential.scope

            try:
                return original_run(*args, **kwargs)
            finally:
                if credential:
                    logger.debug(f"VAJRA: Credential {credential.id} consumed")

        # Preserve metadata
        wrapped_run.__name__ = original_run.__name__
        wrapped_run.__doc__ = original_run.__doc__
        tool._run = wrapped_run

        # Also wrap async if present
        if hasattr(tool, "_arun"):
            original_arun = tool._arun

            async def wrapped_arun(*args, **kwargs):
                credential = self._mint_credential(tool.name, args, kwargs)
                if credential:
                    kwargs["_vajra_credential"] = credential
                try:
                    return await original_arun(*args, **kwargs)
                finally:
                    pass

            tool._arun = wrapped_arun

        return tool

    def _mint_credential(
        self, tool_name: str, args: tuple, kwargs: dict
    ) -> Optional[VajraCredential]:
        """Request a credential from the VAJRA daemon."""
        try:
            payload = json.dumps({
                "tool": tool_name,
                "agent_id": self.agent_id,
                "args": str(args),
                "kwargs": str(kwargs),
                "ttl": self.default_ttl,
                "policy_tags": self.policy_tags,
            }).encode()

            req = request.Request(
                f"{self.endpoint}/v1/mint",
                data=payload,
                headers={
                    "Content-Type": "application/json",
                    "Authorization": f"Bearer {self.agent_secret}",
                },
                method="POST",
            )

            with request.urlopen(req, timeout=2) as resp:
                data = json.loads(resp.read().decode())
                return VajraCredential(data)

        except error.URLError as e:
            logger.warning(f"VAJRA: Could not mint credential: {e}")
            return None
        except Exception as e:
            logger.error(f"VAJRA: Unexpected error minting credential: {e}")
            return None


__all__ = ["VajraToolInterceptor", "VajraCredential"]
