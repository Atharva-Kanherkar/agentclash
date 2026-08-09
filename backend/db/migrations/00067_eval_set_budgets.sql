-- +goose Up
ALTER TABLE eval_sets
    DROP CONSTRAINT IF EXISTS eval_sets_status_check;

ALTER TABLE eval_sets
    ADD CONSTRAINT eval_sets_status_check
    CHECK (status IN (
        'queued',
        'expanding',
        'running',
        'aggregating',
        'completed',
        'failed',
        'cancelled',
        'budget_exceeded'
    ));

ALTER TABLE eval_sets
    ADD COLUMN IF NOT EXISTS spent_usd numeric NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS estimated_cost_usd numeric;

-- +goose Down
ALTER TABLE eval_sets
    DROP COLUMN IF EXISTS estimated_cost_usd;

ALTER TABLE eval_sets
    DROP COLUMN IF EXISTS spent_usd;

ALTER TABLE eval_sets
    DROP CONSTRAINT IF EXISTS eval_sets_status_check;

ALTER TABLE eval_sets
    ADD CONSTRAINT eval_sets_status_check
    CHECK (status IN (
        'queued',
        'expanding',
        'running',
        'aggregating',
        'completed',
        'failed',
        'cancelled'
    ));
