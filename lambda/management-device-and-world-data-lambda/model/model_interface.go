package model

import (
	"context"
	"database/sql"
)

type ManagementRepository struct {
	db *sql.DB
}

func NewManagementRepository(db *sql.DB) *ManagementRepository {
	return &ManagementRepository{db: db}
}

type ManagementRepositoryInterface interface {
	CreateSessionIfNotExists(ctx context.Context, tx *sql.Tx, sessionID string) error
	CheckDeviceNotExists(ctx context.Context, tx *sql.Tx, deviceID string) error
	RegisterNewPowerGenerationModule(ctx context.Context, tx *sql.Tx, sessionID, deviceID, deviceType string) error
	GetLatestPowerData(ctx context.Context, tx *sql.Tx, deviceType string, sessionId string) (float32, error)
	CreateNewWorldIfNotExists(ctx context.Context, tx *sql.Tx, sessionID string) error
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

func (repo *ManagementRepository) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return repo.db.BeginTx(ctx, opts)
}
