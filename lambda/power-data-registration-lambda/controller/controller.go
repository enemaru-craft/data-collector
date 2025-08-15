package controller

import (
	"context"
	"encoding/json"
	"errors"
	"power-manager/custmerr"
	"power-manager/model"
)

type Payload struct {
	SessionID string  `json:"session_id"`
	DeviceID  string  `json:"device_id"`
	Power     float32 `json:"power"`
	GeoLat    string  `json:"geo_lat"`
	GeoLon    string  `json:"geo_lon"`
}

func RegisterGeothermalPower(ctx context.Context, event json.RawMessage) (string, error) {
	var payload Payload
	if err := json.Unmarshal(event, &payload); err != nil {
		return "Failed to parse payload", err
	}

	if payload.SessionID == "" || payload.DeviceID == "" || payload.Power <= 0 {
		return "Invalid payload: missing required fields", nil
	}

	conn := model.GetConn()
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return "Failed to begin transaction: " + err.Error(), err
	}
	defer tx.Rollback()

	err = model.RegisterNewPowerLog(ctx, tx, payload.SessionID, payload.DeviceID, payload.GeoLat, payload.GeoLon, payload.Power)
	if err != nil {
		tx.Rollback()
		var lErr *custmerr.LogicalErr
		var tErr *custmerr.TechnicalErr
		switch {
		case errors.As(err, &lErr):
			return "Session or device not found: " + err.Error(), nil
		case errors.As(err, &tErr):
			return "Technical error occurred: " + err.Error(), nil
		}
	}

	tx.Commit()

	return "Failed to register geothermal power data: " + err.Error(), err
}

func RegisterSolarPower(ctx context.Context, event json.RawMessage) (string, error) {
	var payload Payload
	if err := json.Unmarshal(event, &payload); err != nil {
		return "Failed to parse payload", err
	}

	if payload.SessionID == "" || payload.DeviceID == "" || payload.Power <= 0 {
		return "Invalid payload: missing required fields", nil
	}

	conn := model.GetConn()
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return "Failed to begin transaction: " + err.Error(), err
	}
	defer tx.Rollback()

	err = model.RegisterNewPowerLog(ctx, tx, payload.SessionID, payload.DeviceID, payload.GeoLat, payload.GeoLon, payload.Power)
	if err != nil {
		tx.Rollback()
		var lErr *custmerr.LogicalErr
		var tErr *custmerr.TechnicalErr
		switch {
		case errors.As(err, &lErr):
			return "Session or device not found: " + err.Error(), nil
		case errors.As(err, &tErr):
			return "Technical error occurred: " + err.Error(), nil
		}
	}

	tx.Commit()

	return "Failed to register geothermal power data: " + err.Error(), err
}

func RegisterWindPower(ctx context.Context, event json.RawMessage) (string, error) {
	var payload Payload
	if err := json.Unmarshal(event, &payload); err != nil {
		return "Failed to parse payload", err
	}

	if payload.SessionID == "" || payload.DeviceID == "" || payload.Power <= 0 {
		return "Invalid payload: missing required fields", nil
	}

	conn := model.GetConn()
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return "Failed to begin transaction: " + err.Error(), err
	}
	defer tx.Rollback()

	err = model.RegisterNewPowerLog(ctx, tx, payload.SessionID, payload.DeviceID, payload.GeoLat, payload.GeoLon, payload.Power)
	if err != nil {
		tx.Rollback()
		var lErr *custmerr.LogicalErr
		var tErr *custmerr.TechnicalErr
		switch {
		case errors.As(err, &lErr):
			return "Session or device not found: " + err.Error(), nil
		case errors.As(err, &tErr):
			return "Technical error occurred: " + err.Error(), nil
		}
	}

	tx.Commit()

	return "Failed to register geothermal power data: " + err.Error(), err
}

func RegisterHydrogenPower(ctx context.Context, event json.RawMessage) (string, error) {
	var payload Payload
	if err := json.Unmarshal(event, &payload); err != nil {
		return "Failed to parse payload", err
	}

	if payload.SessionID == "" || payload.DeviceID == "" || payload.Power <= 0 {
		return "Invalid payload: missing required fields", nil
	}

	conn := model.GetConn()
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return "Failed to begin transaction: " + err.Error(), err
	}
	defer tx.Rollback()

	err = model.RegisterNewPowerLog(ctx, tx, payload.SessionID, payload.DeviceID, payload.GeoLat, payload.GeoLon, payload.Power)
	if err != nil {
		tx.Rollback()
		var lErr *custmerr.LogicalErr
		var tErr *custmerr.TechnicalErr
		switch {
		case errors.As(err, &lErr):
			return "Session or device not found: " + err.Error(), nil
		case errors.As(err, &tErr):
			return "Technical error occurred: " + err.Error(), nil
		}
	}

	tx.Commit()

	return "Failed to register geothermal power data: " + err.Error(), err
}

func RegisterHandCrankPower(ctx context.Context, event json.RawMessage) (string, error) {
	var payload Payload
	if err := json.Unmarshal(event, &payload); err != nil {
		return "Failed to parse payload", err
	}

	if payload.SessionID == "" || payload.DeviceID == "" || payload.Power <= 0 {
		return "Invalid payload: missing required fields", nil
	}

	conn := model.GetConn()
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return "Failed to begin transaction: " + err.Error(), err
	}
	defer tx.Rollback()

	err = model.RegisterNewPowerLog(ctx, tx, payload.SessionID, payload.DeviceID, payload.GeoLat, payload.GeoLon, payload.Power)
	if err != nil {
		tx.Rollback()
		var lErr *custmerr.LogicalErr
		var tErr *custmerr.TechnicalErr
		switch {
		case errors.As(err, &lErr):
			return "Session or device not found: " + err.Error(), nil
		case errors.As(err, &tErr):
			return "Technical error occurred: " + err.Error(), nil
		}
	}

	tx.Commit()

	return "Failed to register geothermal power data: " + err.Error(), err
}
