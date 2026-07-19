"""
VAJRA CrewAI Integration
========================
Ephemeral credential injection for CrewAI agents.

Installation:
    pip install vajra-crewai

Usage:
    from vajra_crewai import VajraToolWrapper
"""

import json
import logging
import os
from typing import Any, Callable, Dict, Optional, List
from urllib import request, error

logger = logging.getLogger("vajra")


class VajraToolWrapper:
    """
    Wraps CrewAI tools with VAJRA ephemeral credential injection.
    """

    def __init__(
        self,
        endpoint: str = "http://localhost:9735",
        agent_id: Optional[str] = None,
        agent_secret: Optional[str] = None,
        default_ttl: str = "5s",
        policy_tags: Optional[List[str]] = None,
    ):
        self.endpoint = endpoint.rstrip("/")
        self.agent_id = agent_id or os.getenv("VAJRA_AGENT_ID", "default-agent")
        self.agent_secret = agent_secret or os.getenv("VAJRA_AGENT_SECRET", "")
        self.default_ttl = default_ttl
        self.policy_tags = policy_tags or []

    def wrap(self, func: Callable) -> Callable:
        """
        Wrap a CrewAI tool function with VAJRA credential injection.
        """
        import functools

        @functools.wraps(func)
        def wrapper(*args, **kwargs):
            credential = self._mint_credential(func.__name__, args, kwargs)
            if credential:
                logger.info(f"VAJRA: Minted credential for {func.__name__}")
                kwargs["_vajra_credential_id"] = credential.get("credential_id", "")

            try:
                return func(*args, **kwargs)
            finally:
                if credential:
                    logger.debug(f"VAJRA: Credential consumed for {func.__name__}")

        return wrapper

    def _mint_credential(
        self, tool_name: str, args: tuple, kwargs: dict
    ) -> Optional[Dict[str, Any]]:
        """Request a credential from the VAJRA daemon."""
        try:
            payload = json.dumps({
                "tool": tool_name,
                "agent_id": self.agent_id,
                "policy_tags": self.policy_tags,
                "ttl": self.default_ttl,
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
                return json.loads(resp.read().decode())

        except error.URLError:
            logger.warning(f"VAJRA: Could not reach daemon at {self.endpoint}")
            return None
        except Exception as e:
            logger.error(f"VAJRA: Error minting credential: {e}")
            return None


__all__ = ["VajraToolWrapper"]
