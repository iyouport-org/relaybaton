package webapi

import (
	"time"

	"github.com/iyouport-org/relaybaton/pkg/model"
)

type PostLogRequest struct {
}

type PostLogResponse struct {
}

type DeleteLogRequest struct {
}

type DeleteLogResponse struct {
}

type PutLogRequest struct {
}

type PutLogResponse struct {
}

type GetLogResponse struct {
	ID       uint      `json:"id" validate:"required"`
	CreateAt time.Time `json:"created_at" validate:"required"`
	Level    uint32    `json:"level" validate:"required"`
	Func     string    `json:"func"`
	File     string    `json:"file" `
	Msg      string    `json:"msg" `
	Stack    string    `json:"stack"`
	Fields   string    `json:"fields" `
}

func GetLogs(logs []model.Log) []GetLogResponse {
	ret := make([]GetLogResponse, len(logs))
	for k, v := range logs {
		ret[k] = GetLogResponse{
			ID:       v.ID,
			CreateAt: v.CreatedAt,
			Level:    v.Level,
			Func:     v.Func,
			File:     v.File,
			Msg:      v.Msg,
			Stack:    v.Stack,
			Fields:   v.Fields,
		}
	}
	return ret
}
