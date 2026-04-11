"""Tests for bridge contract types matching Go reasoning.contracts."""

from datetime import datetime, timezone
from uuid import uuid4

from reasoning.models.bridge import (
    CancelRequest,
    StartRequest,
    StartResponse,
    ToolDefinition,
    ToolResult,
    ToolResultsBatch,
)


def test_start_request_round_trips():
    req = StartRequest(
        run_id=uuid4(),
        run_agent_id=uuid4(),
        idempotency_key="idem-1",
        execution_context={"Deployment": {}},
        tools=[ToolDefinition(name="read_file", description="Read a file", parameters={"type": "object"})],
        callback_url="http://localhost:8080/events",
        callback_token="token-abc",
        deadline_at=datetime.now(timezone.utc),
    )
    data = req.model_dump_json()
    restored = StartRequest.model_validate_json(data)
    assert restored.run_id == req.run_id
    assert restored.tools[0].name == "read_file"
    assert restored.callback_token == "token-abc"


def test_start_response_round_trips():
    resp = StartResponse(accepted=True, reasoning_run_id="rr-abc123")
    data = resp.model_dump_json()
    restored = StartResponse.model_validate_json(data)
    assert restored.accepted is True
    assert restored.reasoning_run_id == "rr-abc123"


def test_tool_result_all_statuses():
    for status in ["completed", "blocked", "skipped", "failed"]:
        result = ToolResult(tool_call_id="tc-1", status=status, content="ok")
        assert result.status == status


def test_tool_results_batch_round_trips():
    batch = ToolResultsBatch(
        idempotency_key="key-1",
        tool_results=[
            ToolResult(tool_call_id="tc-1", status="completed", content="file contents"),
            ToolResult(tool_call_id="tc-2", status="failed", error_message="not found"),
        ],
    )
    data = batch.model_dump_json()
    restored = ToolResultsBatch.model_validate_json(data)
    assert len(restored.tool_results) == 2
    assert restored.tool_results[0].status == "completed"
    assert restored.tool_results[1].error_message == "not found"


def test_cancel_request_round_trips():
    req = CancelRequest(idempotency_key="cancel-1", reason="timeout")
    data = req.model_dump_json()
    restored = CancelRequest.model_validate_json(data)
    assert restored.reason == "timeout"
