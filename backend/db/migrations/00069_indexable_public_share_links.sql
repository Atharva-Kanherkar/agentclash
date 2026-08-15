-- +goose Up
CREATE INDEX public_share_links_indexable_cursor_idx
ON public_share_links (id)
WHERE is_active AND search_indexing AND revoked_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS public_share_links_indexable_cursor_idx;
