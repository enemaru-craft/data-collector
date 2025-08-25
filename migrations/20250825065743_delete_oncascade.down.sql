-- --- DOWN MIGRATION ---
-- 外部キー制約を削除
ALTER TABLE session_devices DROP CONSTRAINT session_devices_session_id_fkey;
ALTER TABLE session_devices DROP CONSTRAINT session_devices_device_id_fkey;
ALTER TABLE power_logs DROP CONSTRAINT power_logs_session_device_id_fkey;
-- 再作成（ON DELETE CASCADE あり）
ALTER TABLE session_devices
ADD CONSTRAINT session_devices_session_id_fkey FOREIGN KEY (session_id) REFERENCES sessions(session_id) ON DELETE CASCADE;
ALTER TABLE session_devices
ADD CONSTRAINT session_devices_device_id_fkey FOREIGN KEY (device_id) REFERENCES devices(device_id) ON DELETE CASCADE;
ALTER TABLE power_logs
ADD CONSTRAINT power_logs_session_device_id_fkey FOREIGN KEY (session_device_id) REFERENCES session_devices(id) ON DELETE CASCADE;