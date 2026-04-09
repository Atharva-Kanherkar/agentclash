"""Tests for the OpenAI-compatible model client."""

import pytest
import httpx

from reasoning.client.model_client import ModelClient, ModelClientError


MOCK_COMPLETION = {
    "id": "chatcmpl-test",
    "object": "chat.completion",
    "choices": [
        {
            "index": 0,
            "message": {"role": "assistant", "content": "Paris"},
            "finish_reason": "stop",
        }
    ],
    "usage": {"prompt_tokens": 10, "completion_tokens": 3, "total_tokens": 13},
}

MOCK_TOOL_COMPLETION = {
    "id": "chatcmpl-test-tool",
    "object": "chat.completion",
    "choices": [
        {
            "index": 0,
            "message": {
                "role": "assistant",
                "content": None,
                "tool_calls": [
                    {
                        "id": "call_abc123",
                        "type": "function",
                        "function": {"name": "read_file", "arguments": '{"path": "/workspace/data.txt"}'},
                    }
                ],
            },
            "finish_reason": "tool_calls",
        }
    ],
    "usage": {"prompt_tokens": 15, "completion_tokens": 8, "total_tokens": 23},
}


@pytest.fixture
def mock_transport():
    """Creates an httpx mock transport for testing."""

    class MockTransport(httpx.AsyncBaseTransport):
        def __init__(self):
            self.requests: list[httpx.Request] = []
            self.responses: list[httpx.Response] = []
            self._response_queue: list[httpx.Response] = []

        def queue_response(self, status_code: int, json_data: dict | None = None, text: str = ""):
            import json

            if json_data is not None:
                content = json.dumps(json_data).encode()
                headers = {"content-type": "application/json"}
            else:
                content = text.encode()
                headers = {"content-type": "text/plain"}
            self._response_queue.append(httpx.Response(status_code, content=content, headers=headers))

        async def handle_async_request(self, request: httpx.Request) -> httpx.Response:
            self.requests.append(request)
            if not self._response_queue:
                return httpx.Response(500, content=b"no response queued")
            resp = self._response_queue.pop(0)
            self.responses.append(resp)
            return resp

    return MockTransport()


@pytest.fixture
def client_with_transport(mock_transport):
    """Creates a ModelClient that uses the mock transport."""
    client = ModelClient(api_key="test-key", base_url="https://api.test.com/v1")
    client._client = httpx.AsyncClient(transport=mock_transport, base_url="https://api.test.com/v1")
    return client, mock_transport


@pytest.mark.asyncio
async def test_successful_completion(client_with_transport):
    client, transport = client_with_transport
    transport.queue_response(200, MOCK_COMPLETION)

    response = await client.chat_completions(
        model="gpt-4o",
        messages=[{"role": "user", "content": "What is the capital of France?"}],
    )

    assert response.finish_reason == "stop"
    assert response.output_text == "Paris"
    assert response.usage.input_tokens == 10
    assert response.usage.output_tokens == 3
    assert len(response.tool_calls) == 0


@pytest.mark.asyncio
async def test_tool_call_completion(client_with_transport):
    client, transport = client_with_transport
    transport.queue_response(200, MOCK_TOOL_COMPLETION)

    response = await client.chat_completions(
        model="gpt-4o",
        messages=[{"role": "user", "content": "Read the data file"}],
        tools=[{"name": "read_file", "description": "Read a file", "parameters": {"type": "object"}}],
    )

    assert response.finish_reason == "tool_calls"
    assert len(response.tool_calls) == 1
    assert response.tool_calls[0].name == "read_file"
    assert response.tool_calls[0].id == "call_abc123"


@pytest.mark.asyncio
async def test_retries_on_429(client_with_transport):
    client, transport = client_with_transport
    transport.queue_response(429, text="rate limited")
    transport.queue_response(429, text="rate limited")
    transport.queue_response(200, MOCK_COMPLETION)

    response = await client.chat_completions(
        model="gpt-4o",
        messages=[{"role": "user", "content": "test"}],
    )

    assert response.output_text == "Paris"
    assert len(transport.requests) == 3


@pytest.mark.asyncio
async def test_no_retry_on_401(client_with_transport):
    client, transport = client_with_transport
    transport.queue_response(401, text="unauthorized")

    with pytest.raises(ModelClientError) as exc_info:
        await client.chat_completions(model="gpt-4o", messages=[{"role": "user", "content": "test"}])

    assert exc_info.value.status_code == 401
    assert not exc_info.value.retryable
    assert len(transport.requests) == 1


@pytest.mark.asyncio
async def test_exhausted_retries(client_with_transport):
    client, transport = client_with_transport
    for _ in range(3):
        transport.queue_response(500, text="server error")

    with pytest.raises(ModelClientError) as exc_info:
        await client.chat_completions(model="gpt-4o", messages=[{"role": "user", "content": "test"}])

    assert exc_info.value.retryable
    assert len(transport.requests) == 3
