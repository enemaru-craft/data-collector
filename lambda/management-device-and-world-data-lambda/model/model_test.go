package model

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestGetPowerHistory(t *testing.T) {
	// モックデータベースを作成
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	// レポジトリを作成
	repo := &ManagementRepository{
		db: db,
	}

	t.Run("単一バケット、単一ログのテスト", func(t *testing.T) {
		sessionId := "test-session"

		mock.ExpectBegin()
		tx, err := db.Begin()
		assert.NoError(t, err)
		defer tx.Rollback()

		rows := sqlmock.NewRows([]string{"bucket", "device_id", "device_type", "power"}).
			AddRow(
				time.Date(2025, 9, 22, 19, 27, 0, 0, time.UTC),
				"M5-22-geothermal-1",
				"geothermal",
				20.0,
			)

		mock.ExpectPrepare("SELECT").ExpectQuery().WithArgs(sessionId).WillReturnRows(rows)

		// calculateTotalPowerのモック
		totalPowerRows := sqlmock.NewRows([]string{"bucket", "timestamp", "power", "device_id", "device_type"}).
			AddRow(
				time.Date(2025, 9, 22, 19, 27, 0, 0, time.UTC),
				time.Date(2025, 9, 22, 19, 27, 15, 0, time.UTC),
				20.0,
				"M5-22-geothermal-1",
				"geothermal",
			)
		mock.ExpectPrepare("SELECT").ExpectQuery().WithArgs(sessionId, 10).WillReturnRows(totalPowerRows)

		result, err := repo.GetPowerHistory(context.Background(), tx, sessionId)

		assert.NoError(t, err)

		// 時間のセパレートが正しいか確認する
		assert.Equal(t, []string{"19:27:00"}, result.TimeLabels, "19:27:00が期待されましたが､時間のロジックがおかしいので帰ってきませんでした")
		// 発電量の計算が正しいか確認する
		assert.Equal(t, []float64{20.0}, result.Geothermal, "20.0が期待されましたが､地熱のロジックがおかしいので帰ってきませんでした")

	})

	t.Run("時間をまたがないテスト", func(t *testing.T) {
		sessionID := "9999"

		mock.ExpectBegin()
		tx, err := db.Begin()
		assert.NoError(t, err)
		defer tx.Rollback()

		geothermal := fmt.Sprintf("M5-%s-geothermal-1", sessionID)
		rows := sqlmock.NewRows([]string{"bucket", "device_id", "device_type", "power"}).
			AddRow(
				time.Date(2025, 9, 22, 19, 27, 0, 0, time.UTC),
				geothermal,
				"geothermal",
				80.0,
			).
			AddRow(
				time.Date(2025, 9, 22, 19, 27, 0, 0, time.UTC),
				geothermal,
				"geothermal",
				20.0,
			)

		mock.ExpectPrepare("SELECT").ExpectQuery().WithArgs(sessionID).WillReturnRows(rows)

		// calculateTotalPowerのモック
		totalPowerRows := sqlmock.NewRows([]string{"bucket", "timestamp", "power", "device_id", "device_type"}).
			AddRow(
				time.Date(2025, 9, 22, 19, 27, 0, 0, time.UTC),
				time.Date(2025, 9, 22, 19, 27, 15, 0, time.UTC),
				80.0,
				geothermal,
				"geothermal",
			).
			AddRow(
				time.Date(2025, 9, 22, 19, 27, 0, 0, time.UTC),
				time.Date(2025, 9, 22, 19, 27, 20, 0, time.UTC),
				20.0,
				geothermal,
				"geothermal",
			)
		mock.ExpectPrepare("SELECT").ExpectQuery().WithArgs(sessionID, 10).WillReturnRows(totalPowerRows)

		result, err := repo.GetPowerHistory(context.Background(), tx, sessionID)

		assert.NoError(t, err)

		// 時間のセパレートが正しいか確認する
		assert.Equal(t, []string{"19:27:00"}, result.TimeLabels)
		// 一つの時間の区切りにおいて､複数のログがあっても､正しく集計されることを確認する
		assert.Equal(t, []float64{100.0}, result.Geothermal)
	})

	t.Run("複数バケット、境界を1つまたぐテスト", func(t *testing.T) {
		sessionId := "test-session"

		mock.ExpectBegin()
		tx, err := db.Begin()
		assert.NoError(t, err)
		defer tx.Rollback()

		rows := sqlmock.NewRows([]string{"bucket", "device_id", "device_type", "power"}).
			AddRow(
				time.Date(2025, 9, 22, 19, 27, 0, 0, time.UTC),
				"M5-22-geothermal-1",
				"geothermal",
				20.0,
			).
			AddRow(
				time.Date(2025, 9, 22, 19, 28, 0, 0, time.UTC),
				"M5-22-geothermal-1",
				"geothermal",
				80.0,
			)

		mock.ExpectPrepare("SELECT").ExpectQuery().WithArgs(sessionId).WillReturnRows(rows)

		// calculateTotalPowerのモック
		totalPowerRows := sqlmock.NewRows([]string{"bucket", "timestamp", "power", "device_id", "device_type"}).
			AddRow(
				time.Date(2025, 9, 22, 19, 27, 0, 0, time.UTC),
				time.Date(2025, 9, 22, 19, 27, 33, 0, time.UTC),
				20.0,
				"M5-22-geothermal-1",
				"geothermal",
			).
			AddRow(
				time.Date(2025, 9, 22, 19, 28, 0, 0, time.UTC),
				time.Date(2025, 9, 22, 19, 28, 12, 0, time.UTC),
				80.0,
				"M5-22-geothermal-1",
				"geothermal",
			)
		mock.ExpectPrepare("SELECT").ExpectQuery().WithArgs(sessionId, 10).WillReturnRows(totalPowerRows)

		// テスト実行
		result, err := repo.GetPowerHistory(context.Background(), tx, sessionId)

		// 検証
		assert.NoError(t, err)
		assert.NotNil(t, result)

		assert.Equal(t, []string{"19:27:00", "19:28:00"}, result.TimeLabels, "19:27:00, 19:28:00が期待されましたが､時間のロジックがおかしいので帰ってきませんでした")
		assert.Equal(t, []float64{20.0, 80.0}, result.Geothermal, "20.0, 80.0が期待されましたが､地熱のロジックがおかしいので帰ってきませんでした")

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("複数バケット、境界を2つまたぐテスト", func(t *testing.T) {
		sessionId := "test-session"

		mock.ExpectBegin()
		tx, err := db.Begin()
		assert.NoError(t, err)
		defer tx.Rollback()

		rows := sqlmock.NewRows([]string{"bucket", "device_id", "device_type", "power"}).
			AddRow(
				time.Date(2025, 9, 22, 19, 27, 0, 0, time.UTC),
				"M5-22-geothermal-1",
				"geothermal",
				20.0,
			).
			AddRow(
				time.Date(2025, 9, 22, 19, 28, 0, 0, time.UTC),
				"M5-22-geothermal-1",
				"geothermal",
				80.0,
			).
			AddRow(
				time.Date(2025, 9, 22, 19, 29, 0, 0, time.UTC),
				"M5-22-geothermal-1",
				"geothermal",
				10.0,
			)

		mock.ExpectPrepare("SELECT").ExpectQuery().WithArgs(sessionId).WillReturnRows(rows)

		// calculateTotalPowerのモック
		totalPowerRows := sqlmock.NewRows([]string{"bucket", "timestamp", "power", "device_id", "device_type"}).
			AddRow(
				time.Date(2025, 9, 22, 19, 27, 0, 0, time.UTC),
				time.Date(2025, 9, 22, 19, 27, 33, 0, time.UTC),
				20.0,
				"M5-22-geothermal-1",
				"geothermal",
			).
			AddRow(
				time.Date(2025, 9, 22, 19, 28, 0, 0, time.UTC),
				time.Date(2025, 9, 22, 19, 28, 12, 0, time.UTC),
				80.0,
				"M5-22-geothermal-1",
				"geothermal",
			).
			AddRow(
				time.Date(2025, 9, 22, 19, 29, 0, 0, time.UTC),
				time.Date(2025, 9, 22, 19, 29, 10, 0, time.UTC),
				10.0,
				"M5-22-geothermal-1",
				"geothermal",
			)
		mock.ExpectPrepare("SELECT").ExpectQuery().WithArgs(sessionId, 10).WillReturnRows(totalPowerRows)

		// テスト実行
		result, err := repo.GetPowerHistory(context.Background(), tx, sessionId)

		assert.NoError(t, err)
		assert.NotNil(t, result)

		assert.Equal(t, []string{"19:27:00", "19:28:00", "19:29:00"}, result.TimeLabels, "19:27:00, 19:28:00, 19:29:00が期待されましたが､時間のロジックがおかしいので帰ってきませんでした")
		assert.Equal(t, []float64{20.0, 80.0, 10.0}, result.Geothermal, "20.0, 80.0, 10.0が期待されましたが､地熱のロジックがおかしいので帰ってきませんでした")
	})

	t.Run("複数バケット、境界を2つまたぐテスト､一つ目の境界の後に2つ発電量が記録されている", func(t *testing.T) {
		sessionId := "test-session"

		mock.ExpectBegin()
		tx, err := db.Begin()
		assert.NoError(t, err)
		defer tx.Rollback()

		rows := sqlmock.NewRows([]string{"bucket", "device_id", "device_type", "power"}).
			AddRow(
				time.Date(2025, 9, 22, 19, 27, 0, 0, time.UTC),
				"M5-22-geothermal-1",
				"geothermal",
				20.0,
			).
			AddRow(
				time.Date(2025, 9, 22, 19, 28, 0, 0, time.UTC),
				"M5-22-geothermal-1",
				"geothermal",
				80.0,
			).
			AddRow(
				time.Date(2025, 9, 22, 19, 28, 0, 0, time.UTC),
				"M5-22-geothermal-1",
				"geothermal",
				70.0,
			).
			AddRow(
				time.Date(2025, 9, 22, 19, 29, 0, 0, time.UTC),
				"M5-22-geothermal-1",
				"geothermal",
				10.0,
			)

		mock.ExpectPrepare("SELECT").ExpectQuery().WithArgs(sessionId).WillReturnRows(rows)

		// calculateTotalPowerのモック
		totalPowerRows := sqlmock.NewRows([]string{"bucket", "timestamp", "power", "device_id", "device_type"}).
			AddRow(
				time.Date(2025, 9, 22, 19, 27, 0, 0, time.UTC),
				time.Date(2025, 9, 22, 19, 27, 33, 0, time.UTC),
				20.0,
				"M5-22-geothermal-1",
				"geothermal",
			).
			AddRow(
				time.Date(2025, 9, 22, 19, 28, 0, 0, time.UTC),
				time.Date(2025, 9, 22, 19, 28, 12, 0, time.UTC),
				80.0,
				"M5-22-geothermal-1",
				"geothermal",
			).
			AddRow(
				time.Date(2025, 9, 22, 19, 28, 0, 0, time.UTC),
				time.Date(2025, 9, 22, 19, 28, 15, 0, time.UTC),
				70.0,
				"M5-22-geothermal-1",
				"geothermal",
			).
			AddRow(
				time.Date(2025, 9, 22, 19, 29, 0, 0, time.UTC),
				time.Date(2025, 9, 22, 19, 29, 10, 0, time.UTC),
				10.0,
				"M5-22-geothermal-1",
				"geothermal",
			)
		mock.ExpectPrepare("SELECT").ExpectQuery().WithArgs(sessionId, 10).WillReturnRows(totalPowerRows)

		// テスト実行
		result, err := repo.GetPowerHistory(context.Background(), tx, sessionId)

		assert.NoError(t, err)
		assert.NotNil(t, result)

		assert.Equal(t, []string{"19:27:00", "19:28:00", "19:29:00"}, result.TimeLabels, "19:27:00, 19:28:00, 19:29:00が期待されましたが､時間のロジックがおかしいので帰ってきませんでした")
		assert.Equal(t, []float64{20.0, 150.0, 10.0}, result.Geothermal, "20.0, 150.0, 10.0が期待されましたが､地熱のロジックがおかしいので帰ってきませんでした")
	})
}

// エラーケースのテスト
func TestGetPowerHistoryErrorCases(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &ManagementRepository{
		db: db,
	}

	t.Run("SQLエラーのテスト", func(t *testing.T) {
		sessionId := "test-session"

		// モックトランザクションを開始
		mock.ExpectBegin()
		tx, err := db.Begin()
		assert.NoError(t, err)
		defer tx.Rollback()

		mock.ExpectPrepare("SELECT").WillReturnError(sql.ErrConnDone)

		result, err := repo.GetPowerHistory(context.Background(), tx, sessionId)

		assert.Error(t, err)
		assert.Equal(t, PowerChartData{}, result, "エラー時は空のPowerChartDataが返るはずです")
		assert.NoError(t, mock.ExpectationsWereMet(), "モックの期待値が満たされていません")
	})

	t.Run("空データのテスト", func(t *testing.T) {
		sessionId := "test-session"

		// モックトランザクションを開始
		mock.ExpectBegin()
		tx, err := db.Begin()
		assert.NoError(t, err)
		defer tx.Rollback()

		// GetPowerHistoryのメインクエリ（空データ）
		rows := sqlmock.NewRows([]string{"bucket", "device_id", "device_type", "power"})
		mock.ExpectPrepare("SELECT").ExpectQuery().WithArgs(sessionId).WillReturnRows(rows)

		// calculateTotalPowerのクエリ（空データ）
		totalPowerRows := sqlmock.NewRows([]string{"bucket", "timestamp", "power", "device_id", "device_type"})
		mock.ExpectPrepare("SELECT").ExpectQuery().WithArgs(sessionId, 10).WillReturnRows(totalPowerRows)

		result, err := repo.GetPowerHistory(context.Background(), tx, sessionId)

		assert.NoError(t, err)
		// 現在の実装では空のスライス(timeLabelsはnil、他は空スライス)を返す
		expected := PowerChartData{
			TimeLabels: nil,         // var timeLabels []string のデフォルトはnil
			Geothermal: []float64{}, // make([]float64, 0) と同等
			Hydro:      []float64{},
			Wind:       []float64{},
			Solar:      []float64{},
			TotalPower: 0,
		}
		assert.Equal(t, expected, result, "空データ時は空のPowerChartDataが返るはずです")
	})
}
