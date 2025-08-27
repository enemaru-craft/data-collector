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
	db, err := model.InitDB()
	if err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}
	r = router.NewRouter(db)
}

func main() {
	lambda.Start(handler)
}
