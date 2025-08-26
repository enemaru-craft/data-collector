package model

import (
	"context"
	"data-manager/custmerr"
	"database/sql"
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

func (repo *ManagementRepository) TurnOnEquipment(ctx context.Context, tx *sql.Tx, sessionID string, equipment string) error {
	// 機器ごとに処理を書くのは冗長なので､消費電力の変数を用いて計算した結果を世界データに反映する
	var powerConsumption float32

	switch equipment {
	case "light":
		powerConsumption = 10.0
	case "train":
		powerConsumption = 5.0
	case "factory":
		powerConsumption = 7.0
	default:
		return fmt.Errorf("unknown equipment type: %s", equipment)
	}

	//一旦すべての発電方法の最新発電量を計算する

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
		return &custmerr.TechnicalErr{Err: fmt.Errorf("failed to prepare get all power statement: %w", err)}
	}
	defer getAllPowerStmt.Close()

	rows, err := getAllPowerStmt.QueryContext(ctx, sessionID)
	if err != nil {
		return &custmerr.TechnicalErr{Err: fmt.Errorf("failed to query all power: %w", err)}
	}
	defer rows.Close()

	//各発電方式別の現時点の瞬間での発電量
	latestPower := make(map[string]float32)
	// 現時点の瞬間での総発を量を記しておく
	var allPower float32

	for rows.Next() {
		var deviceType string
		var power float32
		if err := rows.Scan(&deviceType, &power); err != nil {
			return &custmerr.TechnicalErr{Err: fmt.Errorf("failed to scan row: %w", err)}
		}
		latestPower[deviceType] = power
		allPower += power
	}

	// 現在の消費電力も考えるために現在の世界の状態を取得
	getWorldStmt, err := tx.PrepareContext(ctx, `
		SELECT
			is_light_enabled,is_train_enabled,is_factory_enabled,is_blackout
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
		return &custmerr.TechnicalErr{Err: fmt.Errorf("failed to prepare get world state statement: %w", err)}
	}
	defer getWorldStmt.Close()

	var isLightEnabled, isTrainEnabled, isFactoryEnabled, isBlackout bool
	err = getWorldStmt.QueryRowContext(ctx, sessionID).Scan(&isLightEnabled, &isTrainEnabled, &isFactoryEnabled, &isBlackout)
	if err != nil {
		return &custmerr.TechnicalErr{Err: fmt.Errorf("failed to scan world state: %w", err)}
	}

	//現在の時点での消費電力
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

	//今回ONにした分の消費電力と現在消費電力を足したのが新しい消費電力
	newPowerConsumption := currentPowerConsumption + powerConsumption

	registerNewWorldStateStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO
			world_state(session_id,is_light_enabled,is_train_enabled,
			is_factory_enabled,is_blackout,total_power,surplus_power,timestamp)
		VALUES
			($1,$2,$3,$4,$5,$6,$7,NOW())
	`)
	if err != nil {
		return &custmerr.TechnicalErr{Err: fmt.Errorf("failed to prepare register new world state statement: %w", err)}
	}

	switch equipment {
	case "light":
		isLightEnabled = true
	case "train":
		isTrainEnabled = true
	case "factory":
		isFactoryEnabled = true
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
	}

	_, err = registerNewWorldStateStmt.ExecContext(ctx, sessionID, isLightEnabled, isTrainEnabled, isFactoryEnabled, isBlackout, allPower, surplusPower)
	if err != nil {
		return &custmerr.TechnicalErr{Err: fmt.Errorf("failed to insert new world state: %w", err)}
	}

	return nil
}
