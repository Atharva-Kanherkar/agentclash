-- name: UpsertScanFinding :one
INSERT INTO scan_findings (
    workspace_id,
    organization_id,
    eval_set_id,
    case_result_id,
    matrix_key,
    case_key,
    scanner,
    scanner_version,
    severity,
    category,
    evidence,
    confidence,
    status
) VALUES (
    @workspace_id,
    @organization_id,
    @eval_set_id,
    sqlc.narg('case_result_id'),
    @matrix_key,
    @case_key,
    @scanner,
    @scanner_version,
    @severity,
    @category,
    @evidence,
    @confidence,
    @status
)
ON CONFLICT (eval_set_id, case_key, scanner, scanner_version) DO UPDATE SET
    severity = EXCLUDED.severity,
    category = EXCLUDED.category,
    evidence = EXCLUDED.evidence,
    confidence = EXCLUDED.confidence,
    case_result_id = COALESCE(EXCLUDED.case_result_id, scan_findings.case_result_id),
    updated_at = now()
RETURNING *;

-- name: ListScanFindingsByEvalSetID :many
SELECT * FROM scan_findings
WHERE eval_set_id = @eval_set_id
  AND (sqlc.narg('severity')::text IS NULL OR severity = sqlc.narg('severity'))
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status'))
ORDER BY created_at DESC
LIMIT @limit_count OFFSET @offset_count;

-- name: GetScanFindingByID :one
SELECT * FROM scan_findings WHERE id = @id;

-- name: UpdateScanFindingStatus :one
UPDATE scan_findings
SET status = @status,
    status_updated_by = sqlc.narg('status_updated_by'),
    status_updated_at = now(),
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: CountScanFindingsBySeverity :many
SELECT severity, count(*)::bigint AS n
FROM scan_findings
WHERE eval_set_id = @eval_set_id
GROUP BY severity
ORDER BY severity;
