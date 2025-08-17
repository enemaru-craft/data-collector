package main

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-lambda-go/lambda"
)

func handler(ctx context.Context, event json.RawMessage) (string, error) {
	return "OK", nil
}

func main() {
	lambda.Start(handler)
}
