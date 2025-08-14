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

	return "Invalid topic", nil
}
