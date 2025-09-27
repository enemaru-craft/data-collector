package main

import (
	"context"
	"encoding/json"
	"log"
	"power-manager/model"
	"power-manager/router"

	"github.com/aws/aws-lambda-go/lambda"
)

var r *router.Router

func handler(ctx context.Context, event json.RawMessage) (string, error) {
	return r.Route(ctx, event)
}

func init() {
	// Lambdaの特性上仕方なくここで初期化している｡もしいい方法があるなら変える｡
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
