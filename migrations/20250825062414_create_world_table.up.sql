CREATE TABLE world_state (
    id SERIAL PRIMARY KEY,
    session_id VARCHAR(36) NOT NULL,
    is_light_enabled BOOLEAN NOT NULL DEFAULT false,
    is_train_enabled BOOLEAN NOT NULL DEFAULT false,
    is_factory_enabled BOOLEAN NOT NULL DEFAULT false,
    is_blackout BOOLEAN NOT NULL DEFAULT false,
    villagers_text JSONB NOT NULL DEFAULT '[]'::jsonb,
    total_power REAL NOT NULL DEFAULT 0,
    surplus_power REAL NOT NULL DEFAULT 0,
    timestamp TIMESTAMP(3) NOT NULL,
    FOREIGN KEY (session_id) REFERENCES sessions(session_id)
);