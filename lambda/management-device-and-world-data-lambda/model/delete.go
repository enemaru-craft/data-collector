package model

import (
	"context"
	"database/sql"
	"fmt"

	"data-manager/custmerr"
)

func (repo *ManagementRepository) DeleteSessionAndRelatedData(ctx context.Context, tx *sql.Tx, sessionID string) error {
	stmt, err := tx.PrepareContext(ctx,
		`
		DELETE
		FROM
			sessions
		WHERE
			session_id = $1
	`)
	if err != nil {
		return &custmerr.TechnicalErr{Err: fmt.Errorf("failed to prepare delete sessions stmt: %w", err)}
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, sessionID)
	if err != nil {
		return &custmerr.TechnicalErr{Err: fmt.Errorf("failed to delete session: %w", err)}
	}

	return nil
}
