package router

import (
	"context"
	"encoding/json"
	"power-manager/controller"
)

type Topic struct {
	Topic string `json:"topic"`
}

func Route(ctx context.Context, event json.RawMessage) (string, error) {
	var topic Topic
	if err := json.Unmarshal(event, &topic); err != nil {
		return "Topic extraction failed: ", err
	}

	if topic.Topic == "register/geothermal" {
		return controller.RegisterGeothermalPower(ctx, event)
	}

	if topic.Topic == "register/solar" {
		return controller.RegisterSolarPower(ctx, event)
	}

	if topic.Topic == "register/wind" {
		return controller.RegisterWindPower(ctx, event)
	}

	if topic.Topic == "register/hydrogen" {
		return controller.RegisterHydrogenPower(ctx, event)
	}

	if topic.Topic == "register/hand-crank" {
		return controller.RegisterHandCrankPower(ctx, event)
	}

	return "Invalid topic", nil
}
