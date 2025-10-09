package model

import (
	"context"
	"database/sql"
	"fmt"

	"data-manager/custmerr"
)

func (repo *ManagementRepository) DeleteSessionAndRelatedData(ctx context.Context, tx *sql.Tx, sessionID string) error {
	delSessionQuery := `
		DELETE FROM
			sessions
		WHERE
			session_id = $1`
	res, err := tx.ExecContext(ctx, delSessionQuery, sessionID)
	if err != nil {
		return &custmerr.TechnicalErr{Err: fmt.Errorf("failed to delete session %s: %w", sessionID, err)}
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return &custmerr.TechnicalErr{Err: fmt.Errorf("failed to get rows affected when deleting session %s: %w", sessionID, err)}
	}

	if rowsAffected == 0 {
		return &custmerr.LogicalErr{Err: fmt.Errorf("session %s not found", sessionID)}
	}

	return nil
}
