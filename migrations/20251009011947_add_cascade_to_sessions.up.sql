-- sessionsテーブル削除時に関連するすべてのデータを削除するCASCADE制約を設定

-- world_stateテーブルのsessions外部キーにCASCADE制約を追加
ALTER TABLE world_state
DROP CONSTRAINT IF EXISTS world_state_session_id_fkey;

ALTER TABLE world_state
ADD CONSTRAINT world_state_session_id_fkey
FOREIGN KEY (session_id) REFERENCES sessions(session_id) ON DELETE CASCADE;

-- session_devicesの制約も念のため確認・設定
ALTER TABLE session_devices
DROP CONSTRAINT IF EXISTS session_devices_session_id_fkey;

ALTER TABLE session_devices
ADD CONSTRAINT session_devices_session_id_fkey
FOREIGN KEY (session_id) REFERENCES sessions(session_id) ON DELETE CASCADE;