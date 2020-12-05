package webapi

import (
	"time"

	"github.com/iyouport-org/relaybaton/pkg/model"
)

type PostUserRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"base64,required"`
	Captcha  string `json:"captcha" validate:"numeric,gte=0,lte=999999,required"`
}

type PostUserResponse struct {
	OK       bool   `json:"ok"`
	ErrorMsg string `json:"errorMsg"`
}

type User struct {
	ID                 uint      `json:"id" validate:"required"`
	Username           string    `json:"username" validate:"required"`
	Role               uint      `json:"role" validate:"required"`
	PlanID             uint      `json:"plan_id" validate:"required"`
	PlanName           string    `json:"plan_name" validate:"required"`
	PlanBandwidthLimit uint      `json:"plan_bandwidth_limit" validate:"required"`
	PlanTrafficLimit   uint      `json:"plan_traffic_limit" validate:"required"`
	TrafficUsed        uint      `json:"traffic_used" validate:"required"`
	PlanStart          time.Time `json:"plan_start" validate:"required"`
	PlanReset          time.Time `json:"plan_reset" validate:"required"`
	PlanEnd            time.Time `json:"plan_end" validate:"required"`
}

func GetUsers(users []model.User) []User {
	ret := make([]User, len(users))
	for k, v := range users {
		ret[k] = User{
			ID:                 v.ID,
			Username:           v.Username,
			Role:               v.Role,
			PlanID:             v.PlanID,
			PlanName:           v.Plan.Name,
			PlanBandwidthLimit: v.Plan.BandwidthLimit,
			PlanTrafficLimit:   v.Plan.TrafficLimit,
			TrafficUsed:        v.TrafficUsed,
			PlanStart:          v.PlanStart,
			PlanReset:          v.PlanReset,
			PlanEnd:            v.PlanEnd,
		}
	}
	return ret
}
