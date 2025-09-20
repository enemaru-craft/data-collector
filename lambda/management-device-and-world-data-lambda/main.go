package main

import (
	"context"
	"data-manager/model"
	"data-manager/router"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

var r *router.Router

func handler(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	return r.Route(ctx, req)
}

func init() {
	db, err := model.InitDB()
	if err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}

	dc, err := model.InitDynamoDB()
	if err != nil {
		log.Fatalf("DynamoDB initialization failed: %v", err)
	}

	r = router.NewRouter(db, dc)
}

func main() {
	lambda.Start(handler)
}
