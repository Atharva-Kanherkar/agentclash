"""Tests for the ReAct reasoning engine."""

import asyncio
from datetime import datetime, timezone
from unittest.mock import AsyncMock
from uuid import uuid4

import pytest

from reasoning.client.model_client import ModelClient, ModelClientError, ModelResponse, ToolCall, Usage
from reasoning.emitter.callback import CallbackEmitter
from reasoning.engine.react import ReactEngine
from reasoning.models.bridge import StartRequest, ToolDefinition, ToolResult, ToolResultsBatch


def make_request(**overrides) -> StartRequest:
    defaults = {
        "run_id": uuid4(),
        "run_agent_id": uuid4(),
        "idempotency_key": "test-key",
        "execution_context": {
            "Deployment": {
                "RuntimeProfile": {"MaxIterations": 10, "StepTimeoutSeconds": 60, "RunTimeoutSeconds": 300},
                "ProviderAccount": {"ProviderKey": "openai", "_resolved_credential": "sk-test"},
                "ModelAlias": {"ModelCatalogEntry": {"ProviderModelID": "gpt-4o"}},
                "AgentBuildVersion": {"PolicySpec": None, "OutputSchema": None, "AgentKind": "reasoning_v1"},
            },
            "ChallengeInputSet": {"InputKey": "test-input", "Items": [{"Content": "What is 2+2?"}]},
        },
        "tools": [],
        "callback_url": "http://localhost:8080/events",
        "callback_token": "test-token",
        "deadline_at": datetime.now(timezone.utc),
    }
    defaults.update(overrides)
    return StartRequest(**defaults)


class FakeModelClient:
    """Mock model client that returns queued responses."""

    def __init__(self):
        self._responses: list[ModelResponse | ModelClientError] = []

    def queue(self, response: ModelResponse | ModelClientError):
        self._responses.append(response)

    async def chat_completions(self, **kwargs) -> ModelResponse:
        if not self._responses:
            raise ModelClientError("no responses queued")
        resp = self._responses.pop(0)
        if isinstance(resp, ModelClientError):
            raise resp
        return resp

    async def close(self):
        pass


class FakeEmitter:
    """Mock callback emitter that records events."""

    def __init__(self):
        self.events: list[dict] = []

    async def emit(self, event):
        self.events.append({"event_type": event.event_type, "payload": event.payload})

    async def close(self):
        pass


@pytest.mark.asyncio
async def test_tool_free_success():
    """Model returns a direct answer with no tool calls."""
    model = FakeModelClient()
    model.queue(ModelResponse(
        finish_reason="stop",
        output_text="The answer is 4.",
        tool_calls=[],
        usage=Usage(input_tokens=10, output_tokens=5, total_tokens=15),
    ))

    emitter = FakeEmitter()
    engine = ReactEngine(request=make_request(), model_client=model, emitter=emitter)
    await engine.run()

    event_types = [e["event_type"] for e in emitter.events]
    assert event_types == [
        "system.run.started",
        "system.step.started",
        "model.call.started",
        "model.call.completed",
        "system.step.completed",
        "system.output.finalized",
        "system.run.completed",
    ]

    completed = emitter.events[-1]
    assert completed["payload"]["final_output"] == "The answer is 4."
    assert completed["payload"]["step_count"] == 1
    assert completed["payload"]["input_tokens"] == 10


@pytest.mark.asyncio
async def test_tool_using_turn():
    """Model proposes a tool call, gets results, then answers."""
    model = FakeModelClient()
    # First call: propose tool
    model.queue(ModelResponse(
        finish_reason="tool_calls",
        output_text="",
        tool_calls=[ToolCall(id="tc-1", name="read_file", arguments='{"path": "/data.txt"}')],
        usage=Usage(input_tokens=20, output_tokens=10, total_tokens=30),
    ))
    # Second call: final answer after tool result
    model.queue(ModelResponse(
        finish_reason="stop",
        output_text="The file says hello.",
        tool_calls=[],
        usage=Usage(input_tokens=30, output_tokens=8, total_tokens=38),
    ))

    emitter = FakeEmitter()
    request = make_request(tools=[ToolDefinition(name="read_file", description="Read a file")])
    engine = ReactEngine(request=request, model_client=model, emitter=emitter)

    # Run engine in background, deliver tool results when proposal arrives.
    async def deliver_results():
        while True:
            events = [e["event_type"] for e in emitter.events]
            if "model.tool_calls.proposed" in events:
                engine.deliver_tool_results(ToolResultsBatch(
                    idempotency_key="tools-1",
                    tool_results=[ToolResult(tool_call_id="tc-1", status="completed", content="hello world")],
                ))
                return
            await asyncio.sleep(0.01)

    await asyncio.gather(engine.run(), deliver_results())

    event_types = [e["event_type"] for e in emitter.events]
    assert "model.tool_calls.proposed" in event_types
    assert "tool.call.completed" in event_types
    assert "system.run.completed" in event_types
    assert emitter.events[-1]["payload"]["step_count"] == 2
    assert emitter.events[-1]["payload"]["tool_call_count"] == 1


