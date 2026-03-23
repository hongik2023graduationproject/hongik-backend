package service

import (
	"testing"

	"hongik-backend/model"
)

// TestCachedStoreNilCachePassthrough verifies that NewCachedStore
// returns the inner store directly when cache is nil.
func TestCachedStoreNilCachePassthrough(t *testing.T) {
	inner := NewStore()
	store := NewCachedStore(inner, nil)

	// Should be the same pointer — no wrapping
	if store != inner {
		t.Fatal("expected inner store returned directly when cache is nil")
	}
}

// TestCachedStoreListSnippets verifies list passes through with nil cache.
func TestCachedStoreListSnippets(t *testing.T) {
	store := NewCachedStore(NewStore(), nil)

	snippets, total := store.ListSnippets(1, 10)
	if total != 5 {
		t.Fatalf("expected 5 seeded snippets, got %d", total)
	}
	if len(snippets) != 5 {
		t.Fatalf("expected 5 snippets on page 1, got %d", len(snippets))
	}
}

// TestCachedStoreGetSnippet verifies get passes through with nil cache.
func TestCachedStoreGetSnippet(t *testing.T) {
	store := NewCachedStore(NewStore(), nil)
	snippets, _ := store.ListSnippets(1, 1)
	if len(snippets) == 0 {
		t.Fatal("no snippets found")
	}

	sn, ok := store.GetSnippet(snippets[0].ID)
	if !ok {
		t.Fatal("expected snippet found")
	}
	if sn.ID != snippets[0].ID {
		t.Fatal("snippet ID mismatch")
	}
}

// TestCachedStoreCreateAndDelete verifies write operations pass through.
func TestCachedStoreCreateAndDelete(t *testing.T) {
	store := NewCachedStore(NewStore(), nil)

	sn := store.CreateSnippet(model.CreateSnippetRequest{
		Title: "테스트",
		Code:  "출력(1)",
	}, "user1")

	if sn.Title != "테스트" {
		t.Fatalf("expected title '테스트', got %s", sn.Title)
	}

	found, owned := store.DeleteSnippet(sn.ID, "user1")
	if !found || !owned {
		t.Fatal("expected successful delete")
	}

	_, ok := store.GetSnippet(sn.ID)
	if ok {
		t.Fatal("expected snippet not found after delete")
	}
}

// TestCachedStoreSearchSnippets verifies search passes through.
func TestCachedStoreSearchSnippets(t *testing.T) {
	store := NewCachedStore(NewStore(), nil)

	results, total := store.SearchSnippets("안녕", 1, 10)
	if total < 1 {
		t.Fatal("expected at least one result for '안녕' search")
	}
	if len(results) < 1 {
		t.Fatal("expected results returned")
	}
}

// TestCachedStoreUpdateSnippet verifies update pass through.
func TestCachedStoreUpdateSnippet(t *testing.T) {
	store := NewCachedStore(NewStore(), nil)

	sn := store.CreateSnippet(model.CreateSnippetRequest{
		Title: "원본",
		Code:  "출력(1)",
	}, "user1")

	updated, found, owned := store.UpdateSnippet(sn.ID, model.UpdateSnippetRequest{
		Title: "수정됨",
		Code:  "출력(2)",
	}, "user1")

	if !found || !owned {
		t.Fatal("expected successful update")
	}
	if updated.Title != "수정됨" {
		t.Fatalf("expected title '수정됨', got %s", updated.Title)
	}
}

// TestCachedStoreForkSnippet verifies fork passes through.
func TestCachedStoreForkSnippet(t *testing.T) {
	store := NewCachedStore(NewStore(), nil)
	snippets, _ := store.ListSnippets(1, 1)

	forked, ok := store.ForkSnippet(snippets[0].ID, "user1")
	if !ok {
		t.Fatal("expected successful fork")
	}
	if forked.ID == snippets[0].ID {
		t.Fatal("forked snippet should have different ID")
	}
}

// TestCachedStoreShare verifies share operations pass through.
func TestCachedStoreShare(t *testing.T) {
	store := NewCachedStore(NewStore(), nil)

	shared := store.CreateShare(model.ShareRequest{
		Code:  "출력(1)",
		Title: "공유 테스트",
	})
	if shared.Token == "" {
		t.Fatal("expected non-empty token")
	}

	got, ok := store.GetShare(shared.Token)
	if !ok {
		t.Fatal("expected share found")
	}
	if got.Code != "출력(1)" {
		t.Fatal("share code mismatch")
	}
}

// TestCachedStoreAuth verifies auth operations pass through.
func TestCachedStoreAuth(t *testing.T) {
	store := NewCachedStore(NewStore(), nil)

	user, err := store.CreateUser("testuser", "password123")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	if user.Username != "testuser" {
		t.Fatal("username mismatch")
	}

	authed, err := store.AuthenticateUser("testuser", "password123")
	if err != nil {
		t.Fatalf("AuthenticateUser failed: %v", err)
	}
	if authed.ID != user.ID {
		t.Fatal("authenticated user ID mismatch")
	}
}

// TestCachedStoreDeleteNotOwned verifies ownership check.
func TestCachedStoreDeleteNotOwned(t *testing.T) {
	store := NewCachedStore(NewStore(), nil)

	sn := store.CreateSnippet(model.CreateSnippetRequest{
		Title: "소유자 테스트",
		Code:  "출력(1)",
	}, "owner")

	found, owned := store.DeleteSnippet(sn.ID, "other-user")
	if !found {
		t.Fatal("expected snippet found")
	}
	if owned {
		t.Fatal("expected not owned by other user")
	}
}
