package router

import (
	"context"
	"database/sql"
	"encoding/json"
	"power-manager/controller"
	"power-manager/model"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type Topic struct {
	Topic string `json:"topic"`
}

type Router struct {
	db *sql.DB
	dc *dynamodb.Client
}

func NewRouter(db *sql.DB, dc *dynamodb.Client) *Router {
	return &Router{
		db: db,
		dc: dc,
	}
}

func (r *Router) Route(ctx context.Context, event json.RawMessage) (string, error) {
	var topic Topic
	if err := json.Unmarshal(event, &topic); err != nil {
		return "Topic extraction failed: ", err
	}

	repo := model.NewLogRepository(r.db, r.dc)
	ctr := controller.NewLogController(repo)

	if topic.Topic == "register/power" {
		return ctr.RegisterPower(ctx, event)
	}

	return "Invalid topic", nil
}