@pytest.mark.asyncio
async def test_max_iterations():
    """Engine stops after max_iterations."""
    model = FakeModelClient()
    for _ in range(3):
        model.queue(ModelResponse(
            finish_reason="tool_calls",
            output_text="",
            tool_calls=[ToolCall(id="tc-1", name="read_file", arguments='{}')],
            usage=Usage(input_tokens=5, output_tokens=5, total_tokens=10),
        ))

    emitter = FakeEmitter()
    request = make_request(
        execution_context={
            "Deployment": {
                "RuntimeProfile": {"MaxIterations": 2, "StepTimeoutSeconds": 60, "RunTimeoutSeconds": 300},
                "ProviderAccount": {"ProviderKey": "openai", "_resolved_credential": "sk-test"},
                "ModelAlias": {"ModelCatalogEntry": {"ProviderModelID": "gpt-4o"}},
                "AgentBuildVersion": {"PolicySpec": None, "OutputSchema": None},
            },
        },
        tools=[ToolDefinition(name="read_file", description="Read")],
    )
    engine = ReactEngine(request=request, model_client=model, emitter=emitter)

    async def deliver_results():
        delivered = 0
        while delivered < 2:
            events = [e["event_type"] for e in emitter.events]
            count = events.count("model.tool_calls.proposed")
            while delivered < count:
                engine.deliver_tool_results(ToolResultsBatch(
                    idempotency_key=f"tools-{delivered}",
                    tool_results=[ToolResult(tool_call_id="tc-1", status="completed", content="ok")],
                ))
                delivered += 1
            await asyncio.sleep(0.01)

    await asyncio.gather(engine.run(), deliver_results())

    event_types = [e["event_type"] for e in emitter.events]
    assert "system.run.failed" in event_types
    failed = next(e for e in emitter.events if e["event_type"] == "system.run.failed")
    assert failed["payload"]["stop_reason"] == "max_iterations"


@pytest.mark.asyncio
async def test_finish_reason_length():
    """Model returns finish_reason=length -> failure."""
    model = FakeModelClient()
    model.queue(ModelResponse(
        finish_reason="length",
        output_text="truncated...",
        tool_calls=[],
        usage=Usage(input_tokens=10, output_tokens=100, total_tokens=110),
    ))

    emitter = FakeEmitter()
    engine = ReactEngine(request=make_request(), model_client=model, emitter=emitter)
    await engine.run()

    event_types = [e["event_type"] for e in emitter.events]
    assert "system.run.failed" in event_types
    failed = next(e for e in emitter.events if e["event_type"] == "system.run.failed")
    assert failed["payload"]["stop_reason"] == "max_tokens"


@pytest.mark.asyncio
async def test_provider_error():
    """Model client raises error -> failure."""
    model = FakeModelClient()
    model.queue(ModelClientError("connection refused", retryable=False))

    emitter = FakeEmitter()
    engine = ReactEngine(request=make_request(), model_client=model, emitter=emitter)
    await engine.run()

    event_types = [e["event_type"] for e in emitter.events]
    assert "system.run.failed" in event_types
    failed = next(e for e in emitter.events if e["event_type"] == "system.run.failed")
    assert failed["payload"]["stop_reason"] == "provider_error"


@pytest.mark.asyncio
async def test_cancellation():
    """Cancelled run emits failure."""
    model = FakeModelClient()
    # Model would respond, but cancel fires first.
    model.queue(ModelResponse(
        finish_reason="stop", output_text="answer", tool_calls=[], usage=Usage()
    ))

    emitter = FakeEmitter()
    engine = ReactEngine(request=make_request(), model_client=model, emitter=emitter)
    engine.cancel()
    await engine.run()

    event_types = [e["event_type"] for e in emitter.events]
    assert "system.run.failed" in event_types
    failed = next(e for e in emitter.events if e["event_type"] == "system.run.failed")
    assert failed["payload"]["stop_reason"] == "cancelled"
