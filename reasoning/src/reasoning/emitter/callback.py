"""HTTP callback emitter that sends canonical events to Go."""

from __future__ import annotations

import logging
from typing import Any

import httpx

from reasoning.models.events import Envelope

logger = logging.getLogger(__name__)

TERMINAL_EVENT_TYPES = {"system.run.completed", "system.run.failed"}
MAX_TERMINAL_RETRIES = 3


class CallbackDeliveryError(Exception):
    pass


class CallbackEmitter:
    def __init__(self, callback_url: str, callback_token: str):
        self._url = callback_url
        self._token = callback_token
        self._client = httpx.AsyncClient(
            timeout=httpx.Timeout(30.0, connect=5.0),
            headers={"Content-Type": "application/json"},
        )

    async def close(self) -> None:
        await self._client.aclose()

    async def emit(self, event: Envelope) -> None:
        is_terminal = event.event_type in TERMINAL_EVENT_TYPES
        max_attempts = MAX_TERMINAL_RETRIES if is_terminal else 1

        last_error: Exception | None = None
        for attempt in range(max_attempts):
            try:
                resp = await self._client.post(
                    self._url,
                    content=event.model_dump_json(),
                    headers={"Authorization": f"Bearer {self._token}"},
                )
                if 200 <= resp.status_code < 300:
                    return
                last_error = CallbackDeliveryError(
                    f"callback returned {resp.status_code}: {resp.text}"
                )
            except httpx.TransportError as exc:
                last_error = CallbackDeliveryError(f"callback transport error: {exc}")

            if is_terminal and attempt < max_attempts - 1:
                logger.warning(
                    "terminal event delivery failed (attempt %d/%d): %s",
                    attempt + 1, max_attempts, last_error,
                )
                continue

        if is_terminal:
            logger.error("terminal event delivery exhausted all retries: %s", last_error)
            raise last_error  # type: ignore[misc]

        # Non-terminal delivery failure: fail-stop the run.
        raise last_error  # type: ignore[misc]
