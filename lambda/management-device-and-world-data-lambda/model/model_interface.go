package model

import (
	"context"
	"database/sql"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type ManagementRepository struct {
	db *sql.DB
	dc *dynamodb.Client
}

func NewManagementRepository(db *sql.DB, dc *dynamodb.Client) *ManagementRepository {
	return &ManagementRepository{db: db, dc: dc}
}

type ManagementRepositoryInterface interface {
	CreateSessionIfNotExists(ctx context.Context, tx *sql.Tx, sessionID string) error
	CheckDeviceNotExists(ctx context.Context, tx *sql.Tx, deviceID string) error
	RegisterNewPowerGenerationModule(ctx context.Context, tx *sql.Tx, sessionID, deviceID, deviceType string) error
	GetLatestPowerData(ctx context.Context, tx *sql.Tx, deviceType string, sessionId string) (float32, string, string, error)
	GetMultipleDevicesPowerDataFromDynamoDB(ctx context.Context, deviceType string, sessionId string) (MultipleDevicePowerResponse, error)
	CreateNewWorldIfNotExists(ctx context.Context, tx *sql.Tx, sessionID string) error
	TurnOnEquipment(ctx context.Context, tx *sql.Tx, sessionID string, equipment string) (CurrentWorldState, error)
	TurnOffEquipment(ctx context.Context, tx *sql.Tx, sessionID string, equipment string) (CurrentWorldState, error)
	GetCurrentWorldState(ctx context.Context, tx *sql.Tx, sessionID string) (CurrentWorldState, error)
	GetPowerHistory(ctx context.Context, tx *sql.Tx, sessionID string) (PowerChartData, error)
	GetGameResult(ctx context.Context, tx *sql.Tx, sessionID string) (GameResult, error)
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

func (repo *ManagementRepository) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return repo.db.BeginTx(ctx, opts)
}
