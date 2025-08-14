package main

import (
	"context"
	"data-manager/model"
	"data-manager/router"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handler(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {

	if err := model.InitDB(); err != nil {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 500,
			Body:       "Failed to initialize database connection: " + err.Error(),
		}, nil
	}

	return router.Route(ctx, req)
}

func main() {
	lambda.Start(handler)
}
