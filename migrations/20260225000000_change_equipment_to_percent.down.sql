-- Remove percentage columns
ALTER TABLE world_state DROP COLUMN IF EXISTS house_lit_percent;
ALTER TABLE world_state DROP COLUMN IF EXISTS facility_lit_percent;
ALTER TABLE world_state DROP COLUMN IF EXISTS light_lit_percent;
ALTER TABLE world_state DROP COLUMN IF EXISTS factory_lit_percent;

-- Restore boolean columns
ALTER TABLE world_state ADD COLUMN is_light_enabled BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE world_state ADD COLUMN is_factory_enabled BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE world_state ADD COLUMN is_house_enabled BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE world_state ADD COLUMN is_facility_enabled BOOLEAN NOT NULL DEFAULT false;
