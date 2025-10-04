package router

import (
	"context"
	"data-manager/controller"
	"data-manager/model"
	"database/sql"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type Router struct {
	db *sql.DB
	dc *dynamodb.Client
}

func NewRouter(db *sql.DB, dc *dynamodb.Client) *Router {
	return &Router{
		db: db,
		dc: dc,
	}
}

func (r *Router) Route(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	method := req.RequestContext.HTTP.Method
	path := req.RequestContext.HTTP.Path

	repo := model.NewManagementRepository(r.db, r.dc)
	ctr := controller.NewManagementController(repo)

	if method == "POST" && path == "/register-new-power-generation-module" {
		return ctr.RegisterNewPowerGenerationModuleHandler(ctx, req)
	}

	if method == "GET" && path == "/get-latest-power" {
		return ctr.GetLatestPower(ctx, req)
	}

	if method == "GET" && path == "/get-latest-multiple-device-power" {
		return ctr.GetLatestMultipleDevicePower(ctx, req)
	}

	if method == "POST" && path == "/turn-on-equipment" {
		return ctr.TurnOnEquipment(ctx, req)
	}

	if method == "POST" && path == "/turn-off-equipment" {
		return ctr.TurnOffEquipment(ctx, req)
	}

	if method == "POST" && path == "/get-current-world-state" {
		return ctr.GetCurrentWorldState(ctx, req)
	}

	if method == "GET" && path == "/get-power-history" {
		return ctr.GetPowerHistory(ctx, req)
	}

	if method == "GET" && path == "/get-game-result" {
		return ctr.GetGameResult(ctx, req)
	}

	return events.APIGatewayV2HTTPResponse{
		StatusCode: 400,
		Body:       "Invalid method or path",
	}, nil
}
