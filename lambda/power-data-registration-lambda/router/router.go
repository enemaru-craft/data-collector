package router

import (
	"context"
	"database/sql"
	"encoding/json"
	"power-manager/controller"
	"power-manager/model"
)

type Topic struct {
	Topic string `json:"topic"`
}

type Router struct {
	db *sql.DB
}

func NewRouter(db *sql.DB) *Router {
	return &Router{
		db: db,
	}
}

func (r *Router) Route(ctx context.Context, event json.RawMessage) (string, error) {
	var topic Topic
	if err := json.Unmarshal(event, &topic); err != nil {
		return "Topic extraction failed: ", err
	}

	repo := model.NewLogRepository(r.db)
	ctr := controller.NewLogController(repo)

	if topic.Topic == "register/geothermal" {
		return ctr.RegisterGeothermalPower(ctx, event)
	}

	if topic.Topic == "register/solar" {
		return ctr.RegisterSolarPower(ctx, event)
	}

	if topic.Topic == "register/wind" {
		return ctr.RegisterWindPower(ctx, event)
	}

	if topic.Topic == "register/hydrogen" {
		return ctr.RegisterHydrogenPower(ctx, event)
	}

	if topic.Topic == "register/hand-crank" {
		return ctr.RegisterHandCrankPower(ctx, event)
	}

	return "Invalid topic", nil
}
