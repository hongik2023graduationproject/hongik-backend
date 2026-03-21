package service

import (
	"database/sql"
	"log/slog"
	"strings"
	"time"

	"hongik-backend/model"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

// PostgresStore implements Store using PostgreSQL.
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore opens a connection to PostgreSQL and returns a PostgresStore.
func NewPostgresStore(databaseURL string) (*PostgresStore, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	s := &PostgresStore{db: db}
	go s.cleanupExpiredShares()
	return s, nil
}

// Close closes the underlying database connection.
func (s *PostgresStore) Close() error {
	return s.db.Close()
}

func (s *PostgresStore) cleanupExpiredShares() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		result, err := s.db.Exec("DELETE FROM shared_codes WHERE expires_at > 0 AND expires_at < $1", time.Now().Unix())
		if err != nil {
			slog.Error("failed to cleanup expired shares", slog.String("error", err.Error()))
			continue
		}
		if n, _ := result.RowsAffected(); n > 0 {
			slog.Info("cleaned up expired shares", slog.Int64("count", n))
		}
	}
}

// --- Snippet operations ---

func (s *PostgresStore) ListSnippets(page, limit int) ([]model.Snippet, int) {
	var total int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM snippets").Scan(&total); err != nil {
		slog.Error("ListSnippets count failed", slog.String("error", err.Error()))
		return []model.Snippet{}, 0
	}

	offset := (page - 1) * limit
	if offset >= total {
		return []model.Snippet{}, total
	}

	rows, err := s.db.Query(
		"SELECT id, title, code, description, user_id, created_at, updated_at FROM snippets ORDER BY created_at DESC LIMIT $1 OFFSET $2",
		limit, offset,
	)
	if err != nil {
		slog.Error("ListSnippets query failed", slog.String("error", err.Error()))
		return []model.Snippet{}, total
	}
	defer rows.Close() //nolint:errcheck

	snippets := make([]model.Snippet, 0)
	for rows.Next() {
		var sn model.Snippet
		var userID sql.NullString
		if err := rows.Scan(&sn.ID, &sn.Title, &sn.Code, &sn.Description, &userID, &sn.CreatedAt, &sn.UpdatedAt); err != nil {
			slog.Error("ListSnippets scan failed", slog.String("error", err.Error()))
			continue
		}
		sn.UserID = userID.String
		snippets = append(snippets, sn)
	}
	return snippets, total
}

func (s *PostgresStore) SearchSnippets(query string, page, limit int) ([]model.Snippet, int) {
	q := "%" + strings.ToLower(query) + "%"

	var total int
	if err := s.db.QueryRow(
		"SELECT COUNT(*) FROM snippets WHERE LOWER(title) LIKE $1 OR LOWER(description) LIKE $1", q,
	).Scan(&total); err != nil {
		slog.Error("SearchSnippets count failed", slog.String("error", err.Error()))
		return []model.Snippet{}, 0
	}

	offset := (page - 1) * limit
	if offset >= total {
		return []model.Snippet{}, total
	}

	rows, err := s.db.Query(
		"SELECT id, title, code, description, user_id, created_at, updated_at FROM snippets WHERE LOWER(title) LIKE $1 OR LOWER(description) LIKE $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3",
		q, limit, offset,
	)
	if err != nil {
		slog.Error("SearchSnippets query failed", slog.String("error", err.Error()))
		return []model.Snippet{}, total
	}
	defer rows.Close() //nolint:errcheck

	snippets := make([]model.Snippet, 0)
	for rows.Next() {
		var sn model.Snippet
		var userID sql.NullString
		if err := rows.Scan(&sn.ID, &sn.Title, &sn.Code, &sn.Description, &userID, &sn.CreatedAt, &sn.UpdatedAt); err != nil {
			slog.Error("SearchSnippets scan failed", slog.String("error", err.Error()))
			continue
		}
		sn.UserID = userID.String
		snippets = append(snippets, sn)
	}
	return snippets, total
}

func (s *PostgresStore) GetSnippet(id string) (model.Snippet, bool) {
	var sn model.Snippet
	var userID sql.NullString
	err := s.db.QueryRow(
		"SELECT id, title, code, description, user_id, created_at, updated_at FROM snippets WHERE id = $1", id,
	).Scan(&sn.ID, &sn.Title, &sn.Code, &sn.Description, &userID, &sn.CreatedAt, &sn.UpdatedAt)
	if err == sql.ErrNoRows {
		return model.Snippet{}, false
	}
	if err != nil {
		slog.Error("GetSnippet failed", slog.String("error", err.Error()))
		return model.Snippet{}, false
	}
	sn.UserID = userID.String
	return sn, true
}

func (s *PostgresStore) CreateSnippet(req model.CreateSnippetRequest, userID string) model.Snippet {
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

	var uid *string
	if userID != "" {
		uid = &userID
	}

	_, err := s.db.Exec(
		"INSERT INTO snippets (id, title, code, description, user_id, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7)",
		sn.ID, sn.Title, sn.Code, sn.Description, uid, sn.CreatedAt, sn.UpdatedAt,
	)
	if err != nil {
		slog.Error("CreateSnippet failed", slog.String("error", err.Error()))
	}
	return sn
}

