"""Tests for canonical event envelope."""

from uuid import uuid4

from reasoning.models.events import SCHEMA_VERSION, SOURCE, Envelope, SummaryMetadata, make_event_id


def test_envelope_defaults():
    e = Envelope(
        event_id="test-1",
        run_id=uuid4(),
        run_agent_id=uuid4(),
        event_type="system.run.started",
    )
    assert e.schema_version == SCHEMA_VERSION
    assert e.source == SOURCE
    assert e.sequence_number == 0
    assert e.payload == {}


def test_envelope_round_trips():
    run_id = uuid4()
    agent_id = uuid4()
    e = Envelope(
        event_id="test-2",
        run_id=run_id,
        run_agent_id=agent_id,
        event_type="model.call.completed",
        payload={"finish_reason": "stop", "output_text": "Paris"},
        summary=SummaryMetadata(provider_key="openai", provider_model_id="gpt-4o"),
    )
    data = e.model_dump_json()
    restored = Envelope.model_validate_json(data)
    assert restored.run_id == run_id
    assert restored.payload["finish_reason"] == "stop"
    assert restored.summary.provider_key == "openai"


def test_make_event_id_format():
    agent_id = uuid4()
    eid = make_event_id(agent_id, "system.run.started", 1)
    assert eid.startswith("reasoning:")
    assert str(agent_id) in eid
    assert eid.endswith(":1")
