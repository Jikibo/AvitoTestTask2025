package e2e

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"pr-reviewer-service/internal/handlers"
	"pr-reviewer-service/internal/models"
	"pr-reviewer-service/internal/repository"
	"pr-reviewer-service/internal/service"
	"testing"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *sql.DB {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping e2e tests")
	}

	db, err := sql.Open("postgres", dbURL)
	require.NoError(t, err, "Failed to connect to test database")

	_, err = db.Exec("DROP SCHEMA public CASCADE; CREATE SCHEMA public;")
	require.NoError(t, err, "Failed to clean database")

	migrationSQL, err := os.ReadFile("../../migrations/001_init_schema.sql")
	require.NoError(t, err, "Failed to read migration file")

	_, err = db.Exec(string(migrationSQL))
	require.NoError(t, err, "Failed to execute migration")

	return db
}

func setupTestServer(db *sql.DB) *httptest.Server {
	repo := repository.NewRepository(db)
	svc := service.NewService(repo)
	handler := handlers.NewHandler(svc)

	router := mux.NewRouter()
	router.HandleFunc("/team/add", handler.CreateTeam).Methods("POST")
	router.HandleFunc("/team/get", handler.GetTeam).Methods("GET")
	router.HandleFunc("/users/setIsActive", handler.SetUserIsActive).Methods("POST")
	router.HandleFunc("/users/getReview", handler.GetUserReviews).Methods("GET")
	router.HandleFunc("/pullRequest/create", handler.CreatePullRequest).Methods("POST")
	router.HandleFunc("/pullRequest/merge", handler.MergePullRequest).Methods("POST")
	router.HandleFunc("/pullRequest/reassign", handler.ReassignReviewer).Methods("POST")
	router.HandleFunc("/health", handler.HealthCheck).Methods("GET")

	return httptest.NewServer(router)
}

