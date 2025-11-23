package service

import (
	"database/sql"
	"fmt"
	"math/rand"
	"pr-reviewer-service/internal/models"
	"pr-reviewer-service/internal/repository"
	"time"
)

type Service struct {
	repo *repository.Repository
	rand *rand.Rand
}

func NewService(repo *repository.Repository) *Service {
	source := rand.NewSource(time.Now().UnixNano())
	return &Service{
		repo: repo,
		rand: rand.New(source),
	}
}

func (s *Service) CreateTeam(team *models.Team) (*models.Team, error) {
	exists, err := s.repo.TeamExists(team.TeamName)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, &ServiceError{
			Code:    models.ErrCodeTeamExists,
			Message: "team_name already exists",
		}
	}

	if err := s.repo.CreateTeam(team.TeamName); err != nil {
		return nil, err
	}

	for _, member := range team.Members {
		user := &models.User{
			UserID:   member.UserID,
			Username: member.Username,
			TeamName: team.TeamName,
			IsActive: member.IsActive,
		}
		if err := s.repo.UpsertUser(user); err != nil {
			return nil, err
		}
	}

	return s.repo.GetTeam(team.TeamName)
}

func (s *Service) GetTeam(teamName string) (*models.Team, error) {
	team, err := s.repo.GetTeam(teamName)
	if err == sql.ErrNoRows {
		return nil, &ServiceError{
			Code:    models.ErrCodeNotFound,
			Message: "team not found",
		}
	}
	return team, err
}

func (s *Service) SetUserIsActive(userID string, isActive bool) (*models.User, error) {
	user, err := s.repo.GetUser(userID)
	if err == sql.ErrNoRows {
		return nil, &ServiceError{
			Code:    models.ErrCodeNotFound,
			Message: "user not found",
		}
	}
	if err != nil {
		return nil, err
	}

	if err := s.repo.UpdateUserIsActive(userID, isActive); err != nil {
		return nil, err
	}

	user.IsActive = isActive
	return user, nil
}

func (s *Service) CreatePullRequest(prID, prName, authorID string) (*models.PullRequest, error) {
	exists, err := s.repo.PRExists(prID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, &ServiceError{
			Code:    models.ErrCodePRExists,
			Message: "PR id already exists",
		}
	}

	author, err := s.repo.GetUser(authorID)
	if err == sql.ErrNoRows {
		return nil, &ServiceError{
			Code:    models.ErrCodeNotFound,
			Message: "author not found",
		}
	}
	if err != nil {
		return nil, err
	}

	candidates, err := s.repo.GetActiveTeamMembers(author.TeamName, authorID)
	if err != nil {
		return nil, err
	}

	reviewers := s.selectRandomReviewers(candidates, 2)

	pr := &models.PullRequest{
		PullRequestID:     prID,
		PullRequestName:   prName,
		AuthorID:          authorID,
		Status:            models.StatusOpen,
		AssignedReviewers: reviewers,
		CreatedAt:         time.Now(),
	}

	if err := s.repo.CreatePullRequest(pr); err != nil {
		return nil, err
	}

	return pr, nil
}

func (s *Service) MergePullRequest(prID string) (*models.PullRequest, error) {
	pr, err := s.repo.GetPullRequest(prID)
	if err == sql.ErrNoRows {
		return nil, &ServiceError{
			Code:    models.ErrCodeNotFound,
			Message: "PR not found",
		}
	}
	if err != nil {
		return nil, err
	}

	if pr.Status == models.StatusMerged {
		return pr, nil
	}

	if err := s.repo.MergePullRequest(prID); err != nil {
		return nil, err
	}

	return s.repo.GetPullRequest(prID)
}

func (s *Service) ReassignReviewer(prID, oldReviewerID string) (*models.PullRequest, string, error) {
	pr, err := s.repo.GetPullRequest(prID)
	if err == sql.ErrNoRows {
		return nil, "", &ServiceError{
			Code:    models.ErrCodeNotFound,
			Message: "PR not found",
		}
	}
	if err != nil {
		return nil, "", err
	}

	if pr.Status == models.StatusMerged {
		return nil, "", &ServiceError{
			Code:    models.ErrCodePRMerged,
			Message: "cannot reassign on merged PR",
		}
	}

	isAssigned, err := s.repo.IsReviewerAssigned(prID, oldReviewerID)
	if err != nil {
		return nil, "", err
	}
	if !isAssigned {
		return nil, "", &ServiceError{
			Code:    models.ErrCodeNotAssigned,
			Message: "reviewer is not assigned to this PR",
		}
	}

	oldReviewer, err := s.repo.GetUser(oldReviewerID)
	if err == sql.ErrNoRows {
		return nil, "", &ServiceError{
			Code:    models.ErrCodeNotFound,
			Message: "old reviewer not found",
		}
	}
	if err != nil {
		return nil, "", err
	}

	excludeIDs := map[string]bool{
		oldReviewerID: true,
		pr.AuthorID:   true,
	}
	for _, reviewerID := range pr.AssignedReviewers {
		excludeIDs[reviewerID] = true
	}

	candidates, err := s.repo.GetActiveTeamMembers(oldReviewer.TeamName, oldReviewerID)
	if err != nil {
		return nil, "", err
	}

	var availableCandidates []models.User
	for _, candidate := range candidates {
		if !excludeIDs[candidate.UserID] {
			availableCandidates = append(availableCandidates, candidate)
		}
	}

	if len(availableCandidates) == 0 {
		return nil, "", &ServiceError{
			Code:    models.ErrCodeNoCandidate,
			Message: "no active replacement candidate in team",
		}
	}

	newReviewer := availableCandidates[s.rand.Intn(len(availableCandidates))]

	if err := s.repo.ReassignReviewer(prID, oldReviewerID, newReviewer.UserID); err != nil {
		return nil, "", err
	}

	updatedPR, err := s.repo.GetPullRequest(prID)
	if err != nil {
		return nil, "", err
	}

	return updatedPR, newReviewer.UserID, nil
}

func (s *Service) GetUserReviews(userID string) ([]models.PullRequestShort, error) {
	_, err := s.repo.GetUser(userID)
	if err == sql.ErrNoRows {
		return []models.PullRequestShort{}, nil
	}
	if err != nil {
		return nil, err
	}

	return s.repo.GetPullRequestsByReviewer(userID)
}

func (s *Service) selectRandomReviewers(candidates []models.User, maxCount int) []string {
	if len(candidates) == 0 {
		return []string{}
	}

	count := maxCount
	if len(candidates) < count {
		count = len(candidates)
	}

	shuffled := make([]models.User, len(candidates))
	copy(shuffled, candidates)
	s.rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	reviewers := make([]string, count)
	for i := 0; i < count; i++ {
		reviewers[i] = shuffled[i].UserID
	}

	return reviewers
}

type ServiceError struct {
	Code    string
	Message string
}

func (e *ServiceError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (s *Service) GetStatistics() (map[string]interface{}, error) {
	return s.repo.GetStatistics()
}

func (s *Service) DeactivateTeamUsers(teamName string) (int, error) {
	exists, err := s.repo.TeamExists(teamName)
	if err != nil {
		return 0, err
	}
	if !exists {
		return 0, &ServiceError{
			Code:    models.ErrCodeNotFound,
			Message: "team not found",
		}
	}

	return s.repo.DeactivateTeamUsers(teamName)
}
