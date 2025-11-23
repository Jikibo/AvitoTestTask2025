package repository

import (
	"database/sql"
	"fmt"
	"pr-reviewer-service/internal/models"
	"time"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateTeam(teamName string) error {
	query := `INSERT INTO teams (team_name) VALUES ($1)`
	_, err := r.db.Exec(query, teamName)
	return err
}

func (r *Repository) TeamExists(teamName string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM teams WHERE team_name = $1)`
	err := r.db.QueryRow(query, teamName).Scan(&exists)
	return exists, err
}

func (r *Repository) UpsertUser(user *models.User) error {
	query := `
		INSERT INTO users (user_id, username, team_name, is_active, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) 
		DO UPDATE SET 
			username = EXCLUDED.username,
			team_name = EXCLUDED.team_name,
			is_active = EXCLUDED.is_active,
			updated_at = EXCLUDED.updated_at
	`
	_, err := r.db.Exec(query, user.UserID, user.Username, user.TeamName, user.IsActive, time.Now())
	return err
}

func (r *Repository) GetTeam(teamName string) (*models.Team, error) {
	exists, err := r.TeamExists(teamName)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, sql.ErrNoRows
	}

	query := `SELECT user_id, username, is_active FROM users WHERE team_name = $1 ORDER BY user_id`
	rows, err := r.db.Query(query, teamName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	members := []models.TeamMember{}
	for rows.Next() {
		var member models.TeamMember
		if err := rows.Scan(&member.UserID, &member.Username, &member.IsActive); err != nil {
			return nil, err
		}
		members = append(members, member)
	}

	return &models.Team{
		TeamName: teamName,
		Members:  members,
	}, nil
}

func (r *Repository) GetUser(userID string) (*models.User, error) {
	query := `SELECT user_id, username, team_name, is_active, created_at, updated_at 
	          FROM users WHERE user_id = $1`
	
	var user models.User
	err := r.db.QueryRow(query, userID).Scan(
		&user.UserID, &user.Username, &user.TeamName, 
		&user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *Repository) UpdateUserIsActive(userID string, isActive bool) error {
	query := `UPDATE users SET is_active = $1, updated_at = $2 WHERE user_id = $3`
	result, err := r.db.Exec(query, isActive, time.Now(), userID)
	if err != nil {
		return err
	}
	
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repository) GetActiveTeamMembers(teamName string, excludeUserID string) ([]models.User, error) {
	query := `
		SELECT user_id, username, team_name, is_active, created_at, updated_at
		FROM users 
		WHERE team_name = $1 AND is_active = true AND user_id != $2
		ORDER BY user_id
	`
	
	rows, err := r.db.Query(query, teamName, excludeUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		if err := rows.Scan(&user.UserID, &user.Username, &user.TeamName, 
			&user.IsActive, &user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

func (r *Repository) CreatePullRequest(pr *models.PullRequest) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err = tx.Exec(query, pr.PullRequestID, pr.PullRequestName, pr.AuthorID, pr.Status, pr.CreatedAt)
	if err != nil {
		return err
	}

	if len(pr.AssignedReviewers) > 0 {
		reviewerQuery := `INSERT INTO pr_reviewers (pull_request_id, user_id) VALUES ($1, $2)`
		for _, reviewerID := range pr.AssignedReviewers {
			_, err = tx.Exec(reviewerQuery, pr.PullRequestID, reviewerID)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (r *Repository) PRExists(prID string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM pull_requests WHERE pull_request_id = $1)`
	err := r.db.QueryRow(query, prID).Scan(&exists)
	return exists, err
}

