package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// IoTからのイベント構造体（ペイロードの内容＋topic）
type IoTEvent struct {
	ClientID  string  `json:"client_id"`
	Timestamp int64   `json:"timestamp"`
	Temp      float64 `json:"temp"`
	Topic     string  `json:"topic"`
}

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

func handler(ctx context.Context, event IoTEvent) (string, error) {
	var topicLabel string

	// トピックごとに処理分岐（分かりやすく）
	switch {
	case strings.Contains(event.Topic, "test/topic/1"):
		topicLabel = "Topic 1"
	case strings.Contains(event.Topic, "test/topic/2"):
		topicLabel = "Topic 2"
	default:
		topicLabel = "Other Topic"
	}

	// 分岐状況のログを出す
	fmt.Printf("Processing message from %s\n", topicLabel)

	// DynamoDBに共通テーブルへ書き込み
	item := map[string]types.AttributeValue{
		"client_id":   &types.AttributeValueMemberS{Value: event.ClientID},
		"timestamp":   &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", event.Timestamp)},
		"temp":        &types.AttributeValueMemberN{Value: fmt.Sprintf("%f", event.Temp)},
		"topic":       &types.AttributeValueMemberS{Value: event.Topic},
		"topic_label": &types.AttributeValueMemberS{Value: topicLabel},
	}

	_, err := dbClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item:      item,
	})
	if err != nil {
		return "", fmt.Errorf("failed to put item: %w", err)
	}

	return fmt.Sprintf("Item written to %s successfully (%s)", tableName, topicLabel), nil
}

func main() {
	lambda.Start(handler)
}
