package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"data-manager/custmerr"
	"data-manager/model"

	"github.com/aws/aws-lambda-go/events"
)

type RegistrationNewPowerGenerationModuleRequestBody struct {
	SessionID  string `json:"sessionId"`
	DeviceID   string `json:"deviceId"`
	DeviceType string `json:"deviceType"`
}

type ManagementController struct {
	repo model.ManagementRepositoryInterface
}

func NewManagementController(repo model.ManagementRepositoryInterface) *ManagementController {
	return &ManagementController{repo: repo}
}

func (c *ManagementController) RegisterNewPowerGenerationModuleHandler(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	var requestBody RegistrationNewPowerGenerationModuleRequestBody
	if err := json.Unmarshal([]byte(req.Body), &requestBody); err != nil {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 400,
			Body:       fmt.Sprintf("Invalid request body: %v", err),
		}, nil
	}

	if requestBody.SessionID == "" || requestBody.DeviceID == "" || requestBody.DeviceType == "" {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 400,
			Body:       "Missing required fields",
		}, nil
	}

	tx, err := c.repo.BeginTx(ctx, nil)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf("Failed to begin transaction: %v", err),
		}, nil
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
		}
	}()

	err = c.repo.CreateSessionIfNotExists(ctx, tx, requestBody.SessionID)
	if err != nil {
		tx.Rollback()
		var lErr *custmerr.LogicalErr
		var tErr *custmerr.TechnicalErr

		switch {
		case errors.As(err, &lErr):
			return events.APIGatewayV2HTTPResponse{
				StatusCode: 404,
				Body:       fmt.Sprintf("Session found or created failed: %v", err),
			}, nil

		case errors.As(err, &tErr):
			return events.APIGatewayV2HTTPResponse{
				StatusCode: 500,
				Body:       fmt.Sprintf("Technical error occurred: %v", err),
			}, nil
		}

	}

	err = c.repo.RegisterNewPowerGenerationModule(ctx, tx, requestBody.SessionID, requestBody.DeviceID, requestBody.DeviceType)
	if err != nil {
		var tErr *custmerr.TechnicalErr
		if errors.As(err, &tErr) {
			tx.Rollback()
			return events.APIGatewayV2HTTPResponse{
				StatusCode: 500,
				Body:       fmt.Sprintf("Technical error occurred: %v", err),
			}, nil
		}
	}

	err = c.repo.CreateNewWorldIfNotExists(ctx, tx, requestBody.SessionID)
	if err != nil {
		var tErr *custmerr.TechnicalErr
		if errors.As(err, &tErr) {
			tx.Rollback()
			return events.APIGatewayV2HTTPResponse{
				StatusCode: 500,
				Body:       fmt.Sprintf("Technical error occurred: %v", err),
			}, nil
		}
	}

	tx.Commit()

	return events.APIGatewayV2HTTPResponse{
		StatusCode: 200,
		Body:       "Registration successful",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}, nil
}

func (c *ManagementController) GetLatestPower(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	var deviceType string
	if deviceType = req.QueryStringParameters["device_type"]; deviceType == "" {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 400,
			Body:       "Missing required query parameter: device_type",
		}, nil
	}

	var sessionId string
	if sessionId = req.QueryStringParameters["session_id"]; sessionId == "" {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 400,
			Body:       "Missing required query parameter: session_id",
		}, nil
	}

	tx, err := c.repo.BeginTx(ctx, nil)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf("Failed to begin transaction: %v", err),
		}, nil
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
		}
	}()

	latestPowerData, gpsLat, gpsLon, err := c.repo.GetLatestPowerData(ctx, tx, deviceType, sessionId)
	if err != nil {
		tx.Rollback()
		var lErr *custmerr.LogicalErr
		var tErr *custmerr.TechnicalErr
		switch {
		case errors.As(err, &lErr):
			return events.APIGatewayV2HTTPResponse{
				StatusCode: 404,
				Body:       fmt.Sprintf("No power data found for device type %s: %v", deviceType, err),
			}, nil

		case errors.As(err, &tErr):
			return events.APIGatewayV2HTTPResponse{
				StatusCode: 500,
				Body:       fmt.Sprintf("Technical error occurred: %v", err),
			}, nil
		}
	}

	bodyBytes, err := json.Marshal(map[string]interface{}{
		"latestPower": latestPowerData,
		"gpsLat":      gpsLat,
		"gpsLon":      gpsLon,
	})
	if err != nil {
		tx.Rollback()
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf("Failed to marshal response: %v", err),
		}, nil
	}

	tx.Commit()

	return events.APIGatewayV2HTTPResponse{
		StatusCode: 200,
		Body:       string(bodyBytes),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}, nil
}

func (c *ManagementController) GetLatestMultipleDevicePower(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	var deviceType string
	if deviceType = req.QueryStringParameters["device_type"]; deviceType == "" {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 400,
			Body:       "Missing required query parameter: device_type",
		}, nil
	}

	var sessionId string
	if sessionId = req.QueryStringParameters["session_id"]; sessionId == "" {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 400,
			Body:       "Missing required query parameter: session_id",
		}, nil
	}

	tx, err := c.repo.BeginTx(ctx, nil)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf("Failed to begin transaction: %v", err),
		}, nil
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
		}
	}()

	latestPowerData, err := c.repo.GetMultipleDevicesPowerDataFromDynamoDB(ctx, deviceType, sessionId)
	if err != nil {
		tx.Rollback()
		var lErr *custmerr.LogicalErr
		var tErr *custmerr.TechnicalErr
		switch {
		case errors.As(err, &lErr):
			return events.APIGatewayV2HTTPResponse{
				StatusCode: 404,
				Body:       fmt.Sprintf("No power data found for device type %s: %v", deviceType, err),
			}, nil

		case errors.As(err, &tErr):
			return events.APIGatewayV2HTTPResponse{
				StatusCode: 500,
				Body:       fmt.Sprintf("Technical error occurred: %v", err),
			}, nil
		}
	}

	bodyBytes, err := json.Marshal(latestPowerData)
	if err != nil {
		tx.Rollback()
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf("Failed to marshal response: %v", err),
		}, nil
	}

	tx.Commit()

	return events.APIGatewayV2HTTPResponse{
		StatusCode: 200,
		Body:       string(bodyBytes),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}, nil
}

type equipmentRequest struct {
	SessionId string `json:"sessionId"`
	Equipment string `json:"equipment"`
}

func (c *ManagementController) TurnOnEquipment(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	var equipmentRequestBody equipmentRequest
	if err := json.Unmarshal([]byte(req.Body), &equipmentRequestBody); err != nil {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 400,
			Body:       fmt.Sprintf("Invalid request body: %v", err),
		}, nil
	}

	if equipmentRequestBody.Equipment == "" {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 500,
			Body:       "Missing equipment",
		}, nil
	}

	tx, err := c.repo.BeginTx(ctx, nil)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf("Failed to begin transaction: %v", err),
		}, nil
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
		}
	}()

	currentState, err := c.repo.TurnOnEquipment(ctx, tx, equipmentRequestBody.SessionId, equipmentRequestBody.Equipment)
	if err != nil {
		tx.Rollback()
		var lErr *custmerr.LogicalErr
		var tErr *custmerr.TechnicalErr
		switch {
		case errors.As(err, &lErr):
			return events.APIGatewayV2HTTPResponse{
				StatusCode: 404,
				Body:       fmt.Sprintf("Session not found: %v", err),
			}, nil

		case errors.As(err, &tErr):
			return events.APIGatewayV2HTTPResponse{
				StatusCode: 500,
				Body:       fmt.Sprintf("Technical error occurred: %v", err),
			}, nil
		}
	}

	jsonBytes, err := json.Marshal(currentState)
	if err != nil {
		// JSONへの変換に失敗した場合のエラーハンドリング
		tx.Rollback()
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf("Failed to marshal JSON: %v", err),
		}, nil
	}

	tx.Commit()

	return events.APIGatewayV2HTTPResponse{
		StatusCode: 200,
		Body:       string(jsonBytes),
	}, nil
}

func (c *ManagementController) TurnOffEquipment(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	var equipmentRequestBody equipmentRequest
	if err := json.Unmarshal([]byte(req.Body), &equipmentRequestBody); err != nil {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 400,
			Body:       fmt.Sprintf("Invalid request body: %v", err),
		}, nil
	}

	if equipmentRequestBody.Equipment == "" {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 500,
			Body:       "Missing equipment",
		}, nil
	}

	tx, err := c.repo.BeginTx(ctx, nil)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf("Failed to begin transaction: %v", err),
		}, nil
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
		}
	}()

	currentState, err := c.repo.TurnOffEquipment(ctx, tx, equipmentRequestBody.SessionId, equipmentRequestBody.Equipment)
	if err != nil {
		tx.Rollback()
		var lErr *custmerr.LogicalErr
		var tErr *custmerr.TechnicalErr
		switch {
		case errors.As(err, &lErr):
			return events.APIGatewayV2HTTPResponse{
				StatusCode: 404,
				Body:       fmt.Sprintf("Session not found: %v", err),
			}, nil

		case errors.As(err, &tErr):
			return events.APIGatewayV2HTTPResponse{
				StatusCode: 500,
				Body:       fmt.Sprintf("Technical error occurred: %v", err),
			}, nil
		}
	}

	jsonBytes, err := json.Marshal(currentState)
	if err != nil {
		// JSONへの変換に失敗した場合のエラーハンドリング
		tx.Rollback()
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf("Failed to marshal JSON: %v", err),
		}, nil
	}

	tx.Commit()

	return events.APIGatewayV2HTTPResponse{
		StatusCode: 200,
		Body:       string(jsonBytes),
	}, nil
}

type worldStateRequest struct {
	SessionId string `json:"sessionId"`
}

func (c *ManagementController) GetCurrentWorldState(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	var worldStateRequestBody worldStateRequest
	if err := json.Unmarshal([]byte(req.Body), &worldStateRequestBody); err != nil {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 400,
			Body:       fmt.Sprintf("Invalid request body: %v", err),
		}, nil
	}

	if worldStateRequestBody.SessionId == "" {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 400,
			Body:       "Missing session_id",
		}, nil
	}

	tx, err := c.repo.BeginTx(ctx, nil)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf("Failed to begin transaction: %v", err),
		}, nil
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
		}
	}()

	currentState, err := c.repo.GetCurrentWorldState(ctx, tx, worldStateRequestBody.SessionId)
	if err != nil {
		tx.Rollback()
		var lErr *custmerr.LogicalErr
		var tErr *custmerr.TechnicalErr
		switch {
		case errors.As(err, &lErr):
			return events.APIGatewayV2HTTPResponse{
				StatusCode: 404,
				Body:       fmt.Sprintf("Session not found: %v", err),
			}, nil

		case errors.As(err, &tErr):
			return events.APIGatewayV2HTTPResponse{
				StatusCode: 500,
				Body:       fmt.Sprintf("Technical error occurred: %v", err),
			}, nil
		}
	}

	jsonBytes, err := json.Marshal(currentState)
	if err != nil {
		tx.Rollback()
		// JSONへの変換に失敗した場合のエラーハンドリング
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf("Failed to marshal JSON: %v", err),
		}, nil
	}

	tx.Commit()

	return events.APIGatewayV2HTTPResponse{
		StatusCode: 200,
		Body:       string(jsonBytes),
	}, nil
}

func (c *ManagementController) GetPowerHistory(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	var sessionId string
	if sessionId = req.QueryStringParameters["session_id"]; sessionId == "" {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 400,
			Body:       "Missing required query parameter: session_id",
		}, nil
	}

	tx, err := c.repo.BeginTx(ctx, nil)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf("Failed to begin transaction: %v", err),
		}, nil
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
		}
	}()

	chartData, err := c.repo.GetPowerHistory(ctx, tx, sessionId)
	if err != nil {
		tx.Rollback()
		var lErr *custmerr.LogicalErr
		var tErr *custmerr.TechnicalErr
		switch {
		case errors.As(err, &lErr):
			return events.APIGatewayV2HTTPResponse{
				StatusCode: 404,
				Body:       fmt.Sprintf("Session not found: %v", err),
			}, nil

		case errors.As(err, &tErr):
			return events.APIGatewayV2HTTPResponse{
				StatusCode: 500,
				Body:       fmt.Sprintf("Technical error occurred: %v", err),
			}, nil
		}
	}

	bodyBytes, err := json.Marshal(chartData)
	if err != nil {
		tx.Rollback()
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf("Failed to marshal response: %v", err),
		}, nil
	}

	tx.Commit()

	return events.APIGatewayV2HTTPResponse{
		StatusCode: 200,
		Body:       string(bodyBytes),
	}, nil
}

type HappinessDetail struct {
	EnvironmentProblemScore     float64 `json:"environmentProblemScore"`
	EnvironmentProblemNumber    int     `json:"environmentProblemNumber"`
	PowerStabilityScore         float64 `json:"powerStabilityScore"`
	PowerStabilityNumber        int     `json:"powerStabilityNumber"`
	InfrastructureComfortScore  float64 `json:"infrastructureComfortScore"`
	InfrastructureComfortNumber int     `json:"infrastructureComfortNumber"`
}
type GameResult struct {
	TotalPowerGeneration                          float64           `json:"totalPowerGeneration"`
	HydrogenMaximumInstantaneousPowerGeneration   float64           `json:"hydrogenMaximumInstantaneousPowerGeneration"`
	WindMaximumInstantaneousPowerGeneration       float64           `json:"windMaximumInstantaneousPowerGeneration"`
	SolarMaximumInstantaneousPowerGeneration      float64           `json:"solarMaximumInstantaneousPowerGeneration"`
	GeothermalMaximumInstantaneousPowerGeneration float64           `json:"geothermalMaximumInstantaneousPowerGeneration"`
	CO2ReductionAmount                            float64           `json:"co2ReductionAmount"`
	Happiness                                     HappinessDetail   `json:"happiness"`
	VillagersTexts                                map[string]string `json:"villagersTexts"`
}

func (c *ManagementController) GetGameResult(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	if req.QueryStringParameters["session_id"] == "" {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 400,
			Body:       "Missing required query parameter: session_id",
		}, nil
	}

	sessionId := req.QueryStringParameters["session_id"]

	tx, err := c.repo.BeginTx(ctx, nil)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf("Failed to begin transaction: %v", err),
		}, nil
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
		}
	}()

	// 総発電量
	totalPower, err := c.repo.CalculateTotalPower(ctx, tx, sessionId, 10)
	if err != nil {
		tx.Rollback()
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf("Technical error occurred: %v", err),
		}, nil
	}

	maxPower, err := c.repo.GetMaxPowerGeneration(ctx, tx, sessionId)
	if err != nil {
		tx.Rollback()
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf("Technical error occurred: %v", err),
		}, nil
	}

	fireTotalPower, err := c.repo.CalculateTotalPowerByDeviceType(ctx, tx, sessionId, "fire", 10)
	if err != nil {
		tx.Rollback()
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf("Technical error occurred: %v", err),
		}, nil
	}

	windTotalPower, err := c.repo.CalculateTotalPowerByDeviceType(ctx, tx, sessionId, "wind", 10)
	if err != nil {
		tx.Rollback()
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf("Technical error occurred: %v", err),
		}, nil
	}

	blackoutCount, err := c.repo.GetBlackoutCount(ctx, tx, sessionId)
	if err != nil {
		tx.Rollback()
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf("Technical error occurred: %v", err),
		}, nil
	}

	currentWorldState, err := c.repo.GetCurrentWorldStateWithoutChanges(ctx, tx, sessionId)
	if err != nil {
		tx.Rollback()
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf("Technical error occurred: %v", err),
		}, nil
	}

	CO2Emissions := 0.415 * fireTotalPower

	var environmentProblemScore float64
	environmentProblemNumber := 0
	var powerStabilityScore float64
	powerStabilityNumber := 0
	var infrastructureComfortScore float64
	infrastructureComfortNumber := 0

	// 環境問題の苦情人数とスコア
	if CO2Emissions > 20.0 && CO2Emissions <= 30.0 {
		environmentProblemNumber = 25
	} else if CO2Emissions > 30.0 && CO2Emissions <= 40.0 {
		environmentProblemNumber = 50
	} else if CO2Emissions > 40.0 && CO2Emissions <= 50.0 {
		environmentProblemNumber = 75
	}

	if windTotalPower >= 50.0 {
		environmentProblemNumber += 25
	}
	environmentProblemScore = (300.0 - float64(environmentProblemNumber)) / 3

	// 電力の安定性に関する苦情人数とスコア
	if blackoutCount == 1 {
		powerStabilityNumber = 50
	} else if blackoutCount == 2 {
		powerStabilityNumber = 75
	} else if blackoutCount >= 3 {
		powerStabilityNumber = 100
	}
	powerStabilityScore = (300.0 - float64(powerStabilityNumber)) / 3

	// インフラの快適さに関する苦情人数とスコア
	if !currentWorldState.State.IsLightEnabled {
		infrastructureComfortNumber += 100 / 6
	}
	if !currentWorldState.State.IsTrainEnabled {
		infrastructureComfortNumber += 100 / 6
	}
	if !currentWorldState.State.IsFactoryEnabled {
		infrastructureComfortNumber += 100 / 6
	}
	if currentWorldState.State.IsHouseEnabled {
		infrastructureComfortNumber += 100 / 6
	}
	if currentWorldState.State.IsFacilityEnabled {
		infrastructureComfortNumber += 100 / 6
	}
	infrastructureComfortScore = (300.0 - float64(infrastructureComfortNumber)) / 3

	happinessDetail := HappinessDetail{
		EnvironmentProblemScore:     float64(environmentProblemScore),
		EnvironmentProblemNumber:    environmentProblemNumber,
		PowerStabilityScore:         float64(powerStabilityScore),
		PowerStabilityNumber:        powerStabilityNumber,
		InfrastructureComfortScore:  float64(infrastructureComfortScore),
		InfrastructureComfortNumber: infrastructureComfortNumber,
	}

	gameResult := GameResult{
		TotalPowerGeneration:                          totalPower,
		HydrogenMaximumInstantaneousPowerGeneration:   maxPower.Hydrogen,
		WindMaximumInstantaneousPowerGeneration:       maxPower.Wind,
		SolarMaximumInstantaneousPowerGeneration:      maxPower.Solar,
		GeothermalMaximumInstantaneousPowerGeneration: maxPower.Geothermal,

		CO2ReductionAmount: CO2Emissions,
		Happiness:          happinessDetail,
		VillagersTexts:     currentWorldState.Texts,
	}

	jsonGameResult, err := json.Marshal(gameResult)
	if err != nil {
		tx.Rollback()
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf("Failed to marshal JSON: %v", err),
		}, nil
	}

	tx.Commit()

	return events.APIGatewayV2HTTPResponse{
		StatusCode: 200,
		Body:       string(jsonGameResult),
	}, nil
}
