package service

import (
	"errors"
	"log"
	"sync"
	"time"

	"hongik-backend/model"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const shareTTL = 24 * time.Hour

type Store struct {
	mu       sync.RWMutex
	snippets map[string]model.Snippet
	shares   map[string]model.SharedCode
	users    map[string]model.User // keyed by user ID
	userByName map[string]string   // username -> user ID
}

func NewStore() *Store {
	s := &Store{
		snippets:   make(map[string]model.Snippet),
		shares:     make(map[string]model.SharedCode),
		users:      make(map[string]model.User),
		userByName: make(map[string]string),
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
			Code:        "정수 가 = 10\n정수 나 = 20\n정수 합 = 가 + 나\n출력(합)",
		},
		{
			Title:       "조건문",
			Description: "만약/아니면 조건문 예제",
			Code:        "정수 점수 = 85\n\n만약 점수 >= 90 라면:\n    출력(\"A학점\")\n아니면:\n    출력(\"B학점\")",
		},
		{
			Title:       "함수 정의",
			Description: "함수 선언과 호출 예제",
			Code:        "함수: [정수]가, [정수]나 더하기 -> [정수]:\n    리턴 가 + 나\n\n출력(:(3, 5)더하기)",
		},
		{
			Title:       "배열",
			Description: "배열 생성과 순회 예제",
			Code:        "배열 과일 = [\"사과\", \"바나나\", \"포도\"]\n출력(길이(과일))\n출력(과일[0])",
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

func (s *Store) CreateSnippet(req model.CreateSnippetRequest, userID string) model.Snippet {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	sn := model.Snippet{
		ID:          uuid.New().String(),
		Title:       req.Title,
		Code:        req.Code,
		Description: req.Description,
		UserID:      userID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.snippets[sn.ID] = sn
	return sn
}

func (s *Store) UpdateSnippet(id string, req model.UpdateSnippetRequest, userID string) (model.Snippet, bool, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sn, ok := s.snippets[id]
	if !ok {
		return model.Snippet{}, false, false
	}

	// Check ownership: if snippet has a userID, only that user can update
	if sn.UserID != "" && sn.UserID != userID {
		return model.Snippet{}, true, false // exists but not owned
	}

	sn.Title = req.Title
	sn.Code = req.Code
	sn.Description = req.Description
	sn.UpdatedAt = time.Now()
	s.snippets[id] = sn
	return sn, true, true
}

func (s *Store) DeleteSnippet(id string, userID string) (bool, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sn, ok := s.snippets[id]
	if !ok {
		return false, false
	}

	// Check ownership: if snippet has a userID, only that user can delete
	if sn.UserID != "" && sn.UserID != userID {
		return true, false // exists but not owned
	}

	delete(s.snippets, id)
	return true, true
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

// User operations

var (
	ErrUsernameTaken = errors.New("사용자 이름이 이미 존재합니다")
	ErrUserNotFound  = errors.New("사용자를 찾을 수 없습니다")
	ErrInvalidPassword = errors.New("비밀번호가 일치하지 않습니다")
)

func (s *Store) CreateUser(username, password string) (model.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.userByName[username]; exists {
		return model.User{}, ErrUsernameTaken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return model.User{}, err
	}

	user := model.User{
		ID:           uuid.New().String(),
		Username:     username,
		PasswordHash: string(hash),
		CreatedAt:    time.Now(),
	}

	s.users[user.ID] = user
	s.userByName[username] = user.ID
	return user, nil
}

func (s *Store) AuthenticateUser(username, password string) (model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	userID, exists := s.userByName[username]
	if !exists {
		return model.User{}, ErrUserNotFound
	}

	user := s.users[userID]
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return model.User{}, ErrInvalidPassword
	}

	return user, nil
}