func TestE2E_FullWorkflow(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	server := setupTestServer(db)
	defer server.Close()

	t.Run("Create Team", func(t *testing.T) {
		team := models.Team{
			TeamName: "backend",
			Members: []models.TeamMember{
				{UserID: "u1", Username: "Alice", IsActive: true},
				{UserID: "u2", Username: "Bob", IsActive: true},
				{UserID: "u3", Username: "Charlie", IsActive: true},
			},
		}

		body, _ := json.Marshal(team)
		resp, err := http.Post(server.URL+"/team/add", "application/json", bytes.NewBuffer(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result map[string]models.Team
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Equal(t, "backend", result["team"].TeamName)
		assert.Len(t, result["team"].Members, 3)
	})

	t.Run("Get Team", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/team/get?team_name=backend")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var team models.Team
		json.NewDecoder(resp.Body).Decode(&team)
		assert.Equal(t, "backend", team.TeamName)
		assert.Len(t, team.Members, 3)
	})

	var prID string
	t.Run("Create Pull Request", func(t *testing.T) {
		prReq := map[string]string{
			"pull_request_id":   "pr-1001",
			"pull_request_name": "Add search feature",
			"author_id":         "u1",
		}

		body, _ := json.Marshal(prReq)
		resp, err := http.Post(server.URL+"/pullRequest/create", "application/json", bytes.NewBuffer(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result map[string]models.PullRequest
		json.NewDecoder(resp.Body).Decode(&result)
		pr := result["pr"]
		prID = pr.PullRequestID

		assert.Equal(t, "pr-1001", pr.PullRequestID)
		assert.Equal(t, "u1", pr.AuthorID)
		assert.Equal(t, models.StatusOpen, pr.Status)

		assert.Len(t, pr.AssignedReviewers, 2)
		assert.NotContains(t, pr.AssignedReviewers, "u1", "Автор не должен быть ревьювером")
	})

	t.Run("Get User Reviews", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/users/getReview?user_id=u2")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Equal(t, "u2", result["user_id"])

		prs := result["pull_requests"].([]interface{})
		if len(prs) > 0 {
			assert.Greater(t, len(prs), 0)
		}
	})

	t.Run("Deactivate User", func(t *testing.T) {
		req := map[string]interface{}{
			"user_id":   "u3",
			"is_active": false,
		}

		body, _ := json.Marshal(req)
		resp, err := http.Post(server.URL+"/users/setIsActive", "application/json", bytes.NewBuffer(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]models.User
		json.NewDecoder(resp.Body).Decode(&result)
		assert.False(t, result["user"].IsActive)
	})

	t.Run("Merge Pull Request", func(t *testing.T) {
		req := map[string]string{
			"pull_request_id": prID,
		}

		body, _ := json.Marshal(req)
		resp, err := http.Post(server.URL+"/pullRequest/merge", "application/json", bytes.NewBuffer(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]models.PullRequest
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Equal(t, models.StatusMerged, result["pr"].Status)
		assert.NotNil(t, result["pr"].MergedAt)

		resp2, err := http.Post(server.URL+"/pullRequest/merge", "application/json", bytes.NewBuffer(body))
		require.NoError(t, err)
		defer resp2.Body.Close()

		assert.Equal(t, http.StatusOK, resp2.StatusCode)
	})

	t.Run("Reassign After Merge Should Fail", func(t *testing.T) {
		req := map[string]string{
			"pull_request_id": prID,
			"old_user_id":     "u2",
		}

		body, _ := json.Marshal(req)
		resp, err := http.Post(server.URL+"/pullRequest/reassign", "application/json", bytes.NewBuffer(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusConflict, resp.StatusCode)

		var errResp models.ErrorResponse
		json.NewDecoder(resp.Body).Decode(&errResp)
		assert.Equal(t, models.ErrCodePRMerged, errResp.Error.Code)
	})
}

func TestE2E_ReassignReviewer(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	server := setupTestServer(db)
	defer server.Close()

	team := models.Team{
		TeamName: "frontend",
		Members: []models.TeamMember{
			{UserID: "f1", Username: "Frank", IsActive: true},
			{UserID: "f2", Username: "Grace", IsActive: true},
			{UserID: "f3", Username: "Henry", IsActive: true},
			{UserID: "f4", Username: "Ivy", IsActive: true},
		},
	}

	body, _ := json.Marshal(team)
	resp, _ := http.Post(server.URL+"/team/add", "application/json", bytes.NewBuffer(body))
	resp.Body.Close()

	prReq := map[string]string{
		"pull_request_id":   "pr-2001",
		"pull_request_name": "Fix bug",
		"author_id":         "f1",
	}

	body, _ = json.Marshal(prReq)
	resp, _ = http.Post(server.URL+"/pullRequest/create", "application/json", bytes.NewBuffer(body))
	
	var prResult map[string]models.PullRequest
	json.NewDecoder(resp.Body).Decode(&prResult)
	resp.Body.Close()

	originalReviewers := prResult["pr"].AssignedReviewers
	require.Len(t, originalReviewers, 2, "Должно быть назначено 2 ревьювера")

	t.Run("Reassign Reviewer", func(t *testing.T) {
		reassignReq := map[string]string{
			"pull_request_id": "pr-2001",
			"old_user_id":     originalReviewers[0],
		}

		body, _ := json.Marshal(reassignReq)
		resp, err := http.Post(server.URL+"/pullRequest/reassign", "application/json", bytes.NewBuffer(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		pr := result["pr"].(map[string]interface{})
		replacedBy := result["replaced_by"].(string)

		reviewers := pr["assigned_reviewers"].([]interface{})
		assert.Len(t, reviewers, 2, "Должно остаться 2 ревьювера")
		assert.NotContains(t, reviewers, originalReviewers[0], "Старый ревьювер должен быть заменен")
		assert.NotEmpty(t, replacedBy, "Должен быть указан новый ревьювер")
	})
}

func TestE2E_ErrorCases(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	server := setupTestServer(db)
	defer server.Close()

	t.Run("Create Duplicate Team", func(t *testing.T) {
		team := models.Team{
			TeamName: "duplicate",
			Members:  []models.TeamMember{{UserID: "d1", Username: "Dan", IsActive: true}},
		}

		body, _ := json.Marshal(team)
		
		resp, _ := http.Post(server.URL+"/team/add", "application/json", bytes.NewBuffer(body))
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
		resp.Body.Close()

		resp, _ = http.Post(server.URL+"/team/add", "application/json", bytes.NewBuffer(body))
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var errResp models.ErrorResponse
		json.NewDecoder(resp.Body).Decode(&errResp)
		assert.Equal(t, models.ErrCodeTeamExists, errResp.Error.Code)
		resp.Body.Close()
	})

	t.Run("Get Non-Existent Team", func(t *testing.T) {
		resp, _ := http.Get(server.URL + "/team/get?team_name=nonexistent")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		var errResp models.ErrorResponse
		json.NewDecoder(resp.Body).Decode(&errResp)
		assert.Equal(t, models.ErrCodeNotFound, errResp.Error.Code)
	})

	t.Run("Create PR with Non-Existent Author", func(t *testing.T) {
		prReq := map[string]string{
			"pull_request_id":   "pr-9999",
			"pull_request_name": "Test",
			"author_id":         "nonexistent",
		}

		body, _ := json.Marshal(prReq)
		resp, _ := http.Post(server.URL+"/pullRequest/create", "application/json", bytes.NewBuffer(body))
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

func TestHealthCheck(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	server := setupTestServer(db)
	defer server.Close()

	resp, err := http.Get(server.URL + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "ok", result["status"])
}
