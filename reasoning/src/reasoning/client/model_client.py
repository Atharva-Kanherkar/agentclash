"""OpenAI-compatible /chat/completions client with transient retry."""

from __future__ import annotations

import asyncio
import logging
from dataclasses import dataclass, field
from typing import Any

import httpx

logger = logging.getLogger(__name__)

TRANSIENT_STATUS_CODES = {429, 500, 502, 503, 504}
MAX_RETRIES = 3
INITIAL_BACKOFF = 1.0


@dataclass
class Usage:
    input_tokens: int = 0
    output_tokens: int = 0
    total_tokens: int = 0


@dataclass
class ToolCall:
    id: str
    name: str
    arguments: str  # raw JSON string


@dataclass
class ModelResponse:
    finish_reason: str
    output_text: str
    tool_calls: list[ToolCall] = field(default_factory=list)
    usage: Usage = field(default_factory=Usage)
    raw_response: dict[str, Any] = field(default_factory=dict)


class ModelClientError(Exception):
    def __init__(self, message: str, *, retryable: bool = False, status_code: int | None = None):
        super().__init__(message)
        self.retryable = retryable
        self.status_code = status_code


class ModelClient:
    def __init__(self, api_key: str, base_url: str = "https://api.openai.com/v1"):
        self._client = httpx.AsyncClient(
            base_url=base_url,
            headers={"Authorization": f"Bearer {api_key}", "Content-Type": "application/json"},
            timeout=httpx.Timeout(120.0, connect=10.0),
        )

    async def close(self) -> None:
        await self._client.aclose()

    async def chat_completions(
        self,
        model: str,
        messages: list[dict[str, Any]],
        tools: list[dict[str, Any]] | None = None,
        temperature: float = 0.0,
        max_tokens: int | None = None,
    ) -> ModelResponse:
        body: dict[str, Any] = {
            "model": model,
            "messages": messages,
            "temperature": temperature,
        }
        if tools:
            body["tools"] = [{"type": "function", "function": t} for t in tools]
        if max_tokens is not None:
            body["max_tokens"] = max_tokens

        last_error: Exception | None = None
        for attempt in range(MAX_RETRIES):
            try:
                resp = await self._client.post("/chat/completions", json=body)
            except httpx.TransportError as exc:
                last_error = ModelClientError(f"transport error: {exc}", retryable=True)
                await self._backoff(attempt)
                continue

            if resp.status_code in TRANSIENT_STATUS_CODES:
                last_error = ModelClientError(
                    f"provider returned {resp.status_code}", retryable=True, status_code=resp.status_code
                )
                logger.warning("transient provider error (attempt %d/%d): %s", attempt + 1, MAX_RETRIES, last_error)
                await self._backoff(attempt)
                continue

            if resp.status_code >= 400:
                raise ModelClientError(
                    f"provider returned {resp.status_code}: {resp.text}", retryable=False, status_code=resp.status_code
                )

            return self._parse_response(resp.json())

        raise last_error or ModelClientError("all retries exhausted")

    @staticmethod
    def _parse_response(data: dict[str, Any]) -> ModelResponse:
        choice = data.get("choices", [{}])[0]
        message = choice.get("message", {})
        finish_reason = choice.get("finish_reason", "")

        tool_calls: list[ToolCall] = []
        for tc in message.get("tool_calls", []):
            func = tc.get("function", {})
            tool_calls.append(ToolCall(id=tc["id"], name=func.get("name", ""), arguments=func.get("arguments", "{}")))

        usage_data = data.get("usage", {})
        usage = Usage(
            input_tokens=usage_data.get("prompt_tokens", 0),
            output_tokens=usage_data.get("completion_tokens", 0),
            total_tokens=usage_data.get("total_tokens", 0),
        )

        return ModelResponse(
            finish_reason=finish_reason,
            output_text=message.get("content", "") or "",
            tool_calls=tool_calls,
            usage=usage,
            raw_response=data,
        )

    @staticmethod
    async def _backoff(attempt: int) -> None:
        delay = INITIAL_BACKOFF * (2**attempt)
        await asyncio.sleep(delay)
