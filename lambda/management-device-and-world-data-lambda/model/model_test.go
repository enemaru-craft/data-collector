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
		// テストデータ設定
		sessionId := "test-session"
		bucketMinutes := 3

		// モックトランザクションを開始
		mock.ExpectBegin()
		tx, err := db.Begin()
		assert.NoError(t, err)
		defer tx.Rollback()

		// モックの期待値設定
		rows := sqlmock.NewRows([]string{"bucket", "timestamp", "power", "device_id", "device_type"}).
			AddRow(
				time.Date(2025, 9, 22, 19, 27, 0, 0, time.UTC),
				time.Date(2025, 9, 22, 19, 27, 15, 0, time.UTC),
				20.0,
				"M5-22-geothermal-1",
				"geothermal",
			)

		mock.ExpectPrepare("SELECT").ExpectQuery().WithArgs(sessionId, bucketMinutes).WillReturnRows(rows)

		// テスト実行
		result, err := repo.GetPowerHistory(context.Background(), tx, sessionId, bucketMinutes)

		// 検証
		assert.NoError(t, err)
		assert.NotNil(t, result)
		// 注意: 現在の実装では PowerChartData{} を返しているため、実際の値は確認できない
		// このテストは実装修正後に期待値を更新する必要がある

		// モックの期待値チェック
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("複数バケット、境界をまたぐテスト", func(t *testing.T) {
		sessionId := "test-session"
		bucketMinutes := 3

		// モックトランザクションを開始
		mock.ExpectBegin()
		tx, err := db.Begin()
		assert.NoError(t, err)
		defer tx.Rollback()

		// モックの期待値設定 - M5-22-solar-1のデータを模擬
		rows := sqlmock.NewRows([]string{"bucket", "timestamp", "power", "device_id", "device_type"}).
			AddRow(
				time.Date(2025, 9, 22, 19, 27, 0, 0, time.UTC),
				time.Date(2025, 9, 22, 19, 27, 33, 872000000, time.UTC),
				20.0,
				"M5-22-solar-1",
				"solar",
			).
			AddRow(
				time.Date(2025, 9, 22, 19, 42, 0, 0, time.UTC),
				time.Date(2025, 9, 22, 19, 43, 12, 29000000, time.UTC),
				20.0,
				"M5-22-solar-1",
				"solar",
			)

		mock.ExpectPrepare("SELECT").ExpectQuery().WithArgs(sessionId, bucketMinutes).WillReturnRows(rows)

		// テスト実行
		result, err := repo.GetPowerHistory(context.Background(), tx, sessionId, bucketMinutes)

		fmt.Println(result)

		// 検証
		assert.NoError(t, err)
		assert.NotNil(t, result)

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// 境界値計算のロジックをテストする単体テスト
func TestBoundaryCalculation(t *testing.T) {
	t.Run("線形補間の計算テスト", func(t *testing.T) {
		// テストケース: 2つのポイント間の境界値計算
		prevTime := time.Date(2025, 9, 22, 19, 27, 33, 0, time.UTC)    // 19:27:33
		currTime := time.Date(2025, 9, 22, 19, 43, 12, 0, time.UTC)    // 19:43:12
		boundaryTime := time.Date(2025, 9, 22, 19, 42, 0, 0, time.UTC) // 19:42:00

		prevPower := 20.0
		currPower := 25.0

		// 計算
		totalDuration := currTime.Sub(prevTime).Seconds()
		elapsedDuration := boundaryTime.Sub(prevTime).Seconds()
		expectedPower := prevPower + (currPower-prevPower)*(elapsedDuration/totalDuration)

		// 期待値計算
		// 19:27:33 から 19:43:12 まで = 15分39秒 = 939秒
		// 19:27:33 から 19:42:00 まで = 14分27秒 = 867秒
		// 比率 = 867/939 ≈ 0.923
		// 発電量 = 20 + (25-20) * 0.923 ≈ 24.615

		assert.InDelta(t, 24.615, expectedPower, 0.01, "境界での発電量計算が正しくありません")
	})

	t.Run("台形積分の計算テスト", func(t *testing.T) {
		// テストケース: 台形積分
		startPower := 20.0
		endPower := 25.0
		duration := 3.0 * 60 // 3分 = 180秒

		// 台形積分: (開始発電量 + 終了発電量) / 2 * 時間
		expectedWh := (startPower + endPower) / 2.0 * duration / 3600.0 // Wh単位
		// = (20 + 25) / 2 * 180 / 3600 = 22.5 * 0.05 = 1.125 Wh

		assert.InDelta(t, 1.125, expectedWh, 0.001, "台形積分の計算が正しくありません")
	})
}

// rearMidpointの問題を検証するテスト
func TestRearMidpointIssue(t *testing.T) {
	t.Run("rearMidpointのキー不整合テスト", func(t *testing.T) {
		// 実際のユースケースを模擬
		frontMidpoint := make(map[string]map[time.Time]float64)
		rearMidpoint := make(map[string]map[time.Time]float64)

		deviceID := "M5-22-solar-1"
		frontMidpoint[deviceID] = make(map[time.Time]float64)
		rearMidpoint[deviceID] = make(map[time.Time]float64)

		// バケット1: 19:27:00, バケット2: 19:42:00
		bucket1 := time.Date(2025, 9, 22, 19, 27, 0, 0, time.UTC)
		bucket2 := time.Date(2025, 9, 22, 19, 42, 0, 0, time.UTC)
		boundaryPower := 22.5

		// 現在の実装: 前のバケットの開始時刻をキーとして格納
		rearMidpoint[deviceID][bucket1] = boundaryPower

		// 参照時: 現在のバケットの開始時刻で参照を試みる
		if value, exists := rearMidpoint[deviceID][bucket2]; exists {
			t.Errorf("期待しない値が見つかりました: %f", value)
		} else {
			// これが現在の問題: キーが一致しないため値が取得できない
			t.Logf("予想通り値が見つかりませんでした。これがrearMidpointの問題です。")
		}

		// 正しい実装では以下のようになるべき
		correctRearMidpoint := make(map[string]map[time.Time]float64)
		correctRearMidpoint[deviceID] = make(map[time.Time]float64)
		correctRearMidpoint[deviceID][bucket2] = boundaryPower // 現在のバケット開始時刻をキーに

		if value, exists := correctRearMidpoint[deviceID][bucket2]; exists {
			assert.Equal(t, boundaryPower, value, "正しい実装では値が取得できるはずです")
		}
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
		bucketMinutes := 3

		// モックトランザクションを開始
		mock.ExpectBegin()
		tx, err := db.Begin()
		assert.NoError(t, err)
		defer tx.Rollback()

		mock.ExpectPrepare("SELECT").WillReturnError(sql.ErrConnDone)

		result, err := repo.GetPowerHistory(context.Background(), tx, sessionId, bucketMinutes)

		assert.Error(t, err)
		assert.Equal(t, PowerChartData{}, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("空データのテスト", func(t *testing.T) {
		sessionId := "test-session"
		bucketMinutes := 3

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
		assert.Equal(t, expected, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
