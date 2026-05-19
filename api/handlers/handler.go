package handlers

import (
	"hongik-backend/service"
)

// Handler는 데이터 도메인(스니펫/공유)과 메타 조회(builtins/syntax/health)를 담당한다.
// 사용자 코드 실행은 WASM-only 전환으로 클라이언트 측으로 이전되어 더 이상 인터프리터 의존성을 가지지 않는다.
type Handler struct {
	store service.Store
	cache *service.Cache
}

func New(store service.Store, cache *service.Cache) *Handler {
	return &Handler{
		store: store,
		cache: cache,
	}
}
