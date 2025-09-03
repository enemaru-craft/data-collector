package model

import (
	"context"
	"data-manager/custmerr"
	"database/sql"
	"encoding/json"
	"fmt"
)

func (repo *ManagementRepository) CreateSessionIfNotExists(ctx context.Context, tx *sql.Tx, sessionID string) error {
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO
			sessions(session_id, start_time)
		VALUES
			($1, NOW())
		ON CONFLICT
			(session_id)
		DO NOTHING
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, sessionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return &custmerr.LogicalErr{Err: fmt.Errorf("session with ID %s already exists", sessionID)}
		}
		return &custmerr.TechnicalErr{Err: fmt.Errorf("failed to create session: %w", err)}
	}
	return nil
}

func (repo *ManagementRepository) CheckDeviceNotExists(ctx context.Context, tx *sql.Tx, deviceID string) error {
	stmt, err := tx.PrepareContext(ctx, `
		SELECT
			EXISTS
		(SELECT
			1
		FROM
			devices
		WHERE
			device_id = $1)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	var exists bool
	err = stmt.QueryRowContext(ctx, deviceID).Scan(&exists)
	if err != nil {
		return &custmerr.TechnicalErr{Err: fmt.Errorf("failed to check device existence: %w", err)}
	}
	if exists {
		return &custmerr.LogicalErr{Err: fmt.Errorf("device with ID %s already exists", deviceID)}
	}
	return nil
}

func (repo *ManagementRepository) RegisterNewPowerGenerationModule(ctx context.Context, tx *sql.Tx, sessionID, deviceID, deviceType string) error {
	// devices 用の PreparedStatement
	stmtDevice, err := tx.PrepareContext(ctx, `
        INSERT INTO
			devices(device_id, device_type)
        VALUES
			($1, $2)
		ON CONFLICT
			(device_id)
		DO NOTHING
    `)
	if err != nil {
		return &custmerr.TechnicalErr{Err: fmt.Errorf("failed to prepare devices statement: %w", err)}
	}
	defer stmtDevice.Close()

	if _, err := stmtDevice.ExecContext(ctx, deviceID, deviceType); err != nil {
		return &custmerr.TechnicalErr{Err: fmt.Errorf("failed to insert device: %w", err)}
	}

	// session_devices 用の PreparedStatement
	stmtSessionDevice, err := tx.PrepareContext(ctx, `
        INSERT INTO
			session_devices(session_id, device_id)
        VALUES
			($1, $2)
        ON CONFLICT
			(session_id, device_id)
		DO NOTHING
    `)
	if err != nil {
		return &custmerr.TechnicalErr{Err: fmt.Errorf("failed to prepare session_devices statement: %w", err)}
	}
	defer stmtSessionDevice.Close()

	if _, err := stmtSessionDevice.ExecContext(ctx, sessionID, deviceID); err != nil {
		return &custmerr.TechnicalErr{Err: fmt.Errorf("failed to insert session_device: %w", err)}
	}

	return nil
}

func (repo *ManagementRepository) CreateNewWorldIfNotExists(ctx context.Context, tx *sql.Tx, sessionID string) error {
	stmtWorld, err := tx.PrepareContext(ctx, `
        INSERT INTO
			world_state(session_id,timestamp)
        VALUES
			($1, NOW())
    `)
	if err != nil {
		return &custmerr.TechnicalErr{Err: fmt.Errorf("failed to prepare devices statement: %w", err)}
	}
	defer stmtWorld.Close()

	if _, err := stmtWorld.ExecContext(ctx, sessionID); err != nil {
		return &custmerr.TechnicalErr{Err: fmt.Errorf("failed to create new world: %w", err)}
	}

	return nil
}

func (repo *ManagementRepository) TurnOnEquipment(ctx context.Context, tx *sql.Tx, sessionID string, equipment string) (CurrentWorldState, error) {
	// 一旦すべての発電方法の最新発電量を計算する
	getAllPowerStmt, err := tx.PrepareContext(ctx, `
		SELECT DISTINCT ON (d.device_type)
			d.device_type,pl.power
		FROM
			power_logs pl
		JOIN
			session_devices sd
		ON
			pl.session_device_id = sd.id
		JOIN
			devices d
		ON
			sd.device_id = d.device_id
		WHERE
			sd.session_id = $1
		ORDER BY
			d.device_type,pl.timestamp DESC
	`)
	if err != nil {
		return CurrentWorldState{}, &custmerr.TechnicalErr{Err: fmt.Errorf("failed to prepare get all power statement: %w", err)}
	}
	defer getAllPowerStmt.Close()

	rows, err := getAllPowerStmt.QueryContext(ctx, sessionID)
	if err != nil {
		return CurrentWorldState{}, &custmerr.TechnicalErr{Err: fmt.Errorf("failed to query all power: %w", err)}
	}
	defer rows.Close()

	latestPower := make(map[string]float32)
	var allPower float32

	for rows.Next() {
		var deviceType string
		var power float32
		if err := rows.Scan(&deviceType, &power); err != nil {
			return CurrentWorldState{}, &custmerr.TechnicalErr{Err: fmt.Errorf("failed to scan row: %w", err)}
		}
		latestPower[deviceType] = power
		allPower += power
	}

	// 現在の世界の状態を取得
	getWorldStmt, err := tx.PrepareContext(ctx, `
		SELECT
			is_light_enabled,is_train_enabled,is_factory_enabled,is_blackout,villagers_text
		FROM
			world_state
		WHERE
			session_id = $1
		ORDER BY
			timestamp
		DESC
		LIMIT
			1;
	`)
	if err != nil {
		return CurrentWorldState{}, &custmerr.TechnicalErr{Err: fmt.Errorf("failed to prepare get world state statement: %w", err)}
	}
	defer getWorldStmt.Close()

	var isLightEnabled, isTrainEnabled, isFactoryEnabled, isBlackout bool
	var villagersTextBytes []byte

	err = getWorldStmt.QueryRowContext(ctx, sessionID).Scan(&isLightEnabled, &isTrainEnabled, &isFactoryEnabled, &isBlackout, &villagersTextBytes)
	if err != nil {
		return CurrentWorldState{}, &custmerr.TechnicalErr{Err: fmt.Errorf("failed to scan world state: %w", err)}
	}

	var villagersText []string
	if len(villagersTextBytes) > 0 {
		if err := json.Unmarshal(villagersTextBytes, &villagersText); err != nil {
			return CurrentWorldState{}, &custmerr.TechnicalErr{Err: fmt.Errorf("failed to unmarshal villagers_text: %w", err)}
		}
	}

	// 今回ONにする機器に応じて、対応するフラグを更新
	switch equipment {
	case "light":
		isLightEnabled = true
	case "train":
		isTrainEnabled = true
	case "factory":
		isFactoryEnabled = true
	default:
		return CurrentWorldState{}, fmt.Errorf("unknown equipment type: %s", equipment)
	}

	// 新しい状態の消費電力を計算
	var newPowerConsumption float32
	if isLightEnabled {
		newPowerConsumption += 10.0
	}
	if isTrainEnabled {
		newPowerConsumption += 5.0
	}
	if isFactoryEnabled {
		newPowerConsumption += 7.0
	}

	var surplusPower float32
	if newPowerConsumption > allPower {
		isLightEnabled = false
		isTrainEnabled = false
		isFactoryEnabled = false
		isBlackout = true
		surplusPower = 0.0
	} else {
		surplusPower = allPower - newPowerConsumption
		isBlackout = false
	}

	registerNewWorldStateStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO
			world_state(session_id,is_light_enabled,is_train_enabled,
			is_factory_enabled,is_blackout,villagers_text,total_power,surplus_power,timestamp)
		VALUES
			($1,$2,$3,$4,$5,$6,$7,$8,NOW())
	`)
	if err != nil {
		return CurrentWorldState{}, &custmerr.TechnicalErr{Err: fmt.Errorf("failed to prepare register new world state statement: %w", err)}
	}
	defer registerNewWorldStateStmt.Close()

	villagersTextJSON, err := json.Marshal(villagersText)
	if err != nil {
		return CurrentWorldState{}, &custmerr.TechnicalErr{Err: fmt.Errorf("failed to marshal villagers_text: %w", err)}
	}

	_, err = registerNewWorldStateStmt.ExecContext(ctx, sessionID, isLightEnabled, isTrainEnabled, isFactoryEnabled, isBlackout, villagersTextJSON, allPower, surplusPower)
	if err != nil {
		return CurrentWorldState{}, &custmerr.TechnicalErr{Err: fmt.Errorf("failed to insert new world state: %w", err)}
	}

	state := State{
		IsLightEnabled:   isLightEnabled,
		IsTrainEnabled:   isTrainEnabled,
		IsFactoryEnabled: isFactoryEnabled,
		IsBlackout:       isBlackout,
	}

	variables := Variables{
		TotalPower:   allPower,
		SurplusPower: surplusPower,
	}

	returnState := CurrentWorldState{
		State:     state,
		Texts:     villagersText,
		Variables: variables,
	}

	return returnState, nil
}