func (s *PostgresStore) UpdateSnippet(id string, req model.UpdateSnippetRequest, userID string) (model.Snippet, bool, bool) {
	sn, exists := s.GetSnippet(id)
	if !exists {
		return model.Snippet{}, false, false
	}

	if sn.UserID != "" && sn.UserID != userID {
		return model.Snippet{}, true, false
	}

	now := time.Now()
	_, err := s.db.Exec(
		"UPDATE snippets SET title = $1, code = $2, description = $3, updated_at = $4 WHERE id = $5",
		req.Title, req.Code, req.Description, now, id,
	)
	if err != nil {
		slog.Error("UpdateSnippet failed", slog.String("error", err.Error()))
		return model.Snippet{}, true, false
	}

	sn.Title = req.Title
	sn.Code = req.Code
	sn.Description = req.Description
	sn.UpdatedAt = now
	return sn, true, true
}

func (s *PostgresStore) DeleteSnippet(id string, userID string) (bool, bool) {
	sn, exists := s.GetSnippet(id)
	if !exists {
		return false, false
	}

	if sn.UserID != "" && sn.UserID != userID {
		return true, false
	}

	_, err := s.db.Exec("DELETE FROM snippets WHERE id = $1", id)
	if err != nil {
		slog.Error("DeleteSnippet failed", slog.String("error", err.Error()))
		return true, false
	}
	return true, true
}

func (s *PostgresStore) ForkSnippet(id string, userID string) (model.Snippet, bool) {
	original, exists := s.GetSnippet(id)
	if !exists {
		return model.Snippet{}, false
	}

	now := time.Now()
	forked := model.Snippet{
		ID:          uuid.New().String(),
		Title:       original.Title + " (복사본)",
		Code:        original.Code,
		Description: original.Description,
		UserID:      userID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	var uid *string
	if userID != "" {
		uid = &userID
	}

	_, err := s.db.Exec(
		"INSERT INTO snippets (id, title, code, description, user_id, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7)",
		forked.ID, forked.Title, forked.Code, forked.Description, uid, forked.CreatedAt, forked.UpdatedAt,
	)
	if err != nil {
		slog.Error("ForkSnippet failed", slog.String("error", err.Error()))
		return model.Snippet{}, false
	}
	return forked, true
}

// --- Share operations ---

func (s *PostgresStore) CreateShare(req model.ShareRequest) model.SharedCode {
	token := uuid.New().String()
	shared := model.SharedCode{
		Token:     token,
		Code:      req.Code,
		Title:     req.Title,
		CreatedAt: time.Now().Format(time.RFC3339),
		ExpiresAt: time.Now().Add(shareTTL).Unix(),
	}

	_, err := s.db.Exec(
		"INSERT INTO shared_codes (token, code, title, created_at, expires_at) VALUES ($1, $2, $3, $4, $5)",
		shared.Token, shared.Code, shared.Title, shared.CreatedAt, shared.ExpiresAt,
	)
	if err != nil {
		slog.Error("CreateShare failed", slog.String("error", err.Error()))
	}
	return shared
}

func (s *PostgresStore) GetShare(token string) (model.SharedCode, bool) {
	var shared model.SharedCode
	err := s.db.QueryRow(
		"SELECT token, code, title, created_at, expires_at FROM shared_codes WHERE token = $1", token,
	).Scan(&shared.Token, &shared.Code, &shared.Title, &shared.CreatedAt, &shared.ExpiresAt)
	if err == sql.ErrNoRows {
		return model.SharedCode{}, false
	}
	if err != nil {
		slog.Error("GetShare failed", slog.String("error", err.Error()))
		return model.SharedCode{}, false
	}
	if shared.ExpiresAt > 0 && time.Now().Unix() > shared.ExpiresAt {
		return model.SharedCode{}, false
	}
	return shared, true
}

// --- User operations ---

func (s *PostgresStore) CreateUser(username, password string) (model.User, error) {
	// Check if username already exists
	var count int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM users WHERE username = $1", username).Scan(&count); err != nil {
		return model.User{}, err
	}
	if count > 0 {
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

	_, err = s.db.Exec(
		"INSERT INTO users (id, username, password_hash, created_at) VALUES ($1, $2, $3, $4)",
		user.ID, user.Username, user.PasswordHash, user.CreatedAt,
	)
	if err != nil {
		return model.User{}, err
	}
	return user, nil
}

func (s *PostgresStore) AuthenticateUser(username, password string) (model.User, error) {
	var user model.User
	err := s.db.QueryRow(
		"SELECT id, username, password_hash, created_at FROM users WHERE username = $1", username,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt)
	if err == sql.ErrNoRows {
		return model.User{}, ErrUserNotFound
	}
	if err != nil {
		return model.User{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return model.User{}, ErrInvalidPassword
	}
	return user, nil
}
