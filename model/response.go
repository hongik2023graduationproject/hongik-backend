package model

type ExecuteRequest struct {
	Code    string `json:"code" binding:"required"`
	Timeout int    `json:"timeout"` // seconds, 0 = use default (5s)
}

type ExecuteResponse struct {
	Status          string `json:"status"`
	Output          string `json:"output,omitempty"`
	Error           string `json:"error,omitempty"`
	ExecutionTimeMs int64  `json:"execution_time_ms"`
}

type ShareRequest struct {
	Code  string `json:"code" binding:"required"`
	Title string `json:"title"`
}

type ShareResponse struct {
	Token string `json:"token"`
}

type SharedCode struct {
	Token     string `json:"token"`
	Code      string `json:"code"`
	Title     string `json:"title"`
	CreatedAt string `json:"created_at"`
	ExpiresAt int64  `json:"-"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
