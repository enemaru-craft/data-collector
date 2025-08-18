package controller_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"power-manager/controller"
	"power-manager/model"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

type mockRepo struct{}

func (m *mockRepo) RegisterNewPowerLog(ctx context.Context, tx *sql.Tx, sessionID, deviceID, geoLat, geoLon string, power float32) error {
	return nil
}
func (m *mockRepo) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return nil, nil
}

// 間違ったJSONや必要なフィールドが欠けている場合のテスト
func TestRegisterPower_InvalidAndMissingFields(t *testing.T) {
	ctr := controller.NewLogController(&mockRepo{})

	tests := []struct {
		name    string
		method  func(ctx context.Context, event json.RawMessage) (string, error)
		payload string
		wantErr bool
	}{
		// ---------------- Invalid JSON ----------------
		{"Geothermal_InvalidJSON", ctr.RegisterGeothermalPower, "Invalid byte array", true},
		{"Solar_InvalidJSON", ctr.RegisterSolarPower, "Invalid byte array", true},
		{"Wind_InvalidJSON", ctr.RegisterWindPower, "Invalid byte array", true},
		{"Hydrogen_InvalidJSON", ctr.RegisterHydrogenPower, "Invalid byte array", true},
		{"HandCrank_InvalidJSON", ctr.RegisterHandCrankPower, "Invalid byte array", true},

		// ---------------- Geothermal Missing Fields ----------------
		{"Geothermal_MissingSession", ctr.RegisterGeothermalPower, `{"device_id": "67890", "power": 100.0, "geo_lat": "35.6", "geo_lon": "139.7"}`, true},
		{"Geothermal_MissingDevice", ctr.RegisterGeothermalPower, `{"session_id": "abc", "power": 100.0, "geo_lat": "35.6", "geo_lon": "139.7"}`, true},
		{"Geothermal_MissingPower", ctr.RegisterGeothermalPower, `{"session_id": "abc", "device_id": "67890", "geo_lat": "35.6", "geo_lon": "139.7"}`, true},

		// ---------------- Solar Missing Fields ----------------
		{"Solar_MissingSession", ctr.RegisterSolarPower, `{"device_id": "67890", "power": 200.0, "geo_lat": "35.6", "geo_lon": "139.7"}`, true},
		{"Solar_MissingDevice", ctr.RegisterSolarPower, `{"session_id": "abc", "power": 200.0, "geo_lat": "35.6", "geo_lon": "139.7"}`, true},
		{"Solar_MissingPower", ctr.RegisterSolarPower, `{"session_id": "abc", "device_id": "67890", "geo_lat": "35.6", "geo_lon": "139.7"}`, true},

		// ---------------- Wind Missing Fields ----------------
		{"Wind_MissingSession", ctr.RegisterWindPower, `{"device_id": "67890", "power": 300.0, "geo_lat": "35.6", "geo_lon": "139.7"}`, true},
		{"Wind_MissingDevice", ctr.RegisterWindPower, `{"session_id": "abc", "power": 300.0, "geo_lat": "35.6", "geo_lon": "139.7"}`, true},
		{"Wind_MissingPower", ctr.RegisterWindPower, `{"session_id": "abc", "device_id": "67890", "geo_lat": "35.6", "geo_lon": "139.7"}`, true},

		// ---------------- Hydrogen Missing Fields ----------------
		{"Hydrogen_MissingSession", ctr.RegisterHydrogenPower, `{"device_id": "67890", "power": 400.0, "geo_lat": "35.6", "geo_lon": "139.7"}`, true},
		{"Hydrogen_MissingDevice", ctr.RegisterHydrogenPower, `{"session_id": "abc", "power": 400.0, "geo_lat": "35.6", "geo_lon": "139.7"}`, true},
		{"Hydrogen_MissingPower", ctr.RegisterHydrogenPower, `{"session_id": "abc", "device_id": "67890", "geo_lat": "35.6", "geo_lon": "139.7"}`, true},

		// ---------------- HandCrank Missing Fields ----------------
		{"HandCrank_MissingSession", ctr.RegisterHandCrankPower, `{"device_id": "67890", "power": 10.0, "geo_lat": "35.6", "geo_lon": "139.7"}`, true},
		{"HandCrank_MissingDevice", ctr.RegisterHandCrankPower, `{"session_id": "abc", "power": 10.0, "geo_lat": "35.6", "geo_lon": "139.7"}`, true},
		{"HandCrank_MissingPower", ctr.RegisterHandCrankPower, `{"session_id": "abc", "device_id": "67890", "geo_lat": "35.6", "geo_lon": "139.7"}`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.method(context.Background(), []byte(tt.payload))
			if (err != nil) != tt.wantErr {
				t.Errorf("%s: got error = %v, wantErr = %v", tt.name, err, tt.wantErr)
			}
		})
	}
}

// 適切に書き込め場合の返り値が正しいかテストする
func TestRegisterAllPowers_WithSQLMock(t *testing.T) {
	tests := []struct {
		name       string
		methodName string
		deviceID   string
		deviceType string
		power      float32
	}{
		{"Geothermal", "RegisterGeothermalPower", "devGeo", "geothermal", 123.4},
		{"Solar", "RegisterSolarPower", "devSol", "solar", 200.0},
		{"Wind", "RegisterWindPower", "devWind", "wind", 300.0},
		{"Hydrogen", "RegisterHydrogenPower", "devHyd", "hydrogen", 400.0},
		{"HandCrank", "RegisterHandCrankPower", "devHand", "handcrank", 10.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("failed to create sqlmock: %v", err)
			}
			defer db.Close()

			// --- SQLMock の期待値 ---
			mock.ExpectBegin()

			mock.ExpectPrepare("SELECT id FROM session_devices WHERE session_id = \\$1 AND device_id = \\$2").
				ExpectQuery().
				WithArgs("abc123", tt.deviceID).
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

			mock.ExpectPrepare("SELECT device_type FROM devices WHERE device_id = \\$1").
				ExpectQuery().
				WithArgs(tt.deviceID).
				WillReturnRows(sqlmock.NewRows([]string{"device_type"}).AddRow(tt.deviceType))

			mock.ExpectPrepare("INSERT INTO power_logs.*").
				ExpectExec().
				WithArgs(1, tt.power, "35.6895", "139.6917", tt.deviceType).
				WillReturnResult(sqlmock.NewResult(1, 1))

			mock.ExpectCommit()

			// --- Controller 初期化 ---
			repo := model.NewLogRepository(db)
			ctr := controller.NewLogController(repo)

			payload := map[string]interface{}{
				"session_id": "abc123",
				"device_id":  tt.deviceID,
				"power":      tt.power,
				"geo_lat":    "35.6895",
				"geo_lon":    "139.6917",
			}
			data, _ := json.Marshal(payload)

			// --- メソッド呼び出し ---
			var resp string
			switch tt.methodName {
			case "RegisterGeothermalPower":
				resp, err = ctr.RegisterGeothermalPower(context.Background(), data)
			case "RegisterSolarPower":
				resp, err = ctr.RegisterSolarPower(context.Background(), data)
			case "RegisterWindPower":
				resp, err = ctr.RegisterWindPower(context.Background(), data)
			case "RegisterHydrogenPower":
				resp, err = ctr.RegisterHydrogenPower(context.Background(), data)
			case "RegisterHandCrankPower":
				resp, err = ctr.RegisterHandCrankPower(context.Background(), data)
			default:
				t.Fatalf("unknown method: %s", tt.methodName)
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp == "" {
				t.Fatalf("expected non-empty response")
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %v", err)
			}
		})
	}
}
