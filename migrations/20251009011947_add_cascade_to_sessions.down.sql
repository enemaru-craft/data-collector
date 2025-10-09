-- CASCADE制約を削除して元の制約に戻す
ALTER TABLE world_state
DROP CONSTRAINT IF EXISTS world_state_session_id_fkey;

ALTER TABLE world_state
ADD CONSTRAINT world_state_session_id_fkey
FOREIGN KEY (session_id) REFERENCES sessions(session_id);