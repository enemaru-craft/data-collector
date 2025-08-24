package model

import (
	"context"
	"data-manager/custmerr"
	"database/sql"
	"fmt"
)

func (repo *ManagementRepository) GetLatestPowerData(ctx context.Context, tx *sql.Tx, deviceType string) (float32, error) {
	stmt, err := tx.PrepareContext(ctx,
		`
		SELECT
			power
		FROM
			power_logs
		WHERE
			device_type = $1
		ORDER BY
			timestamp DESC
		LIMIT
			1
		`)
	if err != nil {
		return 0, &custmerr.TechnicalErr{Err: err}
	}
	defer stmt.Close()

	var latestPower float32
	err = stmt.QueryRowContext(ctx, deviceType).Scan(&latestPower)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, &custmerr.LogicalErr{Err: fmt.Errorf("no power data found for device type %s", deviceType)}
		}
		return 0, &custmerr.TechnicalErr{Err: fmt.Errorf("failed to get latest power data: %w", err)}
	}

	return latestPower, nil
}
