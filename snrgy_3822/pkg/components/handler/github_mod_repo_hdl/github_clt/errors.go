package github_clt

type ResponseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (r *ResponseError) Error() string {
	return r.Message
}

func NewResponseError(c int, m string) *ResponseError {
	return &ResponseError{
		Code:    c,
		Message: m,
	}
}
