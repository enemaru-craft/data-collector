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

func (repo *ManagementRepository) SetEquipmentPercent(ctx context.Context, tx *sql.Tx, sessionID string, equipment string, percent int) (CurrentWorldState, error) {
	// DynamoDBから最新の発電量を取得
	latestPower := make(map[string]float32)
	var allPower float32

	// 各デバイスタイプについてDynamoDBから最新データを取得
	deviceTypes := []string{"solar", "geothermal", "hydrogen", "wind", "fire"}
	for _, deviceType := range deviceTypes {
		response, err := repo.GetMultipleDevicesPowerDataFromDynamoDB(ctx, deviceType, sessionID)
		if err != nil {
			return CurrentWorldState{}, &custmerr.TechnicalErr{Err: fmt.Errorf("failed to get power data from DynamoDB: %w", err)}
		}
		if response.TotalPower > 0 {
			latestPower[deviceType] = response.TotalPower
			allPower += response.TotalPower
		}
	}

	// 現在の世界の状態を取得
	getWorldStmt, err := tx.PrepareContext(ctx, `
		SELECT
			house_lit_percent,facility_lit_percent,light_lit_percent,factory_lit_percent,is_train_enabled,is_blackout,villagers_text,blackout_count
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

	var houseLitPercent, facilityLitPercent, lightLitPercent, factoryLitPercent int
	var isTrainEnabled, isBlackout bool
	var villagersTextBytes []byte
	var blackoutCount int

	err = getWorldStmt.QueryRowContext(ctx, sessionID).Scan(&houseLitPercent, &facilityLitPercent, &lightLitPercent, &factoryLitPercent, &isTrainEnabled, &isBlackout, &villagersTextBytes, &blackoutCount)
	if err != nil {
		return CurrentWorldState{}, &custmerr.TechnicalErr{Err: fmt.Errorf("failed to scan world state: %w", err)}
	}

	// 指定された機器のパーセントを更新
	switch equipment {
	case "house":
		houseLitPercent = percent
	case "facility":
		facilityLitPercent = percent
	case "light":
		lightLitPercent = percent
	case "factory":
		factoryLitPercent = percent
	case "train":
		if percent == 0 {
			isTrainEnabled = false
		} else {
			isTrainEnabled = true
		}
	default:
		return CurrentWorldState{}, fmt.Errorf("unknown equipment type: %s", equipment)
	}

	// 新しい状態の消費電力を計算
	var newPowerConsumption float32
	newPowerConsumption += 5.0 * float32(lightLitPercent) / 100.0
	if isTrainEnabled {
		newPowerConsumption += 410.0
	}
	newPowerConsumption += 300.0 * float32(factoryLitPercent) / 100.0
	newPowerConsumption += 300.0 * float32(houseLitPercent) / 100.0
	newPowerConsumption += 1015.0 * float32(facilityLitPercent) / 100.0

	// 以前のblackout状態を保存
	previousBlackoutState := isBlackout

	var surplusPower float32
	if newPowerConsumption > allPower {
		houseLitPercent = 0
		facilityLitPercent = 0
		lightLitPercent = 0
		factoryLitPercent = 0
		isTrainEnabled = false
		isBlackout = true
		surplusPower = 0.0
	} else {
		surplusPower = allPower - newPowerConsumption
		isBlackout = false
	}

	// blackout状態が変化した場合（停電が発生した場合）にカウントを増加
	if !previousBlackoutState && isBlackout {
		blackoutCount++
	}

	registerNewWorldStateStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO
			world_state(session_id,house_lit_percent,facility_lit_percent,
			light_lit_percent,factory_lit_percent,is_train_enabled,is_blackout,total_power,surplus_power,villagers_text,blackout_count,timestamp)
		VALUES
			($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,NOW())
	`)
	if err != nil {
		return CurrentWorldState{}, &custmerr.TechnicalErr{Err: fmt.Errorf("failed to prepare register new world state statement: %w", err)}
	}
	defer registerNewWorldStateStmt.Close()

	// APIレスポンス用にはDBには空配列を保存
	emptyTexts := []string{}
	villagersTextJSON, err := json.Marshal(emptyTexts)
	if err != nil {
		return CurrentWorldState{}, &custmerr.TechnicalErr{Err: fmt.Errorf("failed to marshal villagers_text: %w", err)}
	}

	_, err = registerNewWorldStateStmt.ExecContext(ctx, sessionID, houseLitPercent, facilityLitPercent, lightLitPercent, factoryLitPercent, isTrainEnabled, isBlackout, allPower, surplusPower, villagersTextJSON, blackoutCount)
	if err != nil {
		return CurrentWorldState{}, &custmerr.TechnicalErr{Err: fmt.Errorf("failed to insert new world state: %w", err)}
	}

	state := State{
		HouseLitPercent:    houseLitPercent,
		FacilityLitPercent: facilityLitPercent,
		LightLitPercent:    lightLitPercent,
		FactoryLitPercent:  factoryLitPercent,
		IsTrainEnabled:     isTrainEnabled,
		IsBlackout:         isBlackout,
	}

	variables := Variables{
		TotalPower:   allPower,
		SurplusPower: surplusPower,
	}

	returnState := CurrentWorldState{
		State:     state,
		Texts:     []string{},
		Variables: variables,
	}

	return returnState, nil
}
