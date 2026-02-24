-- Remove boolean columns for equipment state
ALTER TABLE world_state DROP COLUMN IF EXISTS is_light_enabled;
ALTER TABLE world_state DROP COLUMN IF EXISTS is_factory_enabled;
ALTER TABLE world_state DROP COLUMN IF EXISTS is_house_enabled;
ALTER TABLE world_state DROP COLUMN IF EXISTS is_facility_enabled;

-- Add percentage columns for equipment state
ALTER TABLE world_state ADD COLUMN house_lit_percent INTEGER NOT NULL DEFAULT 0;
ALTER TABLE world_state ADD COLUMN facility_lit_percent INTEGER NOT NULL DEFAULT 0;
ALTER TABLE world_state ADD COLUMN light_lit_percent INTEGER NOT NULL DEFAULT 0;
ALTER TABLE world_state ADD COLUMN factory_lit_percent INTEGER NOT NULL DEFAULT 0;
