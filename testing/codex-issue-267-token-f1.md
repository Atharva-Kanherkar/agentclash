# Issue 267: token_f1 validator contract

## Functional expectations

1. Add a new deterministic validator type `token_f1`.
2. `token_f1` must compute token-overlap precision, recall, and F1 using whitespace tokenization (`strings.Fields`).
3. The validator must support config-driven normalization by reusing the existing normalization pipeline before tokenization.
4. The validator config must accept:
   - `threshold` in `[0,1]`
   - `normalize` as a shorthand for the default normalization pipeline
   - `remove_articles`
   - `remove_punctuation`
5. When shorthand booleans are enabled, normalization must be applied in a predictable order before tokenization:
   - default normalize pipeline when `normalize=true`
   - punctuation stripping when `remove_punctuation=true`
   - article removal when `remove_articles=true`
   - whitespace collapse before tokenization
6. Exact token overlap must yield score `1.0` and verdict `pass`.
7. Partial overlap must yield a score strictly between `0` and `1`, with pass/fail determined by `threshold`.
8. No token overlap must yield score `0.0` and verdict `fail` unless threshold is `0`.
9. Empty prediction with non-empty reference must yield score `0.0`.
10. Empty reference with non-empty prediction must yield score `0.0`.
11. Empty prediction and empty reference must yield score `1.0`.
12. Validator evidence must include the computed score components needed to inspect the result.
13. `token_f1` must be accepted by spec loading, validation, and runtime validator dispatch.

## Tests to add or update

- Unit tests in `backend/internal/scoring/string_validators_test.go`
  - exact match
  - partial overlap
  - no overlap
  - normalization via `normalize`, `remove_articles`, and `remove_punctuation`
  - empty prediction/reference edge cases
  - invalid config / invalid threshold
- Spec-loading and config-validation coverage in:
  - `backend/internal/scoring/loader_test.go`
  - `backend/internal/scoring/validation.go` via loader-facing tests
- Integration-style engine coverage in `backend/internal/scoring/engine_test.go`
  - pass above threshold
  - fail below threshold
  - normalized comparison behavior

## Manual verification

1. Run targeted scoring tests for the backend package.
2. Confirm the new validator type is accepted by spec loading and produces validator results with normalized scores.
3. Confirm no unrelated scoring validator tests regress.
