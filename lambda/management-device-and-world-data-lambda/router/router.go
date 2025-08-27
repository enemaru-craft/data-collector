package router

import (
	"context"
	"data-manager/controller"
	"data-manager/model"
	"database/sql"

	"github.com/aws/aws-lambda-go/events"
)

type Router struct {
	db *sql.DB
}

func NewRouter(db *sql.DB) *Router {
	return &Router{
		db: db,
	}
}

func (r *Router) Route(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	method := req.RequestContext.HTTP.Method
	path := req.RequestContext.HTTP.Path

	repo := model.NewManagementRepository(r.db)
	ctr := controller.NewManagementController(repo)

	if method == "POST" && path == "/register-new-power-generation-module" {
		return ctr.RegisterNewPowerGenerationModuleHandler(ctx, req)
	}

	if method == "GET" && path == "/get-latest-power" {
		return ctr.GetLatestPower(ctx, req)
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

	return events.APIGatewayV2HTTPResponse{
		StatusCode: 400,
		Body:       "Invalid method or path",
	}, nil
}
