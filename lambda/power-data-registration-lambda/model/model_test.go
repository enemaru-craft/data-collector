package model

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestNewLogRepository(t *testing.T) {
	db, _, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	// DynamoDBクライアントはnilでテスト（実際の接続は統合テストで行う）
	repo := NewLogRepository(db, nil)

	assert.NotNil(t, repo)
	assert.Equal(t, db, repo.db)
}

func TestRegisterNewPowerLog(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := NewLogRepository(db, nil)

	t.Run("正常系: パワーログの登録が成功", func(t *testing.T) {
		sessionID := "test-session"
		deviceID := "test-device"
		gpsLat := "35.6762"
		gpsLon := "139.6503"
		power := float32(100.5)

		mock.ExpectBegin()
		tx, err := db.Begin()
		assert.NoError(t, err)

		// session_devices テーブルからIDを取得するクエリをモック
		mock.ExpectPrepare("SELECT id FROM session_devices WHERE session_id = \\$1 AND device_id = \\$2").
			ExpectQuery().
			WithArgs(sessionID, deviceID).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

		// devices テーブルからdevice_typeを取得するクエリをモック
		mock.ExpectPrepare("SELECT device_type FROM devices WHERE device_id = \\$1").
			ExpectQuery().
			WithArgs(deviceID).
			WillReturnRows(sqlmock.NewRows([]string{"device_type"}).AddRow("solar"))

		// power_logs テーブルにINSERTするクエリをモック
		mock.ExpectPrepare("INSERT INTO power_logs").
			ExpectExec().
			WithArgs(1, power, gpsLat, gpsLon, "solar").
			WillReturnResult(sqlmock.NewResult(1, 1))

		err = repo.RegisterNewPowerLog(context.Background(), tx, sessionID, deviceID, gpsLat, gpsLon, power)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("異常系: session_deviceが見つからない", func(t *testing.T) {
		sessionID := "test-session"
		deviceID := "non-existent-device"
		gpsLat := "35.6762"
		gpsLon := "139.6503"
		power := float32(100.5)

		mock.ExpectBegin()
		tx, err := db.Begin()
		assert.NoError(t, err)

		// session_devices テーブルからIDを取得するクエリで見つからない
		mock.ExpectPrepare("SELECT id FROM session_devices WHERE session_id = \\$1 AND device_id = \\$2").
			ExpectQuery().
			WithArgs(sessionID, deviceID).
			WillReturnError(sql.ErrNoRows)

		err = repo.RegisterNewPowerLog(context.Background(), tx, sessionID, deviceID, gpsLat, gpsLon, power)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session_device not found")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("異常系: deviceが見つからない", func(t *testing.T) {
		sessionID := "test-session"
		deviceID := "non-existent-device"
		gpsLat := "35.6762"
		gpsLon := "139.6503"
		power := float32(100.5)

		mock.ExpectBegin()
		tx, err := db.Begin()
		assert.NoError(t, err)

		// session_devices テーブルからIDを取得するクエリは成功
		mock.ExpectPrepare("SELECT id FROM session_devices WHERE session_id = \\$1 AND device_id = \\$2").
			ExpectQuery().
			WithArgs(sessionID, deviceID).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

		// devices テーブルからdevice_typeを取得するクエリで見つからない
		mock.ExpectPrepare("SELECT device_type FROM devices WHERE device_id = \\$1").
			ExpectQuery().
			WithArgs(deviceID).
			WillReturnError(sql.ErrNoRows)

		err = repo.RegisterNewPowerLog(context.Background(), tx, sessionID, deviceID, gpsLat, gpsLon, power)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "device_type not found")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestBeginTx(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := NewLogRepository(db, nil)

	t.Run("正常系: トランザクション開始が成功", func(t *testing.T) {
		mock.ExpectBegin()

		tx, err := repo.BeginTx(context.Background(), nil)

		assert.NoError(t, err)
		assert.NotNil(t, tx)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// initialize.go のテスト用のヘルパー関数
// 実際のDBやAWS接続のテストは統合テストで行うため、ここでは関数の存在確認のみ
func TestInitFunctions(t *testing.T) {
	t.Run("InitDB関数が存在することを確認", func(t *testing.T) {
		// 環境変数が設定されていない場合はエラーになるが、関数が存在することは確認できる
		_, err := InitDB()
		// エラーは期待される（環境変数未設定のため）が、パニックしないことを確認
		assert.Error(t, err)
	})

	t.Run("InitDynamoDB関数が存在することを確認", func(t *testing.T) {
		// InitDynamoDB関数が存在し、パニックしないことを確認
		client, err := InitDynamoDB()
		// 成功する場合とエラーになる場合があるが、どちらも正常
		if err == nil {
			assert.NotNil(t, client)
		} else {
			assert.Error(t, err)
		}
	})
}
