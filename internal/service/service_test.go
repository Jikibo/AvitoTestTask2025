package service

import (
	"pr-reviewer-service/internal/models"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSelectRandomReviewers(t *testing.T) {
	svc := NewService(nil)

	tests := []struct {
		name       string
		candidates []models.User
		maxCount   int
		wantCount  int
	}{
		{
			name:       "Нет кандидатов",
			candidates: []models.User{},
			maxCount:   2,
			wantCount:  0,
		},
		{
			name: "Один кандидат, запрашиваем 2",
			candidates: []models.User{
				{UserID: "u1", Username: "Alice", IsActive: true},
			},
			maxCount:  2,
			wantCount: 1,
		},
		{
			name: "Три кандидата, запрашиваем 2",
			candidates: []models.User{
				{UserID: "u1", Username: "Alice", IsActive: true},
				{UserID: "u2", Username: "Bob", IsActive: true},
				{UserID: "u3", Username: "Charlie", IsActive: true},
			},
			maxCount:  2,
			wantCount: 2,
		},
		{
			name: "Два кандидата, запрашиваем 2",
			candidates: []models.User{
				{UserID: "u1", Username: "Alice", IsActive: true},
				{UserID: "u2", Username: "Bob", IsActive: true},
			},
			maxCount:  2,
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reviewers := svc.selectRandomReviewers(tt.candidates, tt.maxCount)
			assert.Equal(t, tt.wantCount, len(reviewers), "Неверное количество выбранных ревьюверов")

			candidateIDs := make(map[string]bool)
			for _, c := range tt.candidates {
				candidateIDs[c.UserID] = true
			}
			for _, reviewerID := range reviewers {
				assert.True(t, candidateIDs[reviewerID], "Ревьювер %s не найден в списке кандидатов", reviewerID)
			}

			uniqueReviewers := make(map[string]bool)
			for _, reviewerID := range reviewers {
				assert.False(t, uniqueReviewers[reviewerID], "Ревьювер %s выбран дважды", reviewerID)
				uniqueReviewers[reviewerID] = true
			}
		})
	}
}

func TestServiceError(t *testing.T) {
	err := &ServiceError{
		Code:    models.ErrCodeNotFound,
		Message: "resource not found",
	}

	assert.Equal(t, "NOT_FOUND: resource not found", err.Error())
}

func TestSelectRandomReviewers_Randomness(t *testing.T) {
	svc := NewService(nil)

	candidates := []models.User{
		{UserID: "u1", Username: "Alice", IsActive: true},
		{UserID: "u2", Username: "Bob", IsActive: true},
		{UserID: "u3", Username: "Charlie", IsActive: true},
		{UserID: "u4", Username: "Dave", IsActive: true},
		{UserID: "u5", Username: "Eve", IsActive: true},
	}

	results := make(map[string]int)
	iterations := 100

	for i := 0; i < iterations; i++ {
		reviewers := svc.selectRandomReviewers(candidates, 2)
		assert.Equal(t, 2, len(reviewers), "Должно быть выбрано 2 ревьювера")

		key := reviewers[0] + "," + reviewers[1]
		if reviewers[0] > reviewers[1] {
			key = reviewers[1] + "," + reviewers[0]
		}
		results[key]++
	}

	assert.Greater(t, len(results), 1, "Выбор должен быть случайным, получена только одна комбинация")
}
