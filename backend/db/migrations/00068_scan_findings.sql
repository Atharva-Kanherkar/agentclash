-- +goose Up
CREATE TABLE scan_findings (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id uuid NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
    organization_id uuid NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    eval_set_id uuid NOT NULL REFERENCES eval_sets (id) ON DELETE CASCADE,
    case_result_id uuid REFERENCES case_results (id) ON DELETE SET NULL,
    matrix_key text NOT NULL DEFAULT '',
    case_key text NOT NULL DEFAULT '',
    scanner text NOT NULL,
    scanner_version text NOT NULL,
    severity text NOT NULL,
    category text NOT NULL DEFAULT '',
    evidence text NOT NULL DEFAULT '',
    confidence double precision NOT NULL DEFAULT 0,
    status text NOT NULL DEFAULT 'open'
        CHECK (status IN ('open', 'triaged', 'dismissed')),
    status_updated_by uuid REFERENCES users (id) ON DELETE SET NULL,
    status_updated_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (eval_set_id, case_key, scanner, scanner_version)
);

CREATE INDEX scan_findings_eval_set_idx ON scan_findings (eval_set_id, severity, status);
CREATE INDEX scan_findings_workspace_idx ON scan_findings (workspace_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS scan_findings;