func (repo *ManagementRepository) TurnOffEquipment(ctx context.Context, tx *sql.Tx, sessionID string, equipment string) (CurrentWorldState, error) {
	// 一旦すべての発電方法の最新発電量を計算する
	getAllPowerStmt, err := tx.PrepareContext(ctx, `
		SELECT DISTINCT ON (d.device_type)
			d.device_type,pl.power
		FROM
			power_logs pl
		JOIN
			session_devices sd
		ON
			pl.session_device_id = sd.id
		JOIN
			devices d
		ON
			sd.device_id = d.device_id
		WHERE
			sd.session_id = $1
		ORDER BY
			d.device_type,pl.timestamp DESC
	`)
	if err != nil {
		return CurrentWorldState{}, &custmerr.TechnicalErr{Err: fmt.Errorf("failed to prepare get all power statement: %w", err)}
	}
	defer getAllPowerStmt.Close()

	rows, err := getAllPowerStmt.QueryContext(ctx, sessionID)
	if err != nil {
		return CurrentWorldState{}, &custmerr.TechnicalErr{Err: fmt.Errorf("failed to query all power: %w", err)}
	}
	defer rows.Close()

	latestPower := make(map[string]float32)
	var allPower float32

	for rows.Next() {
		var deviceType string
		var power float32
		if err := rows.Scan(&deviceType, &power); err != nil {
			return CurrentWorldState{}, &custmerr.TechnicalErr{Err: fmt.Errorf("failed to scan row: %w", err)}
		}
		latestPower[deviceType] = power
		allPower += power
	}

	// 現在の世界の状態を取得
	getWorldStmt, err := tx.PrepareContext(ctx, `
		SELECT
			is_light_enabled,is_train_enabled,is_factory_enabled,is_blackout,villagers_text
		FROM
			world_state
		WHERE
			session_id = $1
		ORDER BY
			timestamp
		DESC
		LIMIT
			1;
	`)
	if err != nil {
		return CurrentWorldState{}, &custmerr.TechnicalErr{Err: fmt.Errorf("failed to prepare get world state statement: %w", err)}
	}
	defer getWorldStmt.Close()

	var isLightEnabled, isTrainEnabled, isFactoryEnabled, isBlackout bool
	var villagersTextBytes []byte

	err = getWorldStmt.QueryRowContext(ctx, sessionID).Scan(&isLightEnabled, &isTrainEnabled, &isFactoryEnabled, &isBlackout, &villagersTextBytes)
	if err != nil {
		return CurrentWorldState{}, &custmerr.TechnicalErr{Err: fmt.Errorf("failed to scan world state: %w", err)}
	}

	var villagersText []string
	if len(villagersTextBytes) > 0 {
		if err := json.Unmarshal(villagersTextBytes, &villagersText); err != nil {
			return CurrentWorldState{}, &custmerr.TechnicalErr{Err: fmt.Errorf("failed to unmarshal villagers_text: %w", err)}
		}
	}

	// オフにする機器に応じてフラグを更新
	switch equipment {
	case "light":
		isLightEnabled = false
	case "train":
		isTrainEnabled = false
	case "factory":
		isFactoryEnabled = false
	default:
		return CurrentWorldState{}, fmt.Errorf("unknown equipment type: %s", equipment)
	}

	var currentPowerConsumption float32
	if isLightEnabled {
		currentPowerConsumption += 10.0
	}
	if isTrainEnabled {
		currentPowerConsumption += 5.0
	}
	if isFactoryEnabled {
		currentPowerConsumption += 7.0
	}

	var surplusPower float32
	if currentPowerConsumption > allPower {
		isBlackout = true
		isLightEnabled = false
		isTrainEnabled = false
		isFactoryEnabled = false
		surplusPower = 0.0
	} else {
		isBlackout = false
		surplusPower = allPower - currentPowerConsumption
	}

	registerNewWorldStateStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO
			world_state(session_id,is_light_enabled,is_train_enabled,
			is_factory_enabled,is_blackout,total_power,surplus_power,villagers_text,timestamp)
		VALUES
			($1,$2,$3,$4,$5,$6,$7,$8,NOW())
	`)
	if err != nil {
		return CurrentWorldState{}, &custmerr.TechnicalErr{Err: fmt.Errorf("failed to prepare register new world state statement: %w", err)}
	}
	defer registerNewWorldStateStmt.Close()

	villagersTextJSON, err := json.Marshal(villagersText)
	if err != nil {
		return CurrentWorldState{}, &custmerr.TechnicalErr{Err: fmt.Errorf("failed to marshal villagers_text: %w", err)}
	}

	_, err = registerNewWorldStateStmt.ExecContext(ctx, sessionID, isLightEnabled, isTrainEnabled, isFactoryEnabled, isBlackout, allPower, surplusPower, villagersTextJSON)
	if err != nil {
		return CurrentWorldState{}, &custmerr.TechnicalErr{Err: fmt.Errorf("failed to insert new world state: %w", err)}
	}

	state := State{
		IsLightEnabled:   isLightEnabled,
		IsTrainEnabled:   isTrainEnabled,
		IsFactoryEnabled: isFactoryEnabled,
		IsBlackout:       isBlackout,
	}

	variables := Variables{
		TotalPower:   allPower,
		SurplusPower: surplusPower,
	}

	returnState := CurrentWorldState{
		State:     state,
		Texts:     villagersText,
		Variables: variables,
	}

	return returnState, nil
}
