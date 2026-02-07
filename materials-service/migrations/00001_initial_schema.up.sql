-- +goose Up
-- +goose StatementBegin
-- Create nodes table
CREATE TABLE IF NOT EXISTS nodes (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    parent_id INTEGER REFERENCES nodes(id) ON DELETE SET NULL,
    type VARCHAR(10) NOT NULL CHECK (type IN ('folder', 'file', 'link')),
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP
);

-- Create indexes for nodes
CREATE INDEX IF NOT EXISTS idx_nodes_user_parent ON nodes(user_id, parent_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_nodes_user_id ON nodes(user_id, id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_nodes_deleted_at ON nodes(deleted_at);

-- Create unique constraint for name within parent for active nodes
CREATE UNIQUE INDEX IF NOT EXISTS idx_nodes_unique_name_in_parent 
ON nodes(user_id, parent_id, name) 
WHERE deleted_at IS NULL;

-- Create files table
CREATE TABLE IF NOT EXISTS files (
    node_id INTEGER PRIMARY KEY REFERENCES nodes(id) ON DELETE CASCADE,
    object_key VARCHAR(512) NOT NULL,
    mime_type VARCHAR(255) NOT NULL,
    size BIGINT NOT NULL,
    checksum VARCHAR(64) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- Create indexes for files
CREATE INDEX IF NOT EXISTS idx_files_object_key ON files(object_key);

-- Create links table
CREATE TABLE IF NOT EXISTS links (
    node_id INTEGER PRIMARY KEY REFERENCES nodes(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    title VARCHAR(255),
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- Create function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create trigger for updated_at
CREATE TRIGGER update_nodes_updated_at 
    BEFORE UPDATE ON nodes 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();
-- +goose StatementEnd