package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type Message struct {
	ClientID  string  `json:"client_id"`
	Timestamp int64   `json:"timestamp"`
	Temp      float64 `json:"temp"`
}

var (
	dbClient  *dynamodb.Client
	tableName = "mqtt_test_table"
)

func init() {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		panic(fmt.Sprintf("Unable to load SDK config: %v", err))
	}
	dbClient = dynamodb.NewFromConfig(cfg)
}

func handler(ctx context.Context, msg Message) (string, error) {
	item := map[string]types.AttributeValue{
		"client_id": &types.AttributeValueMemberS{Value: msg.ClientID},
		"timestamp": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", msg.Timestamp)},
		"temp":      &types.AttributeValueMemberN{Value: fmt.Sprintf("%f", msg.Temp)},
	}

	_, err := dbClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item:      item,
	})
	if err != nil {
		return "", fmt.Errorf("failed to put item: %w", err)
	}

	return "Item written successfully", nil
}

func main() {
	lambda.Start(handler)
}
