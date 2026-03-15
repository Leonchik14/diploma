-- +goose Down
DROP INDEX IF EXISTS idx_users_profile_photo_material_id;

ALTER TABLE users
    DROP COLUMN IF EXISTS profile_photo_material_id;

