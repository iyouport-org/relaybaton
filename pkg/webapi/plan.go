package webapi

import "github.com/iyouport-org/relaybaton/pkg/model"

type PostPlanRequest struct {
	Name           string `json:"name" validate:"required"`
	BandwidthLimit uint   `json:"bandwidth_limit" validate:"required"`
	TrafficLimit   uint   `json:"traffic_limit" validate:"required"`
}

type Plan struct {
	ID             uint   `json:"id" validate:"required"`
	Name           string `json:"name" validate:"required"`
	BandwidthLimit uint   `json:"bandwidth_limit" validate:"required"`
	TrafficLimit   uint   `json:"traffic_limit" validate:"required"`
}

func GetPlan(plan model.Plan) Plan {
	return Plan{
		ID:             plan.ID,
		Name:           plan.Name,
		BandwidthLimit: plan.BandwidthLimit,
		TrafficLimit:   plan.TrafficLimit,
	}
}

func GetPlans(plans []model.Plan) []Plan {
	ret := make([]Plan, len(plans))
	for k, v := range plans {
		ret[k] = GetPlan(v)
	}
	return ret
}
