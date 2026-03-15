-- +goose Up
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS profile_photo_material_id VARCHAR(100);

CREATE INDEX IF NOT EXISTS idx_users_profile_photo_material_id
    ON users(profile_photo_material_id);

