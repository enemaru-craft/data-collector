package router

import (
	"context"
	"data-manager/controller"

	"github.com/aws/aws-lambda-go/events"
)

func Route(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	method := req.RequestContext.HTTP.Method
	path := req.RawPath

	if method == "POST" && path == "/register-new-power-generation-module" {
		return controller.RegisterNewPowerGenerationModuleHandler(ctx, req)
	}

	return events.APIGatewayV2HTTPResponse{
		StatusCode: 400,
		Body:       "Invalid method or path",
	}, nil
}
