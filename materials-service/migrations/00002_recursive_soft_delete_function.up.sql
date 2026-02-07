-- +goose Up
-- +goose StatementBegin
-- Create function for recursive soft delete
CREATE OR REPLACE FUNCTION recursive_soft_delete(node_id_to_delete INTEGER, user_id_check INTEGER)
RETURNS void AS $$
BEGIN
    WITH RECURSIVE descendants AS (
        -- Start with the node to delete
        SELECT id, parent_id
        FROM nodes
        WHERE id = node_id_to_delete AND user_id = user_id_check
        
        UNION ALL
        
        -- Find all children
        SELECT n.id, n.parent_id
        FROM nodes n
        INNER JOIN descendants d ON n.parent_id = d.id
        WHERE n.user_id = user_id_check AND n.deleted_at IS NULL
    )
    UPDATE nodes
    SET deleted_at = CURRENT_TIMESTAMP
    WHERE id IN (SELECT id FROM descendants);
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd