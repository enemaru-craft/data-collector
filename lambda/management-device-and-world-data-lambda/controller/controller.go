package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"

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

type setEquipmentPercentRequest struct {
	SessionId string `json:"sessionId"`
	Equipment string `json:"equipment"`
	Percent   int    `json:"percent"`
}

func (c *ManagementController) SetEquipmentPercent(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	var requestBody setEquipmentPercentRequest
	if err := json.Unmarshal([]byte(req.Body), &requestBody); err != nil {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 400,
			Body:       fmt.Sprintf("Invalid request body: %v", err),
		}, nil
	}

	if requestBody.Equipment == "" {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 400,
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

	currentState, err := c.repo.SetEquipmentPercent(ctx, tx, requestBody.SessionId, requestBody.Equipment, requestBody.Percent)
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

type DeleteSessionRequest struct {
	SessionID string `json:"sessionId"`
	Password  string `json:"password"`
}

func (c *ManagementController) DeleteSessionHandler(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	var body DeleteSessionRequest
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		return events.APIGatewayV2HTTPResponse{StatusCode: 400, Body: fmt.Sprintf("Invalid request body: %v", err)}, nil
	}

	if body.SessionID == "" || body.Password == "" {
		return events.APIGatewayV2HTTPResponse{StatusCode: 400, Body: "Missing session_id or password"}, nil
	}

	correctPasswordHash := []byte("$2a$12$r5QB3cHHkd8EYs/CCYzNOupT76zmEXqtcWqscrUCpARbu/jNAXKy6")

	fmt.Println(body.Password)

	err := bcrypt.CompareHashAndPassword(correctPasswordHash, []byte(string(body.Password)))
	if err != nil {
		fmt.Println(err)
		return events.APIGatewayV2HTTPResponse{StatusCode: 403, Body: "Invalid password"}, nil
	}

	tx, err := c.repo.BeginTx(ctx, nil)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{StatusCode: 500, Body: fmt.Sprintf("Failed to begin transaction: %v", err)}, nil
	}

	if err := c.repo.DeleteSessionAndRelatedData(ctx, tx, body.SessionID); err != nil {
		tx.Rollback()
		var tErr *custmerr.TechnicalErr
		var lErr *custmerr.LogicalErr
		switch {
		case errors.As(err, &tErr):
			return events.APIGatewayV2HTTPResponse{StatusCode: 500, Body: fmt.Sprintf("Technical error occurred: %v", err)}, nil
		case errors.As(err, &lErr):
			return events.APIGatewayV2HTTPResponse{StatusCode: 404, Body: fmt.Sprintf("Session not found: %v", err)}, nil
		default:
			return events.APIGatewayV2HTTPResponse{StatusCode: 500, Body: fmt.Sprintf("Failed to delete session: %v", err)}, nil
		}
	}

	tx.Commit()

	return events.APIGatewayV2HTTPResponse{StatusCode: 200, Body: "Session deleted"}, nil
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

type CommentDetail struct {
	Text      string `json:"text"`
	Sentiment string `json:"sentiment"` // "positive", "neutral", "negative"
}

