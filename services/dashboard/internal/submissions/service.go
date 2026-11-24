package submissions

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	subRepo "code-runner/dashboard/internal/repository/submission"
	"code-runner/shared/cache"
	"code-runner/shared/dto"
)

var (
	ErrSubmissionNotFound = errors.New("submission not found")
)

type Service struct {
	repo  subRepo.Repository
	cache *cache.Cache
}

func NewService(repo subRepo.Repository, cache *cache.Cache) *Service {
	return &Service{
		repo:  repo,
		cache: cache,
	}
}

// List retrieves paginated submissions, hitting cache first then querying DB on miss.
func (s *Service) List(ctx context.Context, userID uuid.UUID, query dto.SubmissionListQuery) ([]dto.SubmissionSummaryResponse, int64, error) {
	page := query.Page
	if page < 1 {
		page = 1
	}
	limit := query.Limit
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	cacheKey := cache.Key("dashboard", "submissions", userID.String(), "list", page, limit, query.Status, query.Language)

	return cache.FetchPaginated(ctx, s.cache, cacheKey, 30*time.Second, func() ([]dto.SubmissionSummaryResponse, int64, error) {
		subs, total, err := s.repo.List(ctx, userID, query.Status, query.Language, limit, offset)
		if err != nil {
			return nil, 0, err
		}

		results := make([]dto.SubmissionSummaryResponse, len(subs))
		for i, sub := range subs {
			passed := 0
			for _, tc := range sub.TestResults {
				if tc.Passed {
					passed++
				}
			}

			results[i] = dto.SubmissionSummaryResponse{
				ID:              sub.ID,
				Language:        sub.Language,
				Status:          sub.Status,
				ExitCode:        sub.ExitCode,
				TimeTakenMs:     sub.TimeTakenMs,
				MemoryUsedKB:    sub.MemoryUsedKB,
				TotalTestCases:  len(sub.TestResults),
				PassedTestCases: passed,
				CreatedAt:       sub.CreatedAt,
				CompletedAt:     sub.CompletedAt,
			}
		}

		return results, total, nil
	})
}

// Get retrieves a single submission by ID, hitting cache first then querying DB on miss.
func (s *Service) Get(ctx context.Context, userID, submissionID uuid.UUID) (*dto.SubmissionDetailResponse, error) {
	cacheKey := cache.Key("dashboard", "submissions", userID.String(), "detail", submissionID.String())

	return cache.Fetch(ctx, s.cache, cacheKey, 1*time.Hour, func() (*dto.SubmissionDetailResponse, error) {
		sub, err := s.repo.FindByID(ctx, userID, submissionID)
		if err != nil {
			if errors.Is(err, subRepo.ErrSubmissionNotFound) {
				return nil, ErrSubmissionNotFound
			}
			return nil, err
		}

		return &dto.SubmissionDetailResponse{
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
			TimeTakenMs:   sub.TimeTakenMs,
			MemoryUsedKB:  sub.MemoryUsedKB,
			TestResults:   sub.TestResults,
			Error:         sub.Error,
			CreatedAt:     sub.CreatedAt,
			CompletedAt:   sub.CompletedAt,
		}, nil
	})
}
