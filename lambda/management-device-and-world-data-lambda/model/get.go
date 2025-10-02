package model

import (
	"context"
	"data-manager/custmerr"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
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

		power, err := strconv.ParseFloat(powerStr, 64)
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
	IsLightEnabled    bool `json:"isLightEnabled"`
	IsTrainEnabled    bool `json:"isTrainEnabled"`
	IsFactoryEnabled  bool `json:"isFactoryEnabled"`
	IsHouseEnabled    bool `json:"isHouseEnabled"`
	IsFacilityEnabled bool `json:"isFacilityEnabled"`
	IsBlackout        bool `json:"isBlackout"`
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
			is_light_enabled,is_train_enabled,is_factory_enabled,is_blackout,is_house_enabled,is_facility_enabled,villagers_text
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

	err = getWorldStmt.QueryRowContext(ctx, sessionID).Scan(
		&isLightEnabled,
		&isTrainEnabled,
		&isFactoryEnabled,
		&isBlackout,
		&isHouseEnabled,
		&isFacilityEnabled,
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

	// DynamoDBから最新の発電量を取得
	latestPower := make(map[string]float32)
	var allPower float32

	// 各デバイスタイプについてDynamoDBから最新データを取得
	deviceTypes := []string{"solar", "geothermal", "hydrogen", "wind", "fire"}
	for _, deviceType := range deviceTypes {
		response, err := repo.GetMultipleDevicesPowerDataFromDynamoDB(ctx, deviceType, sessionID)
		if err != nil {
			// エラーが発生しても他のデバイスタイプの処理を続行
			continue
		}
		if response.TotalPower > 0 {
			latestPower[deviceType] = response.TotalPower
			allPower += response.TotalPower
		}
	}

	var currentPowerConsumption float32
	if isLightEnabled {
		currentPowerConsumption += 5.0
	}
	if isTrainEnabled {
		currentPowerConsumption += 410.0
	}
	if isFactoryEnabled {
		currentPowerConsumption += 300.0
	}
	if isHouseEnabled {
		currentPowerConsumption += 300.0
	}
	if isFacilityEnabled {
		currentPowerConsumption += 1015.0
	}

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

	registerNewWorldStateStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO
			world_state(session_id,is_light_enabled,is_train_enabled,
			is_factory_enabled,is_blackout,is_house_enabled,is_facility_enabled,total_power,surplus_power,villagers_text,timestamp)
		VALUES
			($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NOW())
	`)
	if err != nil {
		return CurrentWorldState{}, &custmerr.TechnicalErr{Err: fmt.Errorf("failed to prepare register new world state statement: %w", err)}
	}
	defer registerNewWorldStateStmt.Close()

	villagersTextJSON, err := json.Marshal(villagersText)
	if err != nil {
		return CurrentWorldState{}, &custmerr.TechnicalErr{Err: fmt.Errorf("failed to marshal villagers_text: %w", err)}
	}

	_, err = registerNewWorldStateStmt.ExecContext(ctx, sessionID, isLightEnabled, isTrainEnabled, isFactoryEnabled, isBlackout, isHouseEnabled, isFacilityEnabled, allPower, surplusPower, villagersTextJSON)
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
	TotalPower float64   `json:"totalPower"`
}

type RawLog struct {
	Bucket     time.Time
	Timestamp  time.Time
	Power      float64
	DeviceID   string
	DeviceType string
}

type DeviceBucket struct {
	Bucket time.Time // バケット開始時刻
	Logs   []RawLog
}
type DeviceBuckets struct {
	DeviceID   string
	DeviceType string
	Buckets    []*DeviceBucket
}

func (repo *ManagementRepository) calculateTotalPower(
	ctx context.Context,
	tx *sql.Tx,
	sessionId string,
	bucketSeconds int,
) (float64, error) {

	stmt, err := tx.PrepareContext(ctx, `
			SELECT
				to_timestamp(
					FLOOR(
						(EXTRACT(EPOCH FROM pl.timestamp) +
						CASE
							WHEN MOD(EXTRACT(EPOCH FROM pl.timestamp), $2 ) = 0 THEN 1
							ELSE 0
						END
						) / $2
					) * $2
				) AT TIME ZONE 'UTC' AS bucket,
			pl.timestamp,
			pl.power,
			d.device_id,
			d.device_type
		FROM
			power_logs pl
		JOIN
			session_devices sd ON pl.session_device_id = sd.id
		JOIN
			devices d ON sd.device_id = d.device_id
		WHERE
			sd.session_id = $1
		ORDER BY
			bucket ASC, pl.timestamp ASC;
    `)
	if err != nil {
		return 0.0, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, sessionId, bucketSeconds)
	if err != nil {
		return 0.0, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	var rawLogs []RawLog
	for rows.Next() {
		var pl RawLog
		if err := rows.Scan(&pl.Bucket, &pl.Timestamp, &pl.Power, &pl.DeviceID, &pl.DeviceType); err != nil {
			return 0.0, fmt.Errorf("failed to scan row: %w", err)
		}
		rawLogs = append(rawLogs, pl)
	}
	if err := rows.Err(); err != nil {
		return 0.0, fmt.Errorf("rows error: %w", err)
	}

	deviceMap := make(map[string]*DeviceBuckets)
	for _, log := range rawLogs {

		// 時間どうこう以前に大元のdeviceIDになにもデータが紐づいてなかったら初期化
		if _, exists := deviceMap[log.DeviceID]; !exists {
			deviceMap[log.DeviceID] = &DeviceBuckets{
				DeviceID:   log.DeviceID,
				DeviceType: log.DeviceType,
				Buckets:    []*DeviceBucket{},
			}
		}
		device := deviceMap[log.DeviceID]

		// n分区切りのバケットが存在しているか確認
		var bucket *DeviceBucket
		for _, b := range device.Buckets {
			if b.Bucket.Equal(log.Bucket) {
				bucket = b
				break
			}
		}

		if bucket == nil {
			bucket = &DeviceBucket{
				Bucket: log.Bucket,
				Logs:   []RawLog{},
			}
			device.Buckets = append(device.Buckets, bucket)
		}

		bucket.Logs = append(bucket.Logs, log)
	}

	for _, device := range deviceMap {
		// n分区切りのバケットを昇順ソート
		sort.Slice(device.Buckets, func(i, j int) bool {
			return device.Buckets[i].Bucket.Before(device.Buckets[j].Bucket)
		})

		//  各バケット内のログを昇順にソート
		for _, b := range device.Buckets {
			sort.Slice(b.Logs, func(i, j int) bool {
				return b.Logs[i].Timestamp.Before(b.Logs[j].Timestamp)
			})
		}
	}

	// 時間をまたぐ場合の時間軸でどのような発電量を取るか記録する
	frontMidpoint := make(map[string]map[time.Time]float64)
	rearMidpoint := make(map[string]map[time.Time]float64)

	// まだ時間の区切りが一つしか無い場合は別の処理をする必要があるので記録しておく
	single := make(map[string]bool)

	for deviceID, device := range deviceMap {
		frontMidpoint[deviceID] = make(map[time.Time]float64)
		rearMidpoint[deviceID] = make(map[time.Time]float64)

		if len(device.Buckets) < 2 {
			single[deviceID] = true
			continue
		}

		// bucketsの要素を最初の一つ飛ばして一つずつ取り出す
		for i := 1; i < len(device.Buckets); i++ {
			bucket := device.Buckets[i]

			prevBucket := device.Buckets[i-1]
			prevBucketLastLog := prevBucket.Logs[len(prevBucket.Logs)-1]
			currBucketFirstLog := bucket.Logs[0]

			// 区間全体の長さ
			totalDuration := currBucketFirstLog.Timestamp.Sub(prevBucketLastLog.Timestamp).Seconds()
			if totalDuration <= 0 {
				continue // 時間が逆転している場合はスキップ
			}

			// 上記の区間全体のうち､境界はどれだけ進んでいるか
			elapsedDuration := bucket.Bucket.Sub(prevBucketLastLog.Timestamp).Seconds()
			if elapsedDuration < 0 {
				continue // 境界が prevBucketLastLog より前ならスキップ
			}

			// 傾きに先ほど求めた割合をかけて､最後の点からの増分を求める
			powerAtBoundary := prevBucketLastLog.Power +
				(currBucketFirstLog.Power-prevBucketLastLog.Power)*(elapsedDuration/totalDuration)

			frontMidpoint[deviceID][bucket.Bucket] = powerAtBoundary
			rearMidpoint[deviceID][prevBucket.Bucket] = powerAtBoundary
		}
	}

	// バケットごとの積分結果を保存するマップ
	bucketPowerResult := make(map[string]map[time.Time]float64) // deviceID -> bucket -> Wh

	for deviceID, device := range deviceMap {
		bucketPowerResult[deviceID] = make(map[time.Time]float64)

		// 時間の区切りがまだ一つしかない場合
		if single[deviceID] {
			// ログ一つもないならスキップ
			if len(device.Buckets) == 0 {
				continue
			}

			bucket := device.Buckets[0]
			logs := bucket.Logs
			if len(logs) == 0 {
				continue
			}

			sumWs := 0.0

			if len(logs) == 1 {
				// ログが一つしか無いなら長方形でWhは求められる
				onlyLog := logs[0]
				duration := onlyLog.Timestamp.Sub(bucket.Bucket).Seconds()
				if duration > 0 {
					sumWs += onlyLog.Power * duration
				}
			} else {
				for i := 1; i < len(logs); i++ {
					prev := logs[i-1]
					curr := logs[i]

					duration := curr.Timestamp.Sub(prev.Timestamp).Seconds()
					if duration <= 0 {
						continue
					}
					//台形で計算できる
					area := (prev.Power + curr.Power) / 2.0 * duration
					sumWs += area
				}

				// 上記のfor文では台形しか計算していないので､その区間の前後の端は計算されない｡よって計算する｡
				frontDuration := logs[0].Timestamp.Sub(bucket.Bucket).Seconds()
				sumWs += logs[0].Power * frontDuration

				bucketEndTime := bucket.Bucket.Add(time.Duration(bucketSeconds) * time.Second)
				rearDuration := bucketEndTime.Sub(logs[len(logs)-1].Timestamp).Seconds()
				sumWs += logs[len(logs)-1].Power * rearDuration

			}

			bucketPowerResult[deviceID][bucket.Bucket] = sumWs / 3600.0 // Ws -> Wh

		} else {
			for idx, bucket := range device.Buckets {
				logs := bucket.Logs
				if len(logs) == 0 {
					continue
				}

				bucketStart := bucket.Bucket
				bucketStartEnd := bucket.Bucket.Add(time.Duration(bucketSeconds) * time.Second)
				bucketSumWs := 0.0

				var powerAtStart float64
				if idx == 0 {
					// 最初のバケットは長方形で計算するためそのままの値を使う
					powerAtStart = logs[0].Power
				} else {
					// 最初出ない場合は時間の区切りをまたぐので先程計算した値を使う
					powerAtStart = frontMidpoint[deviceID][bucketStart]
					fmt.Println(powerAtStart)
				}

				// 実際に区切りの最初の部分の面積を計算
				if idx == 0 {
					// 最初のバケットは長方形で計算するためそのままの値を使う
					duration := logs[0].Timestamp.Sub(bucketStart).Seconds()
					bucketSumWs += powerAtStart * duration
				} else {
					// 最初でない場合は台形で計算する
					duration := logs[0].Timestamp.Sub(bucketStart).Seconds()
					bucketSumWs += (powerAtStart + logs[0].Power) / 2.0 * duration
				}

				// 残りは普通に台形で計算
				for i := 1; i < len(logs); i++ {
					prev := logs[i-1]
					curr := logs[i]

					duration := curr.Timestamp.Sub(prev.Timestamp).Seconds()
					if duration <= 0 {
						continue
					}
					//台形で計算できる
					area := (prev.Power + curr.Power) / 2.0 * duration
					bucketSumWs += area
				}

				if idx != len(device.Buckets)-1 {
					// 最後の部分は台形で計算する
					duration := bucketStartEnd.Sub(logs[len(logs)-1].Timestamp).Seconds()
					bucketSumWs += (logs[len(logs)-1].Power + rearMidpoint[deviceID][bucketStart]) / 2.0 * duration
				}

				bucketPowerResult[deviceID][bucketStart] = bucketSumWs / 3600.0 // Ws -> Wh
			}
		}
	}

	bucketSet := make(map[time.Time]struct{})
	for _, device := range deviceMap {
		for _, bucket := range device.Buckets {
			bucketSet[bucket.Bucket] = struct{}{}
		}
	}

	// 時刻をソート
	var bucketTimes []time.Time
	for t := range bucketSet {
		bucketTimes = append(bucketTimes, t)
	}
	sort.Slice(bucketTimes, func(i, j int) bool {
		return bucketTimes[i].Before(bucketTimes[j])
	})
	// timeLabelsを作成
	var timeLabels []string
	for _, t := range bucketTimes {
		timeLabels = append(timeLabels, t.Format("15:04"))
	}

	// 各発電種別の配列をtimeLabelsと同じ長さで確保
	geothermal := make([]float64, len(bucketTimes))
	hydro := make([]float64, len(bucketTimes))
	wind := make([]float64, len(bucketTimes))
	solar := make([]float64, len(bucketTimes))

	// バケット×デバイスごとに対応付け
	for deviceID, device := range deviceMap {
		bucketMap := bucketPowerResult[deviceID]

		for _, bucket := range device.Buckets {
			// バケット位置を特定
			index := sort.Search(len(bucketTimes), func(i int) bool {
				return !bucketTimes[i].Before(bucket.Bucket)
			})
			if index >= len(bucketTimes) || !bucketTimes[index].Equal(bucket.Bucket) {
				continue
			}

			// バケットごとの積分結果からWhを取得
			sumPower, exists := bucketMap[bucket.Bucket]
			if !exists {
				continue
			}

			switch device.DeviceType {
			case "geothermal":
				geothermal[index] += sumPower
			case "hydrogen":
				hydro[index] += sumPower
			case "wind":
				wind[index] += sumPower
			case "solar":
				solar[index] += sumPower
			}
		}
	}

	// 総発電量(kWh)
	totalPower := 0.0

	for _, geo := range geothermal {
		totalPower += geo
	}
	for _, hyd := range hydro {
		totalPower += hyd
	}
	for _, win := range wind {
		totalPower += win
	}
	for _, sol := range solar {
		totalPower += sol
	}

	return totalPower, nil
}

type ResultData struct {
	DeviceID   string
	DeviceType string
	SumPower   float64
}

func (repo *ManagementRepository) GetPowerHistory(ctx context.Context, tx *sql.Tx, sessionId string) (PowerChartData, error) {
	// 10秒間隔で各デバイスの最新発電量を取得
	stmt, err := tx.PrepareContext(ctx, `
        SELECT
            bucket,
            device_id,
            device_type,
            power
        FROM (
            SELECT
                to_timestamp(
                    FLOOR(EXTRACT(EPOCH FROM pl.timestamp) / 10) * 10
                ) AT TIME ZONE 'UTC' AS bucket,
                d.device_id,
                d.device_type,
                pl.power,
                ROW_NUMBER() OVER (
                    PARTITION BY d.device_id, to_timestamp(FLOOR(EXTRACT(EPOCH FROM pl.timestamp) / 10) * 10)
                    ORDER BY pl.timestamp DESC
                ) as rn
            FROM
                power_logs pl
            JOIN
                session_devices sd ON pl.session_device_id = sd.id
            JOIN
                devices d ON sd.device_id = d.device_id
            WHERE
                sd.session_id = $1
        ) ranked_data
        WHERE rn = 1
        ORDER BY
            bucket ASC, device_id;
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

	// 10秒間隔×デバイス別のマップを作成（各デバイスの最新データのみ）
	bucketDeviceMap := make(map[string]map[string]float64) // bucket -> deviceID -> power
	bucketTypeMap := make(map[string]map[string]float64)   // timeLabel -> deviceType -> total power
	bucketToLabelMap := make(map[string]string)            // bucketStr -> timeLabel
	var timeLabels []string
	timeSet := make(map[string]bool)

	for rows.Next() {
		var bucket time.Time
		var deviceID, deviceType string
		var power float64
		if err := rows.Scan(&bucket, &deviceID, &deviceType, &power); err != nil {
			return PowerChartData{}, fmt.Errorf("failed to scan row: %w", err)
		}

		bucketStr := bucket.Format(time.RFC3339)
		timeLabel := bucket.Format("15:04:05") // 時:分:秒 形式

		// 時間ラベルを記録
		if !timeSet[bucketStr] {
			timeSet[bucketStr] = true
			timeLabels = append(timeLabels, timeLabel)
			bucketToLabelMap[bucketStr] = timeLabel
		}

		// デバイス別発電量を記録
		if _, exists := bucketDeviceMap[bucketStr]; !exists {
			bucketDeviceMap[bucketStr] = make(map[string]float64)
		}
		bucketDeviceMap[bucketStr][deviceID] = power

		// deviceType別に加算（timeLabelをキーとして使用）
		if _, exists := bucketTypeMap[timeLabel]; !exists {
			bucketTypeMap[timeLabel] = make(map[string]float64)
		}
		bucketTypeMap[timeLabel][deviceType] += power
	}

	// 時間ラベルをソート
	sort.Strings(timeLabels)

	totalPower, err := repo.calculateTotalPower(ctx, tx, sessionId, 10)
	if err != nil {
		return PowerChartData{}, fmt.Errorf("failed to calculate total power: %w", err)
	}

	// PowerChartData に変換
	chartData := PowerChartData{
		TimeLabels: timeLabels,
		Geothermal: make([]float64, len(timeLabels)),
		Hydro:      make([]float64, len(timeLabels)),
		Wind:       make([]float64, len(timeLabels)),
		Solar:      make([]float64, len(timeLabels)),
		TotalPower: totalPower,
	}

	for i, t := range timeLabels {
		if val, ok := bucketTypeMap[t]["geothermal"]; ok {
			chartData.Geothermal[i] = val
		}
		if val, ok := bucketTypeMap[t]["hydrogen"]; ok {
			chartData.Hydro[i] = val
		}
		if val, ok := bucketTypeMap[t]["wind"]; ok {
			chartData.Wind[i] = val
		}
		if val, ok := bucketTypeMap[t]["solar"]; ok {
			chartData.Solar[i] = val
		}
	}

	return chartData, nil
}
