package stats

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"code-runner/dashboard/internal/submissions"
	"code-runner/shared/cache"
	"code-runner/shared/models"
)

type IStatsService interface {
	GetDashboardStats(ctx context.Context, userID uuid.UUID, plan models.PlanTier) (*DashboardStatsResponse, error)
	GetDailyStats(ctx context.Context, userID uuid.UUID, days int) ([]DailyStatPoint, error)
}

type StatsService struct {
	repo   submissions.ISubmissionsRepository
	rdb    *redis.Client
	caches *Caches
}

func NewStatsService(repo submissions.ISubmissionsRepository, rdb *redis.Client, caches *Caches) IStatsService {
	return &StatsService{
		repo:   repo,
		rdb:    rdb,
		caches: caches,
	}
}

func (s *StatsService) GetDashboardStats(ctx context.Context, userID uuid.UUID, plan models.PlanTier) (*DashboardStatsResponse, error) {
	load := func() (*DashboardStatsResponse, error) {
		return s.computeDashboardStats(ctx, userID, plan)
	}

	if s.caches == nil || s.caches.Overview == nil {
		return load()
	}

	cacheKey := cache.Key("dashboard", "stats", userID.String())
	return s.caches.Overview.Fetch(ctx, cacheKey, 30*time.Second, load)
}

func (s *StatsService) GetDailyStats(ctx context.Context, userID uuid.UUID, days int) ([]DailyStatPoint, error) {
	load := func() (*[]DailyStatPoint, error) {
		rows, err := s.repo.GetDailyRows(ctx, userID, days)
		if err != nil {
			return nil, err
		}
		points := mapDailyPoints(rows)
		return &points, nil
	}

	if s.caches == nil || s.caches.Daily == nil {
		res, err := load()
		if err != nil {
			return nil, err
		}
		return *res, nil
	}

	cacheKey := cache.Key("dashboard", "stats", "daily", userID.String(), days)
	res, err := s.caches.Daily.Fetch(ctx, cacheKey, 2*time.Minute, load)
	if err != nil {
		return nil, err
	}
	return *res, nil
}

func (s *StatsService) computeDashboardStats(ctx context.Context, userID uuid.UUID, plan models.PlanTier) (*DashboardStatsResponse, error) {
	aggStats, err := s.repo.GetStats(ctx, userID)
	if err != nil {
		return nil, err
	}

	successRate := 0.0
	if aggStats.TotalSubmissions > 0 {
		successRate = (float64(aggStats.SuccessfulRuns) / float64(aggStats.TotalSubmissions)) * 100.0
	}

	now := time.Now().UTC()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthlyRunsUsed, _ := s.repo.GetMonthlyRuns(ctx, userID, startOfMonth)
	monthlyRunsUsed = s.mergeRedisQuotaCounter(ctx, userID, now, monthlyRunsUsed)

	return &DashboardStatsResponse{
		TotalSubmissions: aggStats.TotalSubmissions,
		SuccessfulRuns:   aggStats.SuccessfulRuns,
		FailedRuns:       aggStats.FailedRuns,
		TimeoutRuns:      aggStats.TimeoutRuns,
		SuccessRate:      successRate,
		AvgTimeMs:        aggStats.AvgTimeMs,
		TotalComputeSec:  float64(aggStats.TotalTimeMs) / 1000.0,
		MonthlyRunsUsed:  monthlyRunsUsed,
		MonthlyRunsLimit: planMonthlyLimit(plan),
	}, nil
}

func (s *StatsService) mergeRedisQuotaCounter(ctx context.Context, userID uuid.UUID, now time.Time, dbRuns int64) int64 {
	if s.rdb == nil {
		return dbRuns
	}
	ym := fmt.Sprintf("%04d-%02d", now.Year(), now.Month())
	quotaKey := cache.Key("user", "quota", userID.String(), ym, "runs")
	count, err := s.rdb.Get(ctx, quotaKey).Int64()
	if err == nil && count > dbRuns {
		return count
	}
	return dbRuns
}

func mapDailyPoints(rows []submissions.DailyStatRow) []DailyStatPoint {
	points := make([]DailyStatPoint, len(rows))
	for i, row := range rows {
		parsedDate, _ := time.Parse("2006-01-02", row.DateStr)
		points[i] = DailyStatPoint{
			Date:       parsedDate,
			TotalRuns:  row.Total,
			FailedRuns: row.Failed,
			AvgTimeMs:  row.AvgTime,
		}
	}
	return points
}

func planMonthlyLimit(plan models.PlanTier) int64 {
	switch plan {
	case models.PlanPro:
		return 50000
	case models.PlanEnterprise:
		return 1000000
	default:
		return 1000
	}
}
