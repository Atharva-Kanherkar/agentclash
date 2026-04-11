"""Bridge types matching Go reasoning.contracts exactly."""

from __future__ import annotations

from datetime import datetime
from typing import Any, Literal
from uuid import UUID

from pydantic import BaseModel, Field


class ToolDefinition(BaseModel):
    name: str
    description: str = ""
    parameters: dict[str, Any] | None = None


class StartRequest(BaseModel):
    run_id: UUID
    run_agent_id: UUID
    idempotency_key: str
    execution_context: dict[str, Any]
    tools: list[ToolDefinition] = Field(default_factory=list)
    callback_url: str
    callback_token: str
    deadline_at: datetime


class StartResponse(BaseModel):
    accepted: bool
    reasoning_run_id: str
    error: str | None = None


ToolResultStatus = Literal["completed", "blocked", "skipped", "failed"]


class ToolResult(BaseModel):
    tool_call_id: str
    status: ToolResultStatus
    content: str = ""
    error_message: str | None = None


class ToolResultsBatch(BaseModel):
    idempotency_key: str
    tool_results: list[ToolResult]


class CancelRequest(BaseModel):
    idempotency_key: str
    reason: str
