# codex/issue-266-math-equivalence-validator — Test Contract

## Functional Behavior
- Add a deterministic scoring validator type named `math_equivalence`.
- The validator compares math answers using pure Go normalization and parsing logic.
- Equivalent numeric forms pass, including common fraction and decimal representations such as `1/2`, `0.5`, and `2/4`.
- Common answer wrappers and formatting noise are ignored when configured, including `$...$`, `\boxed{...}`, extra whitespace, and GSM8K-style answer extraction after a delimiter.
- Common LaTeX forms pass when they represent the same numeric value, including `\frac{1}{2}` and `\sqrt{2}`.
- Numeric fallback uses tolerance-based comparison when symbolic-style normalization does not yield a direct match.
- Non-equivalent answers fail with diagnostic evidence that explains what was parsed and compared.
- Invalid configs or unparsable expressions return validator errors rather than silently passing.

## Unit Tests
- `TestValidateMathEquivalence` covers fraction, decimal, LaTeX, boxed-answer, extraction, percentage, scientific-notation, and failure cases.
- `TestValidateMathEquivalence_InvalidConfig` rejects malformed or conflicting validator config.
- `TestParseMathEquivalenceConfig` covers defaults and config aliases if any are introduced.
- `TestNormalizeMathExpression` covers stripping wrappers and translating supported LaTeX into parseable forms.
- Existing loader validation tests cover `math_equivalence` as a supported validator type and validate its config constraints.

## Integration / Functional Tests
- `TestEvaluateRunAgent_MathEquivalencePassesForFractions` verifies end-to-end scoring for equivalent numeric expressions.
- `TestEvaluateRunAgent_MathEquivalenceExtractsDelimitedAnswer` verifies extraction after `####`.
- `TestEvaluateRunAgent_MathEquivalenceHandlesLatexWrapper` verifies run-agent evaluation for boxed/LaTeX output.
- `TestEvaluateRunAgent_MathEquivalenceFailsForDifferentAnswers` verifies correctness scoring drops to `0` on mismatch.

## Smoke Tests
- `cd backend && go test ./internal/scoring/...`
- `cd backend && go test ./cmd/validate-pack/...`

## E2E Tests
- N/A — this change is confined to backend scoring validation and spec loading.

## Manual / cURL Tests
```bash
cd /Users/atharva/agentclash/backend
go test ./internal/scoring/... -run 'MathEquivalence|LoadEvaluationSpec'
```

```bash
cd /Users/atharva/agentclash/backend
go test ./cmd/validate-pack/... ./internal/scoring/...
```
