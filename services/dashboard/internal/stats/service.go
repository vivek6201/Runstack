package stats

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	subRepo "code-runner/dashboard/internal/repository/submission"
	"code-runner/shared/cache"
	"code-runner/shared/dto"
	"code-runner/shared/models"
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

// GetDashboardStats wraps the DB aggregation query with cache.Fetch:
// checks Redis first; on miss, aggregates from DB, stores in Redis, and returns.
func (s *Service) GetDashboardStats(ctx context.Context, userID uuid.UUID, plan models.PlanTier) (*dto.DashboardStatsResponse, error) {
	cacheKey := cache.Key("dashboard", "stats", userID.String())

	return cache.Fetch(ctx, s.cache, cacheKey, 30*time.Second, func() (*dto.DashboardStatsResponse, error) {
		aggStats, err := s.repo.GetStats(ctx, userID)
		if err != nil {
			return nil, err
		}

		successRate := 0.0
		if aggStats.TotalSubmissions > 0 {
			successRate = (float64(aggStats.SuccessfulRuns) / float64(aggStats.TotalSubmissions)) * 100.0
		}

		// Monthly quota calculations
		now := time.Now().UTC()
		startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		monthlyRunsUsed, _ := s.repo.GetMonthlyRuns(ctx, userID, startOfMonth)

		// Quota limits based on plan
		var monthlyLimit int64
		switch plan {
		case models.PlanPro:
			monthlyLimit = 50000
		case models.PlanEnterprise:
			monthlyLimit = 1000000
		default: // Free
			monthlyLimit = 1000
		}

		if s.cache != nil && s.cache.Client() != nil {
			ym := fmt.Sprintf("%04d-%02d", now.Year(), now.Month())
			quotaKey := cache.Key("user", "quota", userID.String(), ym, "runs")
			if count, err := s.cache.Client().Get(ctx, quotaKey).Int64(); err == nil && count > monthlyRunsUsed {
				monthlyRunsUsed = count
			}
		}

		return &dto.DashboardStatsResponse{
			TotalSubmissions: aggStats.TotalSubmissions,
			SuccessfulRuns:   aggStats.SuccessfulRuns,
			FailedRuns:       aggStats.FailedRuns,
			TimeoutRuns:      aggStats.TimeoutRuns,
			SuccessRate:      successRate,
			AvgTimeMs:        aggStats.AvgTimeMs,
			TotalComputeSec:  float64(aggStats.TotalTimeMs) / 1000.0,
			MonthlyRunsUsed:  monthlyRunsUsed,
			MonthlyRunsLimit: monthlyLimit,
		}, nil
	})
}

// GetDailyStats wraps the daily stats DB query with cache.Fetch.
func (s *Service) GetDailyStats(ctx context.Context, userID uuid.UUID, days int) ([]dto.DailyStatPoint, error) {
	cacheKey := cache.Key("dashboard", "stats", "daily", userID.String(), days)

	return cache.Fetch(ctx, s.cache, cacheKey, 2*time.Minute, func() ([]dto.DailyStatPoint, error) {
		return s.repo.GetDailyStats(ctx, userID, days)
	})
}