func (r *Repository) GetPullRequest(prID string) (*models.PullRequest, error) {
	query := `
		SELECT pull_request_id, pull_request_name, author_id, status, created_at, merged_at
		FROM pull_requests 
		WHERE pull_request_id = $1
	`
	
	var pr models.PullRequest
	err := r.db.QueryRow(query, prID).Scan(
		&pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID, 
		&pr.Status, &pr.CreatedAt, &pr.MergedAt,
	)
	if err != nil {
		return nil, err
	}

	reviewersQuery := `SELECT user_id FROM pr_reviewers WHERE pull_request_id = $1 ORDER BY user_id`
	rows, err := r.db.Query(reviewersQuery, prID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pr.AssignedReviewers = []string{}
	for rows.Next() {
		var reviewerID string
		if err := rows.Scan(&reviewerID); err != nil {
			return nil, err
		}
		pr.AssignedReviewers = append(pr.AssignedReviewers, reviewerID)
	}

	return &pr, nil
}

func (r *Repository) MergePullRequest(prID string) error {
	query := `
		UPDATE pull_requests 
		SET status = $1, merged_at = $2 
		WHERE pull_request_id = $3 AND status != $1
	`
	_, err := r.db.Exec(query, models.StatusMerged, time.Now(), prID)
	return err
}

func (r *Repository) ReassignReviewer(prID, oldReviewerID, newReviewerID string) error {
	query := `
		UPDATE pr_reviewers 
		SET user_id = $1, assigned_at = $2 
		WHERE pull_request_id = $3 AND user_id = $4
	`
	result, err := r.db.Exec(query, newReviewerID, time.Now(), prID, oldReviewerID)
	if err != nil {
		return err
	}
	
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("reviewer not found in PR")
	}
	return nil
}

func (r *Repository) IsReviewerAssigned(prID, userID string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM pr_reviewers WHERE pull_request_id = $1 AND user_id = $2)`
	err := r.db.QueryRow(query, prID, userID).Scan(&exists)
	return exists, err
}

func (r *Repository) GetPullRequestsByReviewer(userID string) ([]models.PullRequestShort, error) {
	query := `
		SELECT pr.pull_request_id, pr.pull_request_name, pr.author_id, pr.status
		FROM pull_requests pr
		INNER JOIN pr_reviewers prr ON pr.pull_request_id = prr.pull_request_id
		WHERE prr.user_id = $1
		ORDER BY pr.created_at DESC
	`
	
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prs []models.PullRequestShort
	for rows.Next() {
		var pr models.PullRequestShort
		if err := rows.Scan(&pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID, &pr.Status); err != nil {
			return nil, err
		}
		prs = append(prs, pr)
	}

	return prs, nil
}

func (r *Repository) GetStatistics() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	var totalPRs int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM pull_requests`).Scan(&totalPRs)
	if err != nil {
		return nil, err
	}
	stats["total_prs"] = totalPRs

	var openPRs int
	err = r.db.QueryRow(`SELECT COUNT(*) FROM pull_requests WHERE status = 'OPEN'`).Scan(&openPRs)
	if err != nil {
		return nil, err
	}
	stats["open_prs"] = openPRs

	var mergedPRs int
	err = r.db.QueryRow(`SELECT COUNT(*) FROM pull_requests WHERE status = 'MERGED'`).Scan(&mergedPRs)
	if err != nil {
		return nil, err
	}
	stats["merged_prs"] = mergedPRs

	query := `
		SELECT u.user_id, u.username, COUNT(prr.pull_request_id) as review_count
		FROM users u
		LEFT JOIN pr_reviewers prr ON u.user_id = prr.user_id
		GROUP BY u.user_id, u.username
		ORDER BY review_count DESC
		LIMIT 10
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type ReviewerStat struct {
		UserID      string `json:"user_id"`
		Username    string `json:"username"`
		ReviewCount int    `json:"review_count"`
	}

	var topReviewers []ReviewerStat
	for rows.Next() {
		var stat ReviewerStat
		if err := rows.Scan(&stat.UserID, &stat.Username, &stat.ReviewCount); err != nil {
			return nil, err
		}
		topReviewers = append(topReviewers, stat)
	}
	stats["top_reviewers"] = topReviewers

	var totalTeams int
	err = r.db.QueryRow(`SELECT COUNT(*) FROM teams`).Scan(&totalTeams)
	if err != nil {
		return nil, err
	}
	stats["total_teams"] = totalTeams

	var totalUsers int
	err = r.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&totalUsers)
	if err != nil {
		return nil, err
	}
	stats["total_users"] = totalUsers

	var activeUsers int
	err = r.db.QueryRow(`SELECT COUNT(*) FROM users WHERE is_active = true`).Scan(&activeUsers)
	if err != nil {
		return nil, err
	}
	stats["active_users"] = activeUsers

	return stats, nil
}

func (r *Repository) DeactivateTeamUsers(teamName string) (int, error) {
	query := `UPDATE users SET is_active = false, updated_at = $1 WHERE team_name = $2 AND is_active = true`
	result, err := r.db.Exec(query, time.Now(), teamName)
	if err != nil {
		return 0, err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return int(rows), nil
}
