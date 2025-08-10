package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var (
	dbClient  *dynamodb.Client
	tableName = "mqtt_test_table" // 共通テーブル名
)

func init() {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		panic(fmt.Sprintf("unable to load SDK config: %v", err))
	}
	dbClient = dynamodb.NewFromConfig(cfg)
}

func handler(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	path := req.RawPath // 例: /topic1, /topic2
	var topicLabel string

	switch path {
	case "/topic1":
		topicLabel = "Topic 1"
	case "/topic2":
		topicLabel = "Topic 2"
	default:
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 400,
			Body:       "Invalid path",
		}, nil
	}

	// DynamoDBからtopic_labelで絞り込み取得
	// ※topic_labelがGSIにあるならQuery、それ以外はScanを使う（例としてScanで書きます）
	out, err := dbClient.Scan(ctx, &dynamodb.ScanInput{
		TableName:        aws.String(tableName),
		FilterExpression: aws.String("topic_label = :label"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":label": &types.AttributeValueMemberS{Value: topicLabel},
		},
	})
	if err != nil {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf("Failed to scan items: %v", err),
		}, nil
	}

	// 取得アイテムをJSONで返す
	itemsJson, err := json.Marshal(out.Items)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf("Failed to marshal items: %v", err),
		}, nil
	}

	return events.APIGatewayV2HTTPResponse{
		StatusCode: 200,
		Body:       string(itemsJson),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}, nil
}

func main() {
	lambda.Start(handler)
}
