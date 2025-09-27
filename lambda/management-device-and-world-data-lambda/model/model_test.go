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
		bucketMinutes := 1

		mock.ExpectBegin()
		tx, err := db.Begin()
		assert.NoError(t, err)
		defer tx.Rollback()

		rows := sqlmock.NewRows([]string{"bucket", "timestamp", "power", "device_id", "device_type"}).
			AddRow(
				time.Date(2025, 9, 22, 19, 27, 0, 0, time.UTC),
				time.Date(2025, 9, 22, 19, 27, 15, 0, time.UTC),
				20.0,
				"M5-22-geothermal-1",
				"geothermal",
			)

		mock.ExpectPrepare("SELECT").ExpectQuery().WithArgs(sessionId, bucketMinutes).WillReturnRows(rows)

		result, err := repo.GetPowerHistory(context.Background(), tx, sessionId, bucketMinutes)

		assert.NoError(t, err)

		// 時間のセパレートが正しいか確認する
		assert.Equal(t, []string{"19:27"}, result.TimeLabels, "19:27が期待されましたが､時間のロジックがおかしいので帰ってきませんでした")
		// 発電量の計算が正しいか確認する
		assert.Equal(t, []float64{0.08333333333333333}, result.Geothermal, "0.08333333333333333が期待されましたが､地熱のロジックがおかしいので帰ってきませんでした")

	})

	t.Run("時間をまたがないテスト", func(t *testing.T) {
		sessionID := "9999"
		bucketMinutes := 1

		mock.ExpectBegin()
		tx, err := db.Begin()
		assert.NoError(t, err)
		defer tx.Rollback()

		geothermal := fmt.Sprintf("M5-%s-geothermal-1", sessionID)
		rows := sqlmock.NewRows([]string{"bucket", "timestamp", "power", "device_id", "device_type"}).
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

		mock.ExpectPrepare("SELECT").ExpectQuery().WithArgs(sessionID, bucketMinutes).WillReturnRows(rows)

		result, err := repo.GetPowerHistory(context.Background(), tx, sessionID, bucketMinutes)

		assert.NoError(t, err)

		// 時間のセパレートが正しいか確認する
		assert.Equal(t, []string{"19:27"}, result.TimeLabels)
		// 一つの時間の区切りにおいて､複数のログがあっても､正しく集計されることを確認する
		assert.Equal(t, []float64{0.625}, result.Geothermal)
	})

	t.Run("複数バケット、境界を1つまたぐテスト", func(t *testing.T) {
		sessionId := "test-session"
		bucketMinutes := 1

		mock.ExpectBegin()
		tx, err := db.Begin()
		assert.NoError(t, err)
		defer tx.Rollback()

		rows := sqlmock.NewRows([]string{"bucket", "timestamp", "power", "device_id", "device_type"}).
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

		mock.ExpectPrepare("SELECT").ExpectQuery().WithArgs(sessionId, bucketMinutes).WillReturnRows(rows)

		// テスト実行
		result, err := repo.GetPowerHistory(context.Background(), tx, sessionId, bucketMinutes)

		// 検証
		assert.NoError(t, err)
		assert.NotNil(t, result)

		assert.Equal(t, []string{"19:27", "19:28"}, result.TimeLabels, "19:27, 19:28が期待されましたが､時間のロジックがおかしいので帰ってきませんでした")
		assert.Equal(t, []float64{0.48910256410256414, 0.23589743589743592}, result.Geothermal, "0.48910256410256414, 0.23589743589743592が期待されましたが､地熱のロジックがおかしいので帰ってきませんでした")

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("複数バケット、境界を2つまたぐテスト", func(t *testing.T) {
		sessionId := "test-session"
		bucketMinutes := 1

		mock.ExpectBegin()
		tx, err := db.Begin()
		assert.NoError(t, err)
		defer tx.Rollback()

		rows := sqlmock.NewRows([]string{"bucket", "timestamp", "power", "device_id", "device_type"}).
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

		mock.ExpectPrepare("SELECT").ExpectQuery().WithArgs(sessionId, bucketMinutes).WillReturnRows(rows)

		// テスト実行
		result, err := repo.GetPowerHistory(context.Background(), tx, sessionId, bucketMinutes)

		assert.NoError(t, err)
		assert.NotNil(t, result)

		assert.Equal(t, []string{"19:27", "19:28", "19:29"}, result.TimeLabels, "19:27, 19:28, 19:29が期待されましたが､時間のロジックがおかしいので帰ってきませんでした")
		assert.Equal(t, []float64{0.48910256410256414, 0.9163572060123784, 0.04454022988505748}, result.Geothermal, "0.48910256410256414, 0.9163572060123784, 0.04454022988505748が期待されましたが､地熱のロジックがおかしいので帰ってきませんでした")
	})

	t.Run("複数バケット、境界を2つまたぐテスト､一つ目の境界の後に2つ発電量が記録されている", func(t *testing.T) {
		sessionId := "test-session"
		bucketMinutes := 1

		mock.ExpectBegin()
		tx, err := db.Begin()
		assert.NoError(t, err)
		defer tx.Rollback()

		rows := sqlmock.NewRows([]string{"bucket", "timestamp", "power", "device_id", "device_type"}).
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

		mock.ExpectPrepare("SELECT").ExpectQuery().WithArgs(sessionId, bucketMinutes).WillReturnRows(rows)

		// テスト実行
		result, err := repo.GetPowerHistory(context.Background(), tx, sessionId, bucketMinutes)

		assert.NoError(t, err)
		assert.NotNil(t, result)

		assert.Equal(t, []string{"19:27", "19:28", "19:29"}, result.TimeLabels, "19:27, 19:28, 19:29が期待されましたが､時間のロジックがおかしいので帰ってきませんでした")
		assert.Equal(t, []float64{0.48910256410256414, 0.8665792540792541, 0.04292929292929293}, result.Geothermal, "0.48910256410256414, 0.8665792540792541, 0.04292929292929293が期待されましたが､地熱のロジックがおかしいので帰ってきませんでした")
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
		bucketMinutes := 1

		// モックトランザクションを開始
		mock.ExpectBegin()
		tx, err := db.Begin()
		assert.NoError(t, err)
		defer tx.Rollback()

		mock.ExpectPrepare("SELECT").WillReturnError(sql.ErrConnDone)

		result, err := repo.GetPowerHistory(context.Background(), tx, sessionId, bucketMinutes)

		assert.Error(t, err)
		assert.Equal(t, PowerChartData{}, result, "エラー時は空のPowerChartDataが返るはずです")
		assert.NoError(t, mock.ExpectationsWereMet(), "モックの期待値が満たされていません")
	})

	t.Run("空データのテスト", func(t *testing.T) {
		sessionId := "test-session"
		bucketMinutes := 1

		// モックトランザクションを開始
		mock.ExpectBegin()
		tx, err := db.Begin()
		assert.NoError(t, err)
		defer tx.Rollback()

		rows := sqlmock.NewRows([]string{"bucket", "timestamp", "power", "device_id", "device_type"})
		mock.ExpectPrepare("SELECT").ExpectQuery().WithArgs(sessionId, bucketMinutes).WillReturnRows(rows)

		result, err := repo.GetPowerHistory(context.Background(), tx, sessionId, bucketMinutes)

		assert.NoError(t, err)
		// 現在の実装では空のスライス(timeLabelsはnil、他は空スライス)を返す
		expected := PowerChartData{
			TimeLabels: nil,         // var timeLabels []string のデフォルトはnil
			Geothermal: []float64{}, // make([]float64, 0) と同等
			Hydro:      []float64{},
			Wind:       []float64{},
			Solar:      []float64{},
		}
		assert.Equal(t, expected, result, "空データ時は空のPowerChartDataが返るはずです")
		assert.NoError(t, mock.ExpectationsWereMet(), "モックの期待値が満たされていません")
	})
}
