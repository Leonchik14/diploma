-- +goose Up
ALTER TABLE nodes ADD COLUMN IF NOT EXISTS material_id UUID;
UPDATE nodes SET material_id = gen_random_uuid() WHERE material_id IS NULL;
ALTER TABLE nodes ALTER COLUMN material_id SET NOT NULL;
ALTER TABLE nodes ALTER COLUMN material_id SET DEFAULT gen_random_uuid();
CREATE UNIQUE INDEX IF NOT EXISTS idx_nodes_material_id ON nodes(material_id);
