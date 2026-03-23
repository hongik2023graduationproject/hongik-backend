package service

import (
	"fmt"

	"hongik-backend/model"
)

// CachedStore wraps a Store with Redis caching on read operations
// and cache invalidation on write operations.
type CachedStore struct {
	inner Store
	cache *Cache
}

// NewCachedStore creates a caching decorator around an existing Store.
// If cache is nil, all operations pass through to the inner store directly.
func NewCachedStore(inner Store, cache *Cache) Store {
	if cache == nil {
		return inner
	}
	return &CachedStore{inner: inner, cache: cache}
}

// --- Read operations (cached) ---

func (s *CachedStore) ListSnippets(page, limit int) ([]model.Snippet, int) {
	type listResult struct {
		Snippets []model.Snippet `json:"snippets"`
		Total    int             `json:"total"`
	}

	key := fmt.Sprintf("snippets:list:%d:%d", page, limit)
	var cached listResult
	if s.cache.Get(key, &cached) {
		return cached.Snippets, cached.Total
	}

	snippets, total := s.inner.ListSnippets(page, limit)
	s.cache.Set(key, listResult{Snippets: snippets, Total: total})
	return snippets, total
}

func (s *CachedStore) SearchSnippets(query string, page, limit int) ([]model.Snippet, int) {
	type listResult struct {
		Snippets []model.Snippet `json:"snippets"`
		Total    int             `json:"total"`
	}

	key := fmt.Sprintf("snippets:search:%s:%d:%d", query, page, limit)
	var cached listResult
	if s.cache.Get(key, &cached) {
		return cached.Snippets, cached.Total
	}

	snippets, total := s.inner.SearchSnippets(query, page, limit)
	s.cache.Set(key, listResult{Snippets: snippets, Total: total})
	return snippets, total
}

func (s *CachedStore) GetSnippet(id string) (model.Snippet, bool) {
	key := "snippet:" + id
	var cached model.Snippet
	if s.cache.Get(key, &cached) {
		return cached, true
	}

	snippet, ok := s.inner.GetSnippet(id)
	if ok {
		s.cache.Set(key, snippet)
	}
	return snippet, ok
}

func (s *CachedStore) GetShare(token string) (model.SharedCode, bool) {
	key := "share:" + token
	var cached model.SharedCode
	if s.cache.Get(key, &cached) {
		return cached, true
	}

	shared, ok := s.inner.GetShare(token)
	if ok {
		s.cache.Set(key, shared)
	}
	return shared, ok
}

// --- Write operations (invalidate cache) ---

func (s *CachedStore) CreateSnippet(req model.CreateSnippetRequest, userID string) model.Snippet {
	snippet := s.inner.CreateSnippet(req, userID)
	s.cache.DeleteByPrefix("snippets:list:")
	s.cache.DeleteByPrefix("snippets:search:")
	return snippet
}

func (s *CachedStore) UpdateSnippet(id string, req model.UpdateSnippetRequest, userID string) (model.Snippet, bool, bool) {
	snippet, found, owned := s.inner.UpdateSnippet(id, req, userID)
	if found && owned {
		s.cache.Delete("snippet:" + id)
		s.cache.DeleteByPrefix("snippets:list:")
		s.cache.DeleteByPrefix("snippets:search:")
	}
	return snippet, found, owned
}

func (s *CachedStore) DeleteSnippet(id string, userID string) (bool, bool) {
	found, owned := s.inner.DeleteSnippet(id, userID)
	if found && owned {
		s.cache.Delete("snippet:" + id)
		s.cache.DeleteByPrefix("snippets:list:")
		s.cache.DeleteByPrefix("snippets:search:")
	}
	return found, owned
}

func (s *CachedStore) ForkSnippet(id string, userID string) (model.Snippet, bool) {
	forked, ok := s.inner.ForkSnippet(id, userID)
	if ok {
		s.cache.DeleteByPrefix("snippets:list:")
		s.cache.DeleteByPrefix("snippets:search:")
	}
	return forked, ok
}

func (s *CachedStore) CreateShare(req model.ShareRequest) model.SharedCode {
	return s.inner.CreateShare(req)
}

// --- Pass-through operations (no caching for auth) ---

func (s *CachedStore) CreateUser(username, password string) (model.User, error) {
	return s.inner.CreateUser(username, password)
}

func (s *CachedStore) AuthenticateUser(username, password string) (model.User, error) {
	return s.inner.AuthenticateUser(username, password)
}
