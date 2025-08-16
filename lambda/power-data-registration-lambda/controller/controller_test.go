package controller_test

import (
	"context"
	"power-manager/controller"
	"testing"
)

// UnmarshaできないJSONが来たときにエラーを返すこと
func TestRegisterGeothermalPower_InvalidJSON(t *testing.T) {
	//JSONとは違うバイト列を渡すことでテスト
	_, err := controller.RegisterGeothermalPower(context.Background(), []byte("Invalid byte array"))
	if err == nil {
		t.Error("間違ったJSONを渡したのにエラーが帰ってこない")
	}
}

// 必要なフィールドがかけているJSONを渡したときにエラーを返すこと
func TestRegisterGeothermalPower_MissingFieldsJSON(t *testing.T) {
	payload := `{"device_id": "67890", "power": 100.0, "geo_lat": "35.6895", "geo_lon": "139.6917"}`
	_, err := controller.RegisterGeothermalPower(context.Background(), []byte(payload))
	if err == nil {
		t.Error("必要なフィールド(session_id)が欠けているのにエラーが帰ってこない")
	}

	// 必要なフィールドが欠けているJSONを渡す
	payload = `{"session_id": "67890", "power": 100.0, "geo_lat": "35.6895", "geo_lon": "139.6917"}`
	_, err = controller.RegisterGeothermalPower(context.Background(), []byte(payload))
	if err == nil {
		t.Error("必要なフィールド(device_id)が欠けているのにエラーが帰ってこない")
	}

	// 必要なフィールドが欠けているJSONを渡す
	payload = `{"session_id": "67890", "device_id": "67890", "geo_lat": "35.6895", "geo_lon": "139.6917"}`
	_, err = controller.RegisterGeothermalPower(context.Background(), []byte(payload))
	if err == nil {
		t.Error("必要なフィールド(power)が欠けているのにエラーが帰ってこない")
	}

	payload = `{"session_id": "67890", "device_id": "67890", "power": "35.6895", "geo_lon": "139.6917"}`
	_, err = controller.RegisterGeothermalPower(context.Background(), []byte(payload))
	if err == nil {
		t.Error("必要なフィールド(geo_lat)が欠けているのにエラーが帰ってこない")
	}

	payload = `{"session_id": "67890", "device_id": "67890", "power": "35.6895", "geo_lat": "139.6917"}`
	_, err = controller.RegisterGeothermalPower(context.Background(), []byte(payload))
	if err == nil {
		t.Error("必要なフィールド(geo_lon)が欠けているのにエラーが帰ってこない")
	}
}
