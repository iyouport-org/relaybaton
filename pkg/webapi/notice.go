package webapi

import "github.com/iyouport-org/relaybaton/pkg/model"

type PostNoticeRequest struct {
	Title string `json:"title" validate:"required"`
	Text  string `json:"text" `
}

type Notice struct {
	ID    uint   `json:"id" validate:"required"`
	Title string `json:"title" validate:"required"`
	Text  string `json:"text" `
}

func GetNotice(notice model.Notice) Notice {
	return Notice{
		ID:    notice.ID,
		Title: notice.Title,
		Text:  notice.Text,
	}
}

func GetNotices(notices []model.Notice) []Notice {
	ret := make([]Notice, len(notices))
	for k, v := range notices {
		ret[k] = GetNotice(v)
	}
	return ret
}
