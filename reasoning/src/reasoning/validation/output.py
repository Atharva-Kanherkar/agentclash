"""Final-output validation against optional JSON schema."""

from __future__ import annotations

import json
from dataclasses import dataclass
from typing import Any


@dataclass
class ValidationResult:
    valid: bool
    error: str | None = None


def validate_final_output(output_text: str, output_schema: dict[str, Any] | None) -> ValidationResult:
    if not output_text.strip():
        return ValidationResult(valid=False, error="output is empty")

    if output_schema is None:
        return ValidationResult(valid=True)

    try:
        parsed = json.loads(output_text)
    except json.JSONDecodeError as exc:
        return ValidationResult(valid=False, error=f"output is not valid JSON: {exc}")

    # Minimal schema validation: check required fields if present in schema.
    required = output_schema.get("required", [])
    if isinstance(parsed, dict):
        missing = [f for f in required if f not in parsed]
        if missing:
            return ValidationResult(valid=False, error=f"missing required fields: {missing}")

    return ValidationResult(valid=True)
