-- +goose Up
CREATE UNIQUE INDEX workspace_regression_cases_suite_run_agent_challenge_idx
    ON workspace_regression_cases (suite_id, source_run_agent_id, source_challenge_identity_id)
    WHERE source_run_agent_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS workspace_regression_cases_suite_run_agent_challenge_idx;
