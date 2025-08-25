package model

import (
	"context"
	"data-manager/custmerr"
	"database/sql"
	"fmt"
)

func (repo *ManagementRepository) GetLatestPowerData(ctx context.Context, tx *sql.Tx, deviceType string, sessionId string) (float32, error) {
	stmt, err := tx.PrepareContext(ctx,
		`
		SELECT
			pl.power
		FROM
			power_logs pl
		JOIN
			session_devices sd ON pl.session_device_id = sd.id
		JOIN
			devices d ON sd.device_id = d.device_id
		WHERE
			sd.session_id = $1
			AND d.device_type = $2
		ORDER BY
			pl.timestamp DESC
		LIMIT 1;

		`)
	if err != nil {
		return 0, &custmerr.TechnicalErr{Err: err}
	}
	defer stmt.Close()

	var latestPower float32
	err = stmt.QueryRowContext(ctx, sessionId, deviceType).Scan(&latestPower)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, &custmerr.LogicalErr{Err: fmt.Errorf("no power data found for device type %s", deviceType)}
		}
		return 0, &custmerr.TechnicalErr{Err: fmt.Errorf("failed to get latest power data: %w", err)}
	}

	return latestPower, nil
}
