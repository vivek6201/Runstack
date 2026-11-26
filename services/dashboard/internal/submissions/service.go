package submissions

import (
	"context"
	"time"

	"github.com/google/uuid"

	"code-runner/shared/cache"
	"code-runner/shared/models"
)

type ISubmissionsService interface {
	List(ctx context.Context, userID uuid.UUID, query SubmissionListQuery) ([]SubmissionSummaryResponse, int64, error)
	Get(ctx context.Context, userID, submissionID uuid.UUID) (*SubmissionDetailResponse, error)
}

type SubmissionsService struct {
	repo   ISubmissionsRepository
	caches *Caches
}

func NewSubmissionsService(repo ISubmissionsRepository, caches *Caches) ISubmissionsService {
	return &SubmissionsService{
		repo:   repo,
		caches: caches,
	}
}

func (s *SubmissionsService) List(ctx context.Context, userID uuid.UUID, query SubmissionListQuery) ([]SubmissionSummaryResponse, int64, error) {
	page, limit, offset := normalizePagination(query.Page, query.Limit)
	load := func() (*PaginatedSubmissions, error) {
		return s.loadPaginatedFromDB(ctx, userID, query, limit, offset)
	}

	if s.caches == nil || s.caches.List == nil {
		res, err := load()
		if err != nil {
			return nil, 0, err
		}
		return res.Items, res.Total, nil
	}

	cacheKey := cache.Key("dashboard", "submissions", userID.String(), "list", page, limit, query.Status, query.Language)
	res, err := s.caches.List.Fetch(ctx, cacheKey, 30*time.Second, load)
	if err != nil {
		return nil, 0, err
	}
	return res.Items, res.Total, nil
}

func (s *SubmissionsService) Get(ctx context.Context, userID, submissionID uuid.UUID) (*SubmissionDetailResponse, error) {
	load := func() (*SubmissionDetailResponse, error) {
		sub, err := s.repo.FindByID(ctx, userID, submissionID)
		if err != nil {
			return nil, err
		}
		return mapSubmissionDetail(sub), nil
	}

	if s.caches == nil || s.caches.Detail == nil {
		return load()
	}

	cacheKey := cache.Key("dashboard", "submissions", userID.String(), "detail", submissionID.String())
	return s.caches.Detail.Fetch(ctx, cacheKey, 1*time.Hour, load)
}

func (s *SubmissionsService) loadPaginatedFromDB(ctx context.Context, userID uuid.UUID, query SubmissionListQuery, limit, offset int) (*PaginatedSubmissions, error) {
	subs, total, err := s.repo.List(ctx, userID, query.Status, query.Language, limit, offset)
	if err != nil {
		return nil, err
	}
	return &PaginatedSubmissions{
		Items: mapSubmissionSummaries(subs),
		Total: total,
	}, nil
}

func normalizePagination(page, limit int) (int, int, int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	return page, limit, (page - 1) * limit
}

func mapSubmissionSummaries(subs []models.Submission) []SubmissionSummaryResponse {
	results := make([]SubmissionSummaryResponse, len(subs))
	for i, sub := range subs {
		results[i] = SubmissionSummaryResponse{
			ID:              sub.ID,
			Language:        sub.Language,
			Status:          sub.Status,
			ExitCode:        sub.ExitCode,
			TimeTakenMs:     sub.TimeTakenMs,
			MemoryUsedKB:    sub.MemoryUsedKB,
			TotalTestCases:  len(sub.TestResults),
			PassedTestCases: countPassedTestCases(sub.TestResults),
			CreatedAt:       sub.CreatedAt,
			CompletedAt:     sub.CompletedAt,
		}
	}
	return results
}

func countPassedTestCases(results models.JSONBTestResults) int {
	passed := 0
	for _, tc := range results {
		if tc.Passed {
			passed++
		}
	}
	return passed
}

func mapSubmissionDetail(sub *models.Submission) *SubmissionDetailResponse {
	return &SubmissionDetailResponse{
		ID:            sub.ID,
		UserID:        sub.UserID,
		APIKeyID:      sub.APIKeyID,
		Language:      sub.Language,
		Code:          sub.Code,
		Status:        sub.Status,
		TimeLimitMs:   sub.TimeLimitMs,
		MemoryLimitMB: sub.MemoryLimitMB,
		ExitCode:      sub.ExitCode,
		Stdout:        sub.Stdout,
		Stderr:        sub.Stderr,
		Error:         sub.Error,
		TimeTakenMs:   sub.TimeTakenMs,
		MemoryUsedKB:  sub.MemoryUsedKB,
		TestResults:   sub.TestResults,
		CreatedAt:     sub.CreatedAt,
		StartedAt:     sub.StartedAt,
		CompletedAt:   sub.CompletedAt,
	}
}
