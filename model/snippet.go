package model

import "time"

type Snippet struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Code        string    `json:"code"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateSnippetRequest struct {
	Title       string `json:"title" binding:"required"`
	Code        string `json:"code" binding:"required"`
	Description string `json:"description"`
}
