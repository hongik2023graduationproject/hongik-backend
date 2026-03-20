package service

import (
	"log"
	"sync"
	"time"

	"hongik-backend/model"

	"github.com/google/uuid"
)

const shareTTL = 24 * time.Hour

type Store struct {
	mu       sync.RWMutex
	snippets map[string]model.Snippet
	shares   map[string]model.SharedCode
}

func NewStore() *Store {
	s := &Store{
		snippets: make(map[string]model.Snippet),
		shares:   make(map[string]model.SharedCode),
	}
	s.seedExamples()
	go s.cleanupExpiredShares()
	return s
}

func (s *Store) seedExamples() {
	examples := []model.Snippet{
		{
			Title:       "안녕하세요",
			Description: "기본 출력 예제",
			Code:        "출력(\"안녕하세요, 세상!\")",
		},
		{
			Title:       "변수와 연산",
			Description: "변수 선언과 사칙연산 예제",
			Code:        "[정수] 가 = 10\n[정수] 나 = 20\n[정수] 합 = 가 + 나\n출력(합)",
		},
		{
			Title:       "조건문",
			Description: "만약/아니면 조건문 예제",
			Code:        "[정수] 점수 = 85\n\n만약 점수 >= 90 라면:\n    출력(\"A학점\")\n아니면:\n    출력(\"B학점\")",
		},
		{
			Title:       "함수 정의",
			Description: "함수 선언과 호출 예제",
			Code:        "함수: [정수]가, [정수]나 더하기 -> [정수]:\n    리턴 가 + 나\n\n[정수] 결과 = :(3, 5)더하기\n출력(결과)",
		},
		{
			Title:       "배열",
			Description: "배열 생성과 순회 예제",
			Code:        "[배열] 과일 = [\"사과\", \"바나나\", \"포도\"]\n출력(길이(과일))\n출력(과일[0])",
		},
	}

	for i := range examples {
		id := uuid.New().String()
		now := time.Now()
		examples[i].ID = id
		examples[i].CreatedAt = now
		examples[i].UpdatedAt = now
		s.snippets[id] = examples[i]
	}
}

func (s *Store) cleanupExpiredShares() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now().Unix()
		removed := 0
		for token, shared := range s.shares {
			if shared.ExpiresAt > 0 && shared.ExpiresAt < now {
				delete(s.shares, token)
				removed++
			}
		}
		s.mu.Unlock()
		if removed > 0 {
			log.Printf("Cleaned up %d expired share(s)", removed)
		}
	}
}

// Snippet operations

func (s *Store) ListSnippets() []model.Snippet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]model.Snippet, 0, len(s.snippets))
	for _, sn := range s.snippets {
		result = append(result, sn)
	}
	return result
}

func (s *Store) GetSnippet(id string) (model.Snippet, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sn, ok := s.snippets[id]
	return sn, ok
}

func (s *Store) CreateSnippet(req model.CreateSnippetRequest) model.Snippet {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	sn := model.Snippet{
		ID:          uuid.New().String(),
		Title:       req.Title,
		Code:        req.Code,
		Description: req.Description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.snippets[sn.ID] = sn
	return sn
}

// Share operations

func (s *Store) CreateShare(req model.ShareRequest) model.SharedCode {
	s.mu.Lock()
	defer s.mu.Unlock()

	token := uuid.New().String()
	shared := model.SharedCode{
		Token:     token,
		Code:      req.Code,
		Title:     req.Title,
		CreatedAt: time.Now().Format(time.RFC3339),
		ExpiresAt: time.Now().Add(shareTTL).Unix(),
	}
	s.shares[token] = shared
	return shared
}

func (s *Store) GetShare(token string) (model.SharedCode, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	shared, ok := s.shares[token]
	if !ok {
		return shared, false
	}
	if shared.ExpiresAt > 0 && time.Now().Unix() > shared.ExpiresAt {
		return model.SharedCode{}, false
	}
	return shared, true
}
