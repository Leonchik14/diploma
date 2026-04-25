-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS file_interactions (
    id BIGSERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    node_id INTEGER NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    interaction_type VARCHAR(32) NOT NULL,
    interacted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_file_interactions_user_time
    ON file_interactions(user_id, interacted_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_file_interactions_user_node_time
    ON file_interactions(user_id, node_id, interacted_at DESC, id DESC);
-- +goose StatementEnd
