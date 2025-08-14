package model

import (
	"context"
	"database/sql"
	"fmt"
	"power-manager/custmerr"
)

func RegisterNewPowerGenerationModule(ctx context.Context, tx *sql.Tx, sessionID, deviceID, geoLat, geoLon string, power float32) error {
	var sessionDeviceID int

	getIdStmt, err := tx.PrepareContext(ctx, `
		SELECT
			id
		FROM
			session_devices
		WHERE
			session_id = $1
		AND
			device_id = $2
	`)
	if err != nil {
		return &custmerr.TechnicalErr{Err: fmt.Errorf("failed to prepare get session_device ID statement: %w", err)}
	}
	defer getIdStmt.Close()

	err = getIdStmt.QueryRowContext(ctx, sessionID, deviceID).Scan(&sessionDeviceID)
	if err != nil {
		if err == sql.ErrNoRows {
			return &custmerr.LogicalErr{Err: fmt.Errorf("session_device not found for session_id %s and device_id %s", sessionID, deviceID)}
		}
		return &custmerr.TechnicalErr{Err: fmt.Errorf("failed to get session_device ID: %w", err)}
	}

	var deviceType string
	getTypeStmt, err := tx.PrepareContext(ctx, `
		SELECT
			device_type
		FROM
			devices
		WHERE
			device_id = $1
	`)
	if err != nil {
		return &custmerr.TechnicalErr{Err: fmt.Errorf("failed to prepare get device_type statement: %w", err)}
	}
	defer getTypeStmt.Close()

	err = getTypeStmt.QueryRowContext(ctx, deviceID).Scan(&deviceType)
	if err != nil {
		if err == sql.ErrNoRows {
			return &custmerr.LogicalErr{Err: fmt.Errorf("device_type not found for device_id %s", deviceID)}
		}
		return &custmerr.TechnicalErr{Err: fmt.Errorf("failed to get device_type: %w", err)}
	}

	registerPowerStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO
			power_logs (session_device_id, timestamp, power, gps_lat, gps_lon, device_type)
		VALUES
			($1, NOW(), $2, $3, $4, $5)
	`)
	if err != nil {
		return &custmerr.TechnicalErr{Err: fmt.Errorf("failed to prepare register power statement: %w", err)}
	}
	defer registerPowerStmt.Close()

	_, err = registerPowerStmt.ExecContext(ctx, sessionDeviceID, power, geoLat, geoLon, deviceType)
	if err != nil {
		return &custmerr.TechnicalErr{Err: fmt.Errorf("failed to register power data: %w", err)}
	}

	return nil
}
