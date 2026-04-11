"""Tests for final-output validation."""

from reasoning.validation.output import validate_final_output


def test_valid_output_no_schema():
    result = validate_final_output("The answer is 42.", None)
    assert result.valid is True


def test_empty_output_fails():
    result = validate_final_output("", None)
    assert result.valid is False
    assert "empty" in result.error


def test_whitespace_only_fails():
    result = validate_final_output("   ", None)
    assert result.valid is False


def test_valid_json_matching_schema():
    schema = {"type": "object", "required": ["answer"]}
    result = validate_final_output('{"answer": "42"}', schema)
    assert result.valid is True


def test_json_missing_required_field():
    schema = {"type": "object", "required": ["answer", "confidence"]}
    result = validate_final_output('{"answer": "42"}', schema)
    assert result.valid is False
    assert "confidence" in result.error


def test_invalid_json_with_schema():
    schema = {"type": "object", "required": ["answer"]}
    result = validate_final_output("not json at all", schema)
    assert result.valid is False
    assert "not valid JSON" in result.error


def test_valid_non_object_json():
    schema = {"type": "string"}
    result = validate_final_output('"just a string"', schema)
    assert result.valid is True
