package webapi

type PostMetaRequest struct {
}

type PostMetaResponse struct {
}

type DeleteMetaRequest struct {
}

type DeleteMetaResponse struct {
}

type PutMetaRequest struct {
}

type PutMetaResponse struct {
}

type Meta struct {
	Title string `json:"title" validate:"required"`
	Desc  string `json:"desc"`
}
