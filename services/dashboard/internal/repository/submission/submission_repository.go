package submission

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"code-runner/shared/dto"
	"code-runner/shared/models"
)

var (
	ErrSubmissionNotFound = errors.New("submission not found")
)

type StatsResult struct {
	TotalSubmissions int64
	SuccessfulRuns   int64
	FailedRuns       int64
	TimeoutRuns      int64
	AvgTimeMs        float64
	TotalTimeMs      int64
}

type Repository interface {
	Create(ctx context.Context, sub *models.Submission) error
	FindByID(ctx context.Context, userID, subID uuid.UUID) (*models.Submission, error)
	List(ctx context.Context, userID uuid.UUID, status, language string, limit, offset int) ([]models.Submission, int64, error)
	GetStats(ctx context.Context, userID uuid.UUID) (*StatsResult, error)
	GetMonthlyRuns(ctx context.Context, userID uuid.UUID, since time.Time) (int64, error)
	GetDailyStats(ctx context.Context, userID uuid.UUID, days int) ([]dto.DailyStatPoint, error)
}

type GormRepository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

func (r *GormRepository) Create(ctx context.Context, sub *models.Submission) error {
	return r.db.WithContext(ctx).Create(sub).Error
}

func (r *GormRepository) FindByID(ctx context.Context, userID, subID uuid.UUID) (*models.Submission, error) {
	var sub models.Submission
	if err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", subID, userID).First(&sub).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSubmissionNotFound
		}
		return nil, err
	}
	return &sub, nil
}

func (r *GormRepository) List(ctx context.Context, userID uuid.UUID, status, language string, limit, offset int) ([]models.Submission, int64, error) {
	tx := r.db.WithContext(ctx).Model(&models.Submission{}).Where("user_id = ?", userID)
	if status != "" {
		tx = tx.Where("status = ?", status)
	}
	if language != "" {
		tx = tx.Where("language = ?", language)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var subs []models.Submission
	if err := tx.Order("created_at desc").Limit(limit).Offset(offset).Find(&subs).Error; err != nil {
		return nil, 0, err
	}

	return subs, total, nil
}

func (r *GormRepository) GetStats(ctx context.Context, userID uuid.UUID) (*StatsResult, error) {
	var stats StatsResult

	r.db.WithContext(ctx).Model(&models.Submission{}).Where("user_id = ?", userID).Count(&stats.TotalSubmissions)
	r.db.WithContext(ctx).Model(&models.Submission{}).Where("user_id = ? AND status = ?", userID, models.SubmissionCompleted).Count(&stats.SuccessfulRuns)
	r.db.WithContext(ctx).Model(&models.Submission{}).Where("user_id = ? AND status = ?", userID, models.SubmissionFailed).Count(&stats.FailedRuns)
	r.db.WithContext(ctx).Model(&models.Submission{}).Where("user_id = ? AND status = ?", userID, models.SubmissionTimedOut).Count(&stats.TimeoutRuns)

	type Agg struct {
		AvgTime   float64
		TotalTime int64
	}
	var agg Agg
	r.db.WithContext(ctx).Model(&models.Submission{}).
		Select("COALESCE(AVG(time_taken_ms), 0) as avg_time, COALESCE(SUM(time_taken_ms), 0) as total_time").
		Where("user_id = ?", userID).
		Scan(&agg)

	stats.AvgTimeMs = agg.AvgTime
	stats.TotalTimeMs = agg.TotalTime
	return &stats, nil
}

func (r *GormRepository) GetMonthlyRuns(ctx context.Context, userID uuid.UUID, since time.Time) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Submission{}).
		Where("user_id = ? AND created_at >= ?", userID, since).
		Count(&count).Error
	return count, err
}

func (r *GormRepository) GetDailyStats(ctx context.Context, userID uuid.UUID, days int) ([]dto.DailyStatPoint, error) {
	if days <= 0 || days > 90 {
		days = 30
	}

	since := time.Now().UTC().AddDate(0, 0, -days)
	type DailyRow struct {
		DateStr  string  `gorm:"column:d"`
		Total    int64   `gorm:"column:total"`
		Failed   int64   `gorm:"column:failed"`
		AvgTime  float64 `gorm:"column:avg_time"`
	}

	var rows []DailyRow
	query := `
		SELECT 
			TO_CHAR(created_at, 'YYYY-MM-DD') as d,
			COUNT(*) as total,
			COUNT(CASE WHEN status != 'COMPLETED' THEN 1 END) as failed,
			COALESCE(AVG(time_taken_ms), 0) as avg_time
		FROM submissions
		WHERE user_id = ? AND created_at >= ?
		GROUP BY TO_CHAR(created_at, 'YYYY-MM-DD')
		ORDER BY d ASC
	`
	if err := r.db.WithContext(ctx).Raw(query, userID, since).Scan(&rows).Error; err != nil {
		return nil, err
	}

	points := make([]dto.DailyStatPoint, len(rows))
	for i, row := range rows {
		parsedDate, _ := time.Parse("2006-01-02", row.DateStr)
		points[i] = dto.DailyStatPoint{
			Date:       parsedDate,
			TotalRuns:  row.Total,
			FailedRuns: row.Failed,
			AvgTimeMs:  row.AvgTime,
		}
	}
	return points, nil
}
