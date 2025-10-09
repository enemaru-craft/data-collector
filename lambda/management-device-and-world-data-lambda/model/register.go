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
			is_light_enabled,is_train_enabled,is_factory_enabled,is_blackout,is_house_enabled,is_facility_enabled,villagers_text,blackout_count
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

	var isLightEnabled, isTrainEnabled, isFactoryEnabled, isBlackout, isHouseEnabled, isFacilityEnabled bool
	var villagersTextBytes []byte
	var blackoutCount int

	err = getWorldStmt.QueryRowContext(ctx, sessionID).Scan(&isLightEnabled, &isTrainEnabled, &isFactoryEnabled, &isBlackout, &isHouseEnabled, &isFacilityEnabled, &villagersTextBytes, &blackoutCount)
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
	case "house":
		isHouseEnabled = true
	case "facility":
		isFacilityEnabled = true
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
	if isHouseEnabled {
		newPowerConsumption += 300.0
	}
	if isFacilityEnabled {
		newPowerConsumption += 1015.0
	}

	// 以前のblackout状態を保存
	previousBlackoutState := isBlackout

	var surplusPower float32
	if newPowerConsumption > allPower {
		isLightEnabled = false
		isTrainEnabled = false
		isFactoryEnabled = false
		isHouseEnabled = false
		isFacilityEnabled = false
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
			world_state(session_id,is_light_enabled,is_train_enabled,
			is_factory_enabled,is_blackout,is_house_enabled,is_facility_enabled,total_power,surplus_power,villagers_text,blackout_count,timestamp)
		VALUES
			($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,NOW())
	`)
	if err != nil {
		return CurrentWorldState{}, &custmerr.TechnicalErr{Err: fmt.Errorf("failed to prepare register new world state statement: %w", err)}
	}
	defer registerNewWorldStateStmt.Close()

	// APIレスポンス用にはgenerateVillagersTextsを使用し、DBには空配列を保存
	emptyTexts := []string{}
	villagersTextJSON, err := json.Marshal(emptyTexts)
	if err != nil {
		return CurrentWorldState{}, &custmerr.TechnicalErr{Err: fmt.Errorf("failed to marshal villagers_text: %w", err)}
	}

	_, err = registerNewWorldStateStmt.ExecContext(ctx, sessionID, isLightEnabled, isTrainEnabled, isFactoryEnabled, isBlackout, isHouseEnabled, isFacilityEnabled, allPower, surplusPower, villagersTextJSON, blackoutCount)
	if err != nil {
		return CurrentWorldState{}, &custmerr.TechnicalErr{Err: fmt.Errorf("failed to insert new world state: %w", err)}
	}

	state := State{
		IsLightEnabled:    isLightEnabled,
		IsTrainEnabled:    isTrainEnabled,
		IsFactoryEnabled:  isFactoryEnabled,
		IsHouseEnabled:    isHouseEnabled,
		IsFacilityEnabled: isFacilityEnabled,
		IsBlackout:        isBlackout,
	}

	variables := Variables{
		TotalPower:   allPower,
		SurplusPower: surplusPower,
	}

	// fire発電量を取得
	var firePower float32 = 0
	if latestPower["fire"] > 0 {
		firePower = latestPower["fire"]
	}

	// 村人のテキストを生成
	villagersTexts := generateVillagersTexts(isLightEnabled, isTrainEnabled, isFactoryEnabled, isHouseEnabled, isFacilityEnabled, firePower)

	returnState := CurrentWorldState{
		State:     state,
		Texts:     villagersTexts,
		Variables: variables,
	}

	return returnState, nil
}

func (repo *ManagementRepository) TurnOffEquipment(ctx context.Context, tx *sql.Tx, sessionID string, equipment string) (CurrentWorldState, error) {
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
			is_light_enabled,is_train_enabled,is_factory_enabled,is_house_enabled,is_facility_enabled,is_blackout,villagers_text,blackout_count
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

	var isLightEnabled, isTrainEnabled, isFactoryEnabled, isBlackout, isHouseEnabled, isFacilityEnabled bool
	var villagersTextBytes []byte
	var blackoutCount int

	err = getWorldStmt.QueryRowContext(ctx, sessionID).Scan(&isLightEnabled, &isTrainEnabled, &isFactoryEnabled, &isHouseEnabled, &isFacilityEnabled, &isBlackout, &villagersTextBytes, &blackoutCount)
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
	case "house":
		isHouseEnabled = false
	case "facility":
		isFacilityEnabled = false
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
	if isHouseEnabled {
		currentPowerConsumption += 300.0
	}
	if isFacilityEnabled {
		currentPowerConsumption += 1015.0
	}

	// 以前のblackout状態を保存
	previousBlackoutState := isBlackout

	var surplusPower float32
	if currentPowerConsumption > allPower {
		isBlackout = true
		isLightEnabled = false
		isTrainEnabled = false
		isFactoryEnabled = false
		isHouseEnabled = false
		isFacilityEnabled = false
		surplusPower = 0.0
	} else {
		isBlackout = false
		surplusPower = allPower - currentPowerConsumption
	}

	// blackout状態が変化した場合（停電が発生した場合）にカウントを増加
	if !previousBlackoutState && isBlackout {
		blackoutCount++
	}

	registerNewWorldStateStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO
			world_state(session_id,is_light_enabled,is_train_enabled,
			is_factory_enabled,is_blackout,is_house_enabled,is_facility_enabled,total_power,surplus_power,villagers_text,blackout_count,timestamp)
		VALUES
			($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,NOW())
	`)
	if err != nil {
		return CurrentWorldState{}, &custmerr.TechnicalErr{Err: fmt.Errorf("failed to prepare register new world state statement: %w", err)}
	}
	defer registerNewWorldStateStmt.Close()

	// APIレスポンス用にはgenerateVillagersTextsを使用し、DBには空配列を保存
	emptyTexts := []string{}
	villagersTextJSON, err := json.Marshal(emptyTexts)
	if err != nil {
		return CurrentWorldState{}, &custmerr.TechnicalErr{Err: fmt.Errorf("failed to marshal villagers_text: %w", err)}
	}

	_, err = registerNewWorldStateStmt.ExecContext(ctx, sessionID, isLightEnabled, isTrainEnabled, isFactoryEnabled, isBlackout, isHouseEnabled, isFacilityEnabled, allPower, surplusPower, villagersTextJSON, blackoutCount)
	if err != nil {
		return CurrentWorldState{}, &custmerr.TechnicalErr{Err: fmt.Errorf("failed to insert new world state: %w", err)}
	}

	state := State{
		IsLightEnabled:    isLightEnabled,
		IsTrainEnabled:    isTrainEnabled,
		IsFactoryEnabled:  isFactoryEnabled,
		IsHouseEnabled:    isHouseEnabled,
		IsFacilityEnabled: isFacilityEnabled,
		IsBlackout:        isBlackout,
	}

	variables := Variables{
		TotalPower:   allPower,
		SurplusPower: surplusPower,
	}

	// fire発電量を取得
	var firePower float32 = 0
	if latestPower["fire"] > 0 {
		firePower = latestPower["fire"]
	}

	// 村人のテキストを生成
	villagersTexts := generateVillagersTexts(isLightEnabled, isTrainEnabled, isFactoryEnabled, isHouseEnabled, isFacilityEnabled, firePower)

	returnState := CurrentWorldState{
		State:     state,
		Texts:     villagersTexts,
		Variables: variables,
	}

	return returnState, nil
}
