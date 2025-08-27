ALTER TABLE world_state
ADD CONSTRAINT unique_session_id UNIQUE(session_id);