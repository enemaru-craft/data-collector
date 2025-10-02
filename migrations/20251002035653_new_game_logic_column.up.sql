ALTER TABLE world_state
ADD COLUMN is_house_enabled BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE world_state
ADD COLUMN is_facility_enabled BOOLEAN NOT NULL DEFAULT false;