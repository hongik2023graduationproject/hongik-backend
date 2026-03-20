package model

import "time"

type Snippet struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Code        string    `json:"code"`
	Description string    `json:"description"`
	UserID      string    `json:"user_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateSnippetRequest struct {
	Title       string `json:"title" binding:"required"`
	Code        string `json:"code" binding:"required"`
	Description string `json:"description"`
}

type UpdateSnippetRequest struct {
	Title       string `json:"title" binding:"required"`
	Code        string `json:"code" binding:"required"`
	Description string `json:"description"`
}

type SnippetListResponse struct {
	Snippets []Snippet `json:"snippets"`
	Total    int       `json:"total"`
	Page     int       `json:"page"`
	Limit    int       `json:"limit"`
}
