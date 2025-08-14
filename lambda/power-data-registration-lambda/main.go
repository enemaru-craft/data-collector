package main

import (
	"context"
	"encoding/json"
	"power-manager/model"
	"power-manager/router"

	"github.com/aws/aws-lambda-go/lambda"
)

func handler(ctx context.Context, event json.RawMessage) (string, error) {

	if err := model.InitDB(); err != nil {
		return "Failed to Initialize Database: " + err.Error(), err
	}

	return router.Route(ctx, event)
}

func main() {
	lambda.Start(handler)
}
