// Package models содержит структуры данных для сервиса
package models

import "time"

// User представляет пользователя системы
type User struct {
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	TeamName  string `json:"team_name"`
	IsActive  bool   `json:"is_active"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// TeamMember представляет участника команды (упрощенная версия User для API)
type TeamMember struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	IsActive bool   `json:"is_active"`
}

// Team представляет команду с участниками
type Team struct {
	TeamName string       `json:"team_name"`
	Members  []TeamMember `json:"members"`
}

// PullRequest представляет Pull Request
type PullRequest struct {
	PullRequestID     string    `json:"pull_request_id"`
	PullRequestName   string    `json:"pull_request_name"`
	AuthorID          string    `json:"author_id"`
	Status            string    `json:"status"` // OPEN или MERGED
	AssignedReviewers []string  `json:"assigned_reviewers"`
	CreatedAt         time.Time `json:"createdAt,omitempty"`
	MergedAt          *time.Time `json:"mergedAt,omitempty"`
}

// PullRequestShort представляет краткую информацию о PR
type PullRequestShort struct {
	PullRequestID   string `json:"pull_request_id"`
	PullRequestName string `json:"pull_request_name"`
	AuthorID        string `json:"author_id"`
	Status          string `json:"status"`
}

// Константы для статусов PR
const (
	StatusOpen   = "OPEN"
	StatusMerged = "MERGED"
)

// Константы для кодов ошибок API
const (
	ErrCodeTeamExists   = "TEAM_EXISTS"
	ErrCodePRExists     = "PR_EXISTS"
	ErrCodePRMerged     = "PR_MERGED"
	ErrCodeNotAssigned  = "NOT_ASSIGNED"
	ErrCodeNoCandidate  = "NO_CANDIDATE"
	ErrCodeNotFound     = "NOT_FOUND"
)

// ErrorResponse представляет структуру ошибки API
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail содержит детали ошибки
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
