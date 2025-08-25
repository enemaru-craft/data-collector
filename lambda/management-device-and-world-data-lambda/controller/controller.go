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
	SessionID  string `json:"session_id"`
	DeviceID   string `json:"device_id"`
	DeviceType string `json:"device_type"`
}

type ManagementController struct {
	repo model.ManagementRepositoryInterface
}

func NewManagementController(repo model.ManagementRepositoryInterface) *ManagementController {
	return &ManagementController{repo: repo}
}

func (c *ManagementController) RegisterNewPowerGenerationModuleHandler(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
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

	var requestBody RegistrationNewPowerGenerationModuleRequestBody
	if err := json.Unmarshal([]byte(req.Body), &requestBody); err != nil {
		tx.Rollback()
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 400,
			Body:       fmt.Sprintf("Invalid request body: %v", err),
		}, nil
	}

	if requestBody.SessionID == "" || requestBody.DeviceID == "" || requestBody.DeviceType == "" {
		tx.Rollback()
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 400,
			Body:       "Missing required fields",
		}, nil
	}

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

	err = c.repo.CheckDeviceNotExists(ctx, tx, requestBody.DeviceID)
	if err != nil {
		tx.Rollback()
		var lErr *custmerr.LogicalErr
		var tErr *custmerr.TechnicalErr
		switch {
		case errors.As(err, &lErr):
			return events.APIGatewayV2HTTPResponse{
				StatusCode: 404,
				Body:       fmt.Sprintf("Device not found: %v", err),
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
	if deviceType = req.QueryStringParameters["device-type"]; deviceType == "" {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 400,
			Body:       "Missing required query parameter: device-type",
		}, errors.New("missing required query parameter: device-type")
	}

	var sessionId string
	if sessionId = req.QueryStringParameters["session-id"]; sessionId == "" {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 400,
			Body:       "Missing required query parameter: session-id",
		}, errors.New("missing required query parameter: session-id")
	}

	tx, err := c.repo.BeginTx(ctx, nil)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf("Failed to begin transaction: %v", err),
		}, errors.New("failed to begin transaction")
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
		}
	}()

	latestPowerData, err := c.repo.GetLatestPowerData(ctx, tx, deviceType, sessionId)
	if err != nil {
		tx.Rollback()
		var lErr *custmerr.LogicalErr
		var tErr *custmerr.TechnicalErr
		switch {
		case errors.As(err, &lErr):
			return events.APIGatewayV2HTTPResponse{
				StatusCode: 404,
				Body:       fmt.Sprintf("No power data found for device type %s: %v", deviceType, err),
			}, fmt.Errorf("no power data found for device type %s : %v", deviceType, err)

		case errors.As(err, &tErr):
			return events.APIGatewayV2HTTPResponse{
				StatusCode: 500,
				Body:       fmt.Sprintf("Technical error occurred: %v", err),
			}, fmt.Errorf("technical error occurred: %v", err)
		}
	}

	bodyBytes, err := json.Marshal(map[string]float32{"latest_power": latestPowerData})
	if err != nil {
		tx.Rollback()
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf("Failed to marshal response: %v", err),
		}, fmt.Errorf("failed to marshal response: %v", err)
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
