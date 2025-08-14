ALTER TABLE power_logs
ADD COLUMN device_type VARCHAR(50) NOT NULL DEFAULT 'unknown';