package router

import (
	"context"
	"encoding/json"
	"power-manager/controller"
	"power-manager/model"
)

type Topic struct {
	Topic string `json:"topic"`
}

func Route(ctx context.Context, event json.RawMessage) (string, error) {
	var topic Topic
	if err := json.Unmarshal(event, &topic); err != nil {
		return "Topic extraction failed: ", err
	}

	conn := model.GetConn()
	repo := model.NewLogRepository(conn)
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
