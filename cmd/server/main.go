package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"pr-reviewer-service/internal/handlers"
	"pr-reviewer-service/internal/repository"
	"pr-reviewer-service/internal/service"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

func main() {
	dbURL := getEnv("DATABASE_URL", "postgres://pruser:prpassword@localhost:5432/pr_reviewer_db?sslmode=disable")
	port := getEnv("PORT", "8080")

	db, err := connectWithRetry(dbURL, 10, 2*time.Second)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := runMigrations(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	repo := repository.NewRepository(db)
	svc := service.NewService(repo)
	handler := handlers.NewHandler(svc)

	router := setupRouter(handler)

	addr := fmt.Sprintf(":%s", port)
	log.Printf("Server starting on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func setupRouter(h *handlers.Handler) *mux.Router {
	r := mux.NewRouter()

	r.HandleFunc("/team/add", h.CreateTeam).Methods("POST")
	r.HandleFunc("/team/get", h.GetTeam).Methods("GET")

	r.HandleFunc("/users/setIsActive", h.SetUserIsActive).Methods("POST")
	r.HandleFunc("/users/getReview", h.GetUserReviews).Methods("GET")

	r.HandleFunc("/pullRequest/create", h.CreatePullRequest).Methods("POST")
	r.HandleFunc("/pullRequest/merge", h.MergePullRequest).Methods("POST")
	r.HandleFunc("/pullRequest/reassign", h.ReassignReviewer).Methods("POST")

	r.HandleFunc("/health", h.HealthCheck).Methods("GET")

	r.HandleFunc("/statistics", h.GetStatistics).Methods("GET")

	r.HandleFunc("/team/deactivateUsers", h.DeactivateTeamUsers).Methods("POST")

	return r
}

func connectWithRetry(dbURL string, maxRetries int, delay time.Duration) (*sql.DB, error) {
	var db *sql.DB
	var err error

	for i := 0; i < maxRetries; i++ {
		db, err = sql.Open("postgres", dbURL)
		if err != nil {
			log.Printf("Failed to open database (attempt %d/%d): %v", i+1, maxRetries, err)
			time.Sleep(delay)
			continue
		}

		err = db.Ping()
		if err == nil {
			log.Println("Successfully connected to database")
			return db, nil
		}

		log.Printf("Failed to ping database (attempt %d/%d): %v", i+1, maxRetries, err)
		db.Close()
		time.Sleep(delay)
	}

	return nil, fmt.Errorf("failed to connect after %d attempts: %w", maxRetries, err)
}

func runMigrations(db *sql.DB) error {
	log.Println("Running migrations...")

	migrationSQL, err := os.ReadFile("migrations/001_init_schema.sql")
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	if _, err := db.Exec(string(migrationSQL)); err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	log.Println("Migrations completed successfully")
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
