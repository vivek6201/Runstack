package stats

import "time"

type DashboardStatsResponse struct {
	TotalSubmissions int64   `json:"total_submissions"`
	SuccessfulRuns   int64   `json:"successful_runs"`
	FailedRuns       int64   `json:"failed_runs"`
	TimeoutRuns      int64   `json:"timeout_runs"`
	SuccessRate      float64 `json:"success_rate"`
	AvgTimeMs        float64 `json:"avg_time_ms"`
	TotalComputeSec  float64 `json:"total_compute_sec"`
	MonthlyRunsUsed  int64   `json:"monthly_runs_used"`
	MonthlyRunsLimit int64   `json:"monthly_runs_limit"`
}

type DailyStatPoint struct {
	Date       time.Time `json:"date"`
	TotalRuns  int64     `json:"total_runs"`
	FailedRuns int64     `json:"failed_runs"`
	AvgTimeMs  float64   `json:"avg_time_ms"`
}
