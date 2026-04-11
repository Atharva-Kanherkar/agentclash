"""Canonical event envelope matching Go runevents.Envelope."""

from __future__ import annotations

from datetime import datetime, timezone
from typing import Any
from uuid import UUID

from pydantic import BaseModel, Field

SCHEMA_VERSION = "2026-03-15"
SOURCE = "reasoning_engine"


class SummaryMetadata(BaseModel):
    status: str = ""
    step_index: int = 0
    provider_key: str = ""
    provider_model_id: str = ""
    tool_name: str = ""
    sandbox_action: str = ""
    metric_key: str = ""
    external_run_id: str = ""
    evidence_level: str = "native_structured"
    idempotency_key: str = ""


class Envelope(BaseModel):
    event_id: str
    schema_version: str = SCHEMA_VERSION
    run_id: UUID
    run_agent_id: UUID
    sequence_number: int = 0
    event_type: str
    source: str = SOURCE
    occurred_at: datetime = Field(default_factory=lambda: datetime.now(timezone.utc))
    payload: dict[str, Any] = Field(default_factory=dict)
    summary: SummaryMetadata = Field(default_factory=SummaryMetadata)


def make_event_id(run_agent_id: UUID, event_type: str, sequence: int) -> str:
    return f"reasoning:{run_agent_id}:{event_type}:{sequence}"
