-- +goose Up

-- The old unique indexes include archived rows, which causes INSERT failures
-- when resolveOrCreateModelAlias tries to re-create an alias that was archived.
-- Fix: exclude archived rows from the uniqueness constraint.

DROP INDEX IF EXISTS model_aliases_workspace_alias_uq;
DROP INDEX IF EXISTS model_aliases_org_alias_uq;

CREATE UNIQUE INDEX model_aliases_org_alias_uq
ON model_aliases (organization_id, alias_key)
WHERE workspace_id IS NULL AND archived_at IS NULL;

CREATE UNIQUE INDEX model_aliases_workspace_alias_uq
ON model_aliases (workspace_id, alias_key)
WHERE workspace_id IS NOT NULL AND archived_at IS NULL;

-- Same fix for provider_accounts unique indexes.
DROP INDEX IF EXISTS provider_accounts_org_slug_uq;
DROP INDEX IF EXISTS provider_accounts_workspace_slug_uq;

CREATE UNIQUE INDEX provider_accounts_org_slug_uq
ON provider_accounts (organization_id, provider_key, name)
WHERE workspace_id IS NULL AND archived_at IS NULL;

CREATE UNIQUE INDEX provider_accounts_workspace_slug_uq
ON provider_accounts (workspace_id, provider_key, name)
WHERE workspace_id IS NOT NULL AND archived_at IS NULL;

-- +goose Down

DROP INDEX IF EXISTS model_aliases_workspace_alias_uq;
DROP INDEX IF EXISTS model_aliases_org_alias_uq;
DROP INDEX IF EXISTS provider_accounts_org_slug_uq;
DROP INDEX IF EXISTS provider_accounts_workspace_slug_uq;

CREATE UNIQUE INDEX model_aliases_org_alias_uq
ON model_aliases (organization_id, alias_key)
WHERE workspace_id IS NULL;

CREATE UNIQUE INDEX model_aliases_workspace_alias_uq
ON model_aliases (workspace_id, alias_key)
WHERE workspace_id IS NOT NULL;

CREATE UNIQUE INDEX provider_accounts_org_slug_uq
ON provider_accounts (organization_id, provider_key, name)
WHERE workspace_id IS NULL;

CREATE UNIQUE INDEX provider_accounts_workspace_slug_uq
ON provider_accounts (workspace_id, provider_key, name)
WHERE workspace_id IS NOT NULL;
