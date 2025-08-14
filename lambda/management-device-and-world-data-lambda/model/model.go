package model

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var (
	dbClient  *dynamodb.Client
	tableName = "mqtt_test_table"
)

func init() {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		panic(fmt.Sprintf("unable to load SDK config: %v", err))
	}
	dbClient = dynamodb.NewFromConfig(cfg)
}

func FetchByTopicLabel(ctx context.Context, label string) ([]map[string]types.AttributeValue, error) {
	out, err := dbClient.Scan(ctx, &dynamodb.ScanInput{
		TableName:        aws.String(tableName),
		FilterExpression: aws.String("topic_label = :label"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":label": &types.AttributeValueMemberS{Value: label},
		},
	})
	if err != nil {
		return nil, err
	}
	return out.Items, nil
}
