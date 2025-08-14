-- sessions table
CREATE TABLE sessions (
    session_id VARCHAR(36) PRIMARY KEY,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP,
    description TEXT
);
-- devices table
CREATE TABLE devices (
    device_id VARCHAR(36) PRIMARY KEY,
    device_type VARCHAR(50) NOT NULL
);
-- session_devices: デバイスとセッションの中間テーブル
CREATE TABLE session_devices (
    id SERIAL PRIMARY KEY,
    session_id VARCHAR(36) NOT NULL,
    device_id VARCHAR(36) NOT NULL,
    UNIQUE(session_id, device_id),
    FOREIGN KEY (session_id) REFERENCES sessions(session_id) ON DELETE CASCADE,
    FOREIGN KEY (device_id) REFERENCES devices(device_id) ON DELETE CASCADE
);
-- power_logs table
CREATE TABLE power_logs (
    log_id SERIAL PRIMARY KEY,
    session_device_id INT NOT NULL,
    timestamp TIMESTAMP(3) NOT NULL,
    power REAL NOT NULL,
    gps_lat DECIMAL(9, 7),
    gps_lon DECIMAL(10, 7),
    FOREIGN KEY (session_device_id) REFERENCES session_devices(id) ON DELETE CASCADE,
    UNIQUE(session_device_id, timestamp)
);