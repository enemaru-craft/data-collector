package main

import (
	"context"

	"github.com/aws/aws-lambda-go/lambda"
)

func handler(ctx context.Context, event IoTEvent) (string, error) {
	return "OK", nil
}

func main() {
	lambda.Start(handler)
}
