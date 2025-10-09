-- ===== 外部キー削除 =====
ALTER TABLE power_logs DROP CONSTRAINT IF EXISTS power_logs_session_device_id_fkey;
ALTER TABLE session_devices DROP CONSTRAINT IF EXISTS session_devices_session_id_fkey;
ALTER TABLE session_devices DROP CONSTRAINT IF EXISTS session_devices_device_id_fkey;

-- ===== 外部キー再作成（元の状態に戻す）=====
ALTER TABLE session_devices
ADD CONSTRAINT session_devices_session_id_fkey
FOREIGN KEY (session_id) REFERENCES sessions(session_id)
ON DELETE NO ACTION;

ALTER TABLE session_devices
ADD CONSTRAINT session_devices_device_id_fkey
FOREIGN KEY (device_id) REFERENCES devices(device_id)
ON DELETE NO ACTION;

ALTER TABLE power_logs
ADD CONSTRAINT power_logs_session_device_id_fkey
FOREIGN KEY (session_device_id) REFERENCES session_devices(id)
ON DELETE NO ACTION;
