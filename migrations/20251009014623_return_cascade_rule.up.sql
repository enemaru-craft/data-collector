ALTER TABLE power_logs DROP CONSTRAINT IF EXISTS power_logs_session_device_id_fkey;
ALTER TABLE session_devices DROP CONSTRAINT IF EXISTS session_devices_session_id_fkey;
ALTER TABLE session_devices DROP CONSTRAINT IF EXISTS session_devices_device_id_fkey;

ALTER TABLE session_devices
ADD CONSTRAINT session_devices_session_id_fkey
FOREIGN KEY (session_id) REFERENCES sessions(session_id)
ON DELETE CASCADE;

ALTER TABLE session_devices
ADD CONSTRAINT session_devices_device_id_fkey
FOREIGN KEY (device_id) REFERENCES devices(device_id)
ON DELETE NO ACTION;  -- デバイスは削除しない

ALTER TABLE power_logs
ADD CONSTRAINT power_logs_session_device_id_fkey
FOREIGN KEY (session_device_id) REFERENCES session_devices(id)
ON DELETE CASCADE;