type HappinessDetail struct {
	EnvironmentProblemScore      float64       `json:"environmentProblemScore"`
	EnvironmentProblemNumber     int           `json:"environmentProblemNumber"`
	EnvironmentProblemComment    CommentDetail `json:"environmentProblemComment"`
	PowerStabilityScore          float64       `json:"powerStabilityScore"`
	PowerStabilityNumber         int           `json:"powerStabilityNumber"`
	PowerStabilityComment        CommentDetail `json:"powerStabilityComment"`
	InfrastructureComfortScore   float64       `json:"infrastructureComfortScore"`
	InfrastructureComfortNumber  int           `json:"infrastructureComfortNumber"`
	InfrastructureComfortComment CommentDetail `json:"infrastructureComfortComment"`
}
type GameResult struct {
	TotalPowerGeneration                          float64                       `json:"totalPowerGeneration"`
	HydrogenMaximumInstantaneousPowerGeneration   float64                       `json:"hydrogenMaximumInstantaneousPowerGeneration"`
	WindMaximumInstantaneousPowerGeneration       float64                       `json:"windMaximumInstantaneousPowerGeneration"`
	SolarMaximumInstantaneousPowerGeneration      float64                       `json:"solarMaximumInstantaneousPowerGeneration"`
	GeothermalMaximumInstantaneousPowerGeneration float64                       `json:"geothermalMaximumInstantaneousPowerGeneration"`
	CO2ReductionAmount                            float64                       `json:"co2ReductionAmount"`
	GeothermalTotalPower                          float64                       `json:"geothermalTotalPower"`
	FireTotalPower                                float64                       `json:"fireTotalPower"`
	WindTotalPower                                float64                       `json:"windTotalPower"`
	SolarTotalPower                               float64                       `json:"solarTotalPower"`
	HydrogenTotalPower                            float64                       `json:"hydrogenTotalPower"`
	Happiness                                     HappinessDetail               `json:"happiness"`
	VillagersTexts                                map[string]model.VillagerText `json:"villagersTexts"`
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

	totalPowerByPowerGenerationType, err := c.repo.CalculateTotalPowerByPowerGenerationType(ctx, tx, sessionId, 10)
	if err != nil {
		tx.Rollback()
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf("Technical error occurred: %v", err),
		}, nil
	}

	renewableEnergyTotalPower, err := c.repo.CalculateTotalPower(ctx, tx, sessionId, 10)
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

	CO2Emissions := 0.415 * totalPowerByPowerGenerationType.FireTotalPower

	var environmentProblemScore float64
	environmentProblemNumber := 0
	var environmentProblemComment CommentDetail
	var powerStabilityScore float64
	powerStabilityNumber := 0
	var powerStabilityComment CommentDetail
	var infrastructureComfortScore float64
	infrastructureComfortNumber := 0
	var infrastructureComfortComment CommentDetail

	// 環境問題の苦情人数とスコア
	if CO2Emissions > 20.0 && CO2Emissions <= 30.0 {
		environmentProblemNumber = 25
	} else if CO2Emissions > 30.0 && CO2Emissions <= 40.0 {
		environmentProblemNumber = 50
	} else if CO2Emissions > 40.0 && CO2Emissions <= 50.0 {
		environmentProblemNumber = 75
	}

	if totalPowerByPowerGenerationType.WindTotalPower >= 50.0 {
		environmentProblemNumber += 25
	}
	environmentProblemScore = (300.0 - float64(environmentProblemNumber)) / 3

	if environmentProblemScore >= 80 {
		environmentProblemComment = CommentDetail{
			Text:      "The air in the city is clean and fresh!",
			Sentiment: "positive",
		}
	} else if environmentProblemScore >= 50 {
		environmentProblemComment = CommentDetail{
			Text:      "The air might be getting a little polluted...",
			Sentiment: "positive",
		}
	} else {
		environmentProblemComment = CommentDetail{
			Text:      "The air is polluted and it's hard to live here...",
			Sentiment: "negative",
		}
	}

	// 電力の安定性に関する苦情人数とスコア
	if blackoutCount == 1 {
		powerStabilityNumber = 50
	} else if blackoutCount == 2 {
		powerStabilityNumber = 75
	} else if blackoutCount >= 3 {
		powerStabilityNumber = 100
	}
	powerStabilityScore = (300.0 - float64(powerStabilityNumber)) / 3

	if powerStabilityScore >= 80 {
		powerStabilityComment = CommentDetail{
			Text:      "The power supply is stable and comfortable!",
			Sentiment: "positive",
		}
	} else if powerStabilityScore >= 50 {
		powerStabilityComment = CommentDetail{
			Text:      "The power supply might be a bit unstable...",
			Sentiment: "positive",
		}
	} else {
		powerStabilityComment = CommentDetail{
			Text:      "The power is unstable and it's hard to live here...",
			Sentiment: "negative",
		}
	}

	// インフラの快適さに関する苦情人数とスコア
	if currentWorldState.State.LightLitPercent == 0 {
		infrastructureComfortNumber += 100 / 6
	}
	if !currentWorldState.State.IsTrainEnabled {
		infrastructureComfortNumber += 100 / 6
	}
	if currentWorldState.State.FactoryLitPercent == 0 {
		infrastructureComfortNumber += 100 / 6
	}
	if currentWorldState.State.HouseLitPercent > 0 {
		infrastructureComfortNumber += 100 / 6
	}
	if currentWorldState.State.FacilityLitPercent > 0 {
		infrastructureComfortNumber += 100 / 6
	}
	infrastructureComfortScore = (300.0 - float64(infrastructureComfortNumber)) / 3

	if infrastructureComfortScore >= 80 {
		infrastructureComfortComment = CommentDetail{
			Text:      "The city's infrastructure is great and comfortable!",
			Sentiment: "positive",
		}
	} else if infrastructureComfortScore >= 50 {
		infrastructureComfortComment = CommentDetail{
			Text:      "It would be nice if the infrastructure improved a bit more.",
			Sentiment: "positive",
		}
	} else {
		infrastructureComfortComment = CommentDetail{
			Text:      "The infrastructure is poor and it's hard to live here...",
			Sentiment: "negative",
		}
	}

	CO2ReductionAmount := 0.415 * renewableEnergyTotalPower

	happinessDetail := HappinessDetail{
		EnvironmentProblemScore:      float64(environmentProblemScore),
		EnvironmentProblemNumber:     environmentProblemNumber,
		EnvironmentProblemComment:    environmentProblemComment,
		PowerStabilityScore:          float64(powerStabilityScore),
		PowerStabilityNumber:         powerStabilityNumber,
		PowerStabilityComment:        powerStabilityComment,
		InfrastructureComfortScore:   float64(infrastructureComfortScore),
		InfrastructureComfortNumber:  infrastructureComfortNumber,
		InfrastructureComfortComment: infrastructureComfortComment,
	}

	gameResult := GameResult{
		TotalPowerGeneration:                          totalPower,
		HydrogenMaximumInstantaneousPowerGeneration:   maxPower.Hydrogen,
		WindMaximumInstantaneousPowerGeneration:       maxPower.Wind,
		SolarMaximumInstantaneousPowerGeneration:      maxPower.Solar,
		GeothermalMaximumInstantaneousPowerGeneration: maxPower.Geothermal,

		GeothermalTotalPower: totalPowerByPowerGenerationType.GeothermalTotalPower,
		FireTotalPower:       totalPowerByPowerGenerationType.FireTotalPower,
		WindTotalPower:       totalPowerByPowerGenerationType.WindTotalPower,
		SolarTotalPower:      totalPowerByPowerGenerationType.SolarTotalPower,
		HydrogenTotalPower:   totalPowerByPowerGenerationType.HydrogenTotalPower,

		CO2ReductionAmount: CO2ReductionAmount,
		Happiness:          happinessDetail,
		VillagersTexts: model.GenerateVillagersTexts(
			currentWorldState.State.HouseLitPercent,
			currentWorldState.State.FacilityLitPercent,
			currentWorldState.State.LightLitPercent,
			currentWorldState.State.FactoryLitPercent,
			currentWorldState.State.IsTrainEnabled,
			currentWorldState.FirePower,
		),
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
