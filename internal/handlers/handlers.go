package handlers

import (
	"encoding/json"
	"net/http"
	"pr-reviewer-service/internal/models"
	"pr-reviewer-service/internal/service"
)

type Handler struct {
	service *service.Service
}

func NewHandler(svc *service.Service) *Handler {
	return &Handler{service: svc}
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, code, message string) {
	respondJSON(w, status, models.ErrorResponse{
		Error: models.ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}

func handleServiceError(w http.ResponseWriter, err error) {
	if svcErr, ok := err.(*service.ServiceError); ok {
		status := http.StatusBadRequest
		switch svcErr.Code {
		case models.ErrCodeNotFound:
			status = http.StatusNotFound
		case models.ErrCodeTeamExists, models.ErrCodePRExists:
			status = http.StatusBadRequest
		case models.ErrCodePRMerged, models.ErrCodeNotAssigned, models.ErrCodeNoCandidate:
			status = http.StatusConflict
		}
		respondError(w, status, svcErr.Code, svcErr.Message)
		return
	}
	respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
}

func (h *Handler) CreateTeam(w http.ResponseWriter, r *http.Request) {
	var team models.Team
	if err := json.NewDecoder(r.Body).Decode(&team); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON")
		return
	}

	createdTeam, err := h.service.CreateTeam(&team)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"team": createdTeam,
	})
}

func (h *Handler) GetTeam(w http.ResponseWriter, r *http.Request) {
	teamName := r.URL.Query().Get("team_name")
	if teamName == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "team_name is required")
		return
	}

	team, err := h.service.GetTeam(teamName)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, team)
}

func (h *Handler) SetUserIsActive(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID   string `json:"user_id"`
		IsActive bool   `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON")
		return
	}

	user, err := h.service.SetUserIsActive(req.UserID, req.IsActive)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"user": user,
	})
}

func (h *Handler) CreatePullRequest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PullRequestID   string `json:"pull_request_id"`
		PullRequestName string `json:"pull_request_name"`
		AuthorID        string `json:"author_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON")
		return
	}

	pr, err := h.service.CreatePullRequest(req.PullRequestID, req.PullRequestName, req.AuthorID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"pr": pr,
	})
}

func (h *Handler) MergePullRequest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PullRequestID string `json:"pull_request_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON")
		return
	}

	pr, err := h.service.MergePullRequest(req.PullRequestID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"pr": pr,
	})
}

func (h *Handler) ReassignReviewer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PullRequestID string `json:"pull_request_id"`
		OldUserID     string `json:"old_user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON")
		return
	}

	pr, replacedBy, err := h.service.ReassignReviewer(req.PullRequestID, req.OldUserID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"pr":          pr,
		"replaced_by": replacedBy,
	})
}

func (h *Handler) GetUserReviews(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "user_id is required")
		return
	}

	prs, err := h.service.GetUserReviews(userID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"user_id":       userID,
		"pull_requests": prs,
	})
}

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func (h *Handler) GetStatistics(w http.ResponseWriter, r *http.Request) {
	stats, err := h.service.GetStatistics()
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, stats)
}

func (h *Handler) DeactivateTeamUsers(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TeamName string `json:"team_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON")
		return
	}

	count, err := h.service.DeactivateTeamUsers(req.TeamName)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"team_name":        req.TeamName,
		"deactivated_count": count,
	})
}
