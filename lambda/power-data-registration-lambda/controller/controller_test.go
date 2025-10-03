package controller_test

import (
	"context"
	"database/sql"
	"testing"

	"power-manager/controller"

	"github.com/DATA-DOG/go-sqlmock"
)

type mockRepo struct{}

func (m *mockRepo) RegisterNewPowerLog(ctx context.Context, tx *sql.Tx, sessionID, deviceID, gpsLat, gpsLon string, power float32) error {
	return nil
}

func (m *mockRepo) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	db, _, _ := sqlmock.New()
	tx, _ := db.Begin()
	return tx, nil
}

func (m *mockRepo) RegisterNewPowerLogToDynamoDB(ctx context.Context, sessionID, deviceID, gpsLat, gpsLon string, power float32) error {
	return nil
}

func TestNewLogController(t *testing.T) {
	ctr := controller.NewLogController(&mockRepo{})
	if ctr == nil {
		t.Fatal("Expected NewLogController to return a non-nil controller")
	}
}

func TestRegisterPower_InvalidJSON(t *testing.T) {
	ctr := controller.NewLogController(&mockRepo{})

	_, err := ctr.RegisterPower(context.Background(), []byte("Invalid JSON"))
	if err == nil {
		t.Fatal("Expected error for invalid JSON, but got none")
	}
}
