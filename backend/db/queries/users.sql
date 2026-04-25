-- name: BackfillUserEmail :one
UPDATE users
SET
    email = CASE
        WHEN COALESCE(email, '') = '' AND @email::text <> '' THEN @email::text
        ELSE email
    END,
    updated_at = CASE
        WHEN COALESCE(email, '') = '' AND @email::text <> '' THEN now()
        ELSE updated_at
    END
WHERE id = @user_id AND archived_at IS NULL
RETURNING id, workos_user_id, COALESCE(email, '') AS email, COALESCE(display_name, '') AS display_name;
