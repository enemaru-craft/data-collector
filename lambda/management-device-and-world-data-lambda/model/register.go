package model

import (
	"context"
	"data-manager/custmerr"
	"database/sql"
	"fmt"
)

func CheckSessionNotExists(ctx context.Context, tx *sql.Tx, sessionID string) error {
	stmt, err := tx.PrepareContext(ctx, `
		SELECT
			EXISTS
		(SELECT
			1
		FROM
			sessions
		WHERE
			session_id = $1)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	var exists bool
	err = stmt.QueryRowContext(ctx, sessionID).Scan(&exists)
	if err != nil {
		customErr := &custmerr.TechnicalErr{Err: fmt.Errorf("failed to check session existence: %w", err)}
		return customErr
	}
	if exists {
		return &custmerr.LogicalErr{Err: fmt.Errorf("session with ID %s already exists", sessionID)}
	}
	return nil
}

func CheckDeviceNotExists(ctx context.Context, tx *sql.Tx, deviceID string) error {
	stmt, err := tx.PrepareContext(ctx, `
		SELECT
			EXISTS
		(SELECT
			1
		FROM
			devices
		WHERE
			device_id = $1)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	var exists bool
	err = stmt.QueryRowContext(ctx, deviceID).Scan(&exists)
	if err != nil {
		return &custmerr.TechnicalErr{Err: fmt.Errorf("failed to check device existence: %w", err)}
	}
	if exists {
		return &custmerr.LogicalErr{Err: fmt.Errorf("device with ID %s already exists", deviceID)}
	}
	return nil
}

func RegisterNewPowerGenerationModule(ctx context.Context, tx *sql.Tx, sessionID, deviceID, deviceType string) error {
	// sessions 用の PreparedStatement
	stmtSession, err := tx.PrepareContext(ctx, `
        INSERT INTO
			sessions(session_id, start_time)
        VALUES
			($1, NOW())
        ON CONFLICT
			(session_id)
		DO NOTHING
    `)
	if err != nil {
		return &custmerr.TechnicalErr{Err: fmt.Errorf("failed to prepare sessions statement: %w", err)}
	}
	defer stmtSession.Close()

	if _, err := stmtSession.ExecContext(ctx, sessionID); err != nil {
		return &custmerr.TechnicalErr{Err: fmt.Errorf("failed to insert session: %w", err)}
	}

	// devices 用の PreparedStatement
	stmtDevice, err := tx.PrepareContext(ctx, `
        INSERT INTO
			devices(device_id, device_type)
        VALUES
			($1, $2)
        ON CONFLICT
			(device_id)
		DO NOTHING
    `)
	if err != nil {
		return &custmerr.TechnicalErr{Err: fmt.Errorf("failed to prepare devices statement: %w", err)}
	}
	defer stmtDevice.Close()

	if _, err := stmtDevice.ExecContext(ctx, deviceID, deviceType); err != nil {
		return &custmerr.TechnicalErr{Err: fmt.Errorf("failed to insert device: %w", err)}
	}

	// session_devices 用の PreparedStatement
	stmtSessionDevice, err := tx.PrepareContext(ctx, `
        INSERT INTO
			session_devices(session_id, device_id)
        VALUES
			($1, $2)
        ON CONFLICT
			(session_id, device_id)
		DO NOTHING
    `)
	if err != nil {
		return &custmerr.TechnicalErr{Err: fmt.Errorf("failed to prepare session_devices statement: %w", err)}
	}
	defer stmtSessionDevice.Close()

	if _, err := stmtSessionDevice.ExecContext(ctx, sessionID, deviceID); err != nil {
		return &custmerr.TechnicalErr{Err: fmt.Errorf("failed to insert session_device: %w", err)}
	}

	return nil
}
