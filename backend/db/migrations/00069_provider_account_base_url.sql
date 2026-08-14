-- +goose Up
ALTER TABLE provider_accounts
    ADD COLUMN base_url text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE provider_accounts
    DROP COLUMN IF EXISTS base_url;
