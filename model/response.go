package model

// ExecuteRequest/Response 타입은 WASM-only 전환으로 백엔드에서 코드 실행이 사라져 제거되었다.

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
