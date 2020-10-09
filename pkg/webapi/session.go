package webapi

type PostSessionRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"base64,required"`
	//Captcha  string `json:"captcha" validate:"numeric,gte=0,lte=99999,required"`
}

type PostSessionResponse struct {
	OK       bool   `json:"ok"`
	ErrorMsg string `json:"errorMsg"`
}

type DeleteSessionRequest struct {
}

type DeleteSessionResponse struct {
}

type PutSessionRequest struct {
}

type PutSessionResponse struct {
}
