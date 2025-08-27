package controller_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/aws/aws-lambda-go/events"

	"data-manager/controller"
)

type mockRepo struct{}

func (m *mockRepo) CreateSessionIfNotExists(ctx context.Context, tx *sql.Tx, sessionID string) error {
	return nil
}

func (m *mockRepo) CheckDeviceNotExists(ctx context.Context, tx *sql.Tx, deviceID string) error {
	return nil
}

func (m *mockRepo) RegisterNewPowerGenerationModule(ctx context.Context, tx *sql.Tx, sessionID, deviceID, deviceType string) error {
	return nil
}

func (m *mockRepo) GetLatestPowerData(ctx context.Context, tx *sql.Tx, deviceType string, sessionId string) (float32, error) {
	// float32のゼロ値とnilを返す
	return 0, nil
}

func (m *mockRepo) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	// sql.Txのゼロ値とnilを返す
	return nil, nil
}

func (m *mockRepo) CreateNewWorldIfNotExists(ctx context.Context, tx *sql.Tx, sessionId string) error {
	return nil
}

func TestGetLatestPower_ReturnErrorIfReceivedInvalidJSON(t *testing.T) {
	ctr := controller.NewManagementController(&mockRepo{})

	t.Run("device-typeがかけているとエラーが返ってくる", func(t *testing.T) {
		req := events.APIGatewayV2HTTPRequest{
			QueryStringParameters: map[string]string{
				"session-id": "1",
			},
		}

		response, err := ctr.GetLatestPower(context.Background(), req)

		if err == nil {
			t.Fatal("クエリパラメータが欠けている場合はエラーを返す必要がある")
		}

		if response.StatusCode != 400 {
			t.Fatalf("クエリパラメータが欠けている場合はステータスコード400を返す必要があるが%dが返ってきています｡", response.StatusCode)
		}
	})

	t.Run("session-idがかけているとエラーが返ってくる", func(t *testing.T) {
		req := events.APIGatewayV2HTTPRequest{
			QueryStringParameters: map[string]string{
				"device-type": "geothermal",
			},
		}

		response, err := ctr.GetLatestPower(context.Background(), req)

		if err == nil {
			t.Fatal("クエリパラメータが欠けている場合はエラーを返す必要がある")
		}

		if response.StatusCode != 400 {
			t.Fatalf("クエリパラメータが欠けている場合はステータスコード400を返す必要があるが%dが返ってきています｡", response.StatusCode)
		}
	})

	t.Run("クエリパラメータがどちらもかけているとエラーが返ってくる", func(t *testing.T) {
		req := events.APIGatewayV2HTTPRequest{
			QueryStringParameters: map[string]string{},
		}

		response, err := ctr.GetLatestPower(context.Background(), req)

		if err == nil {
			t.Fatal("クエリパラメータが欠けている場合はエラーを返す必要がある")
		}

		if response.StatusCode != 400 {
			t.Fatalf("クエリパラメータが欠けている場合はステータスコード400を返す必要があるが%dが返ってきています｡", response.StatusCode)
		}
	})

}
