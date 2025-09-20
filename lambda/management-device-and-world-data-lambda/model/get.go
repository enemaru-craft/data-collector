package model

import (
	"context"
	"data-manager/custmerr"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go/aws"
)

func (repo *ManagementRepository) GetLatestPowerData(ctx context.Context, tx *sql.Tx, deviceType string, sessionId string) (float32, string, string, error) {
	stmt, err := tx.PrepareContext(ctx,
		`
		SELECT
			pl.power,pl.gps_lat,pl.gps_lon
		FROM
			power_logs pl
		JOIN
			session_devices sd ON pl.session_device_id = sd.id
		JOIN
			devices d ON sd.device_id = d.device_id
		WHERE
			sd.session_id = $1
			AND d.device_type = $2
		ORDER BY
			pl.timestamp DESC
		LIMIT 1;

		`)
	if err != nil {
		return 0, "", "", &custmerr.TechnicalErr{Err: err}
	}
	defer stmt.Close()

	var latestPower float32
	var gpsLat, gpsLon string
	err = stmt.QueryRowContext(ctx, sessionId, deviceType).Scan(&latestPower, &gpsLat, &gpsLon)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, "", "", &custmerr.LogicalErr{Err: fmt.Errorf("no power data found for device type %s", deviceType)}
		}
		return 0, "", "", &custmerr.TechnicalErr{Err: fmt.Errorf("failed to get latest power data: %w", err)}
	}

	return latestPower, gpsLat, gpsLon, nil
}

func (repo *ManagementRepository) GetMultipleDevicesPowerDataFromDynamoDB(ctx context.Context, deviceType string, sessionId string) (MultipleDevicePowerResponse, error) {
	applicableDevices := fmt.Sprintf("M5-%s-%s", sessionId, deviceType)

	input := &dynamodb.QueryInput{
		TableName:              aws.String("tmp_table"),
		KeyConditionExpression: aws.String("session_id = :sid AND begins_with(device_id,:dtype)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":sid":   &types.AttributeValueMemberS{Value: sessionId},
			":dtype": &types.AttributeValueMemberS{Value: applicableDevices},
		},
	}

	result, err := repo.dc.Query(ctx, input)
	if err != nil {
		return MultipleDevicePowerResponse{}, &custmerr.TechnicalErr{Err: fmt.Errorf("failed to query dynamodb: %w", err)}
	}

	var devices []DeviceDetail
	totalPower := 0.0

	for _, item := range result.Items {
		powerStr := item["power"].(*types.AttributeValueMemberN).Value
		deviceId := item["device_id"].(*types.AttributeValueMemberS).Value
		gpsLat := item["gps_lat"].(*types.AttributeValueMemberS).Value
		gpsLon := item["gps_lon"].(*types.AttributeValueMemberS).Value

		power, err := strconv.Atoi(powerStr)
		if err != nil {
			return MultipleDevicePowerResponse{}, &custmerr.TechnicalErr{Err: fmt.Errorf("failed to convert power to int: %w", err)}
		}

		devices = append(devices, DeviceDetail{
			DeviceID:   deviceId,
			DeviceType: deviceType,
			Power:      float32(power),
			GpsLat:     gpsLat,
			GpsLon:     gpsLon,
		})

		totalPower += float64(power)
	}

	return MultipleDevicePowerResponse{
		TotalPower: float32(totalPower),
		Devices:    devices,
	}, nil
}

type CurrentWorldState struct {
	State     State     `json:"state"`
	Texts     []string  `json:"texts"`
	Variables Variables `json:"variables"`
}

type State struct {
	IsLightEnabled   bool `json:"isLightEnabled"`
	IsTrainEnabled   bool `json:"isTrainEnabled"`
	IsFactoryEnabled bool `json:"isFactoryEnabled"`
	IsBlackout       bool `json:"isBlackout"`
}

type Variables struct {
	TotalPower   float32 `json:"totalPower"`
	SurplusPower float32 `json:"surplusPower"`
}

type MultipleDevicePowerResponse struct {
	TotalPower float32        `json:"totalPower"`
	Devices    []DeviceDetail `json:"devices"`
}

type DeviceDetail struct {
	DeviceID   string  `json:"deviceId"`
	DeviceType string  `json:"deviceType"`
	Power      float32 `json:"power"`
	GpsLat     string  `json:"gpsLat"`
	GpsLon     string  `json:"gpsLon"`
}

func (repo *ManagementRepository) GetCurrentWorldState(ctx context.Context, tx *sql.Tx, sessionID string) (CurrentWorldState, error) {
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

	err = getWorldStmt.QueryRowContext(ctx, sessionID).Scan(
		&isLightEnabled,
		&isTrainEnabled,
		&isFactoryEnabled,
		&isBlackout,
		&villagersTextBytes,
	)
	if err != nil {
		return CurrentWorldState{}, &custmerr.TechnicalErr{Err: fmt.Errorf("failed to scan world state: %w", err)}
	}

	var villagersText []string
	if len(villagersTextBytes) > 0 {
		if err := json.Unmarshal(villagersTextBytes, &villagersText); err != nil {
			return CurrentWorldState{}, &custmerr.TechnicalErr{Err: fmt.Errorf("failed to unmarshal villagers_text: %w", err)}
		}
	}

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

type PowerChartData struct {
	TimeLabels []string  `json:"timeLabels"`
	Geothermal []float64 `json:"geothermal"`
	Hydro      []float64 `json:"hydro"`
	Wind       []float64 `json:"wind"`
	Solar      []float64 `json:"solar"`
}

func (repo *ManagementRepository) GetPowerHistory(ctx context.Context, tx *sql.Tx, sessionId string) (PowerChartData, error) {
	stmt, err := tx.PrepareContext(ctx, `
        SELECT
            to_timestamp(
                FLOOR(EXTRACT(EPOCH FROM pl.timestamp) / (3*60)) * 3*60
            ) AS bucket,
            d.device_type,
            AVG(pl.power) AS avg_power
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
        GROUP BY
			bucket, d.device_type
        ORDER BY
			bucket
		ASC, d.device_type;
    `)
	if err != nil {
		return PowerChartData{}, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, sessionId)
	if err != nil {
		return PowerChartData{}, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// 時間スロットごとのマップを作成
	timeMap := make(map[string]map[string]float64)
	var timeLabels []string

	for rows.Next() {
		var bucket time.Time
		var deviceType string
		var avgPower float64
		if err := rows.Scan(&bucket, &deviceType, &avgPower); err != nil {
			return PowerChartData{}, fmt.Errorf("failed to scan row: %w", err)
		}

		t := bucket.Format(time.RFC3339)
		if _, exists := timeMap[t]; !exists {
			timeMap[t] = make(map[string]float64)
			timeLabels = append(timeLabels, t)
		}
		timeMap[t][deviceType] = avgPower
	}

	// PowerChartData に変換
	chartData := PowerChartData{
		TimeLabels: timeLabels,
		Geothermal: make([]float64, len(timeLabels)),
		Hydro:      make([]float64, len(timeLabels)),
		Wind:       make([]float64, len(timeLabels)),
		Solar:      make([]float64, len(timeLabels)),
	}

	for i, t := range timeLabels {
		if val, ok := timeMap[t]["geothermal"]; ok {
			chartData.Geothermal[i] = val
		}
		if val, ok := timeMap[t]["hydrogen"]; ok {
			chartData.Hydro[i] = val
		}
		if val, ok := timeMap[t]["wind"]; ok {
			chartData.Wind[i] = val
		}
		if val, ok := timeMap[t]["solar"]; ok {
			chartData.Solar[i] = val
		}
	}

	return chartData, nil
}
