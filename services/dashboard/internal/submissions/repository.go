package submissions

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"code-runner/shared/models"
)

var ErrSubmissionNotFound = errors.New("submission not found")

type ISubmissionsRepository interface {
	Create(ctx context.Context, sub *models.Submission) error
	FindByID(ctx context.Context, userID, subID uuid.UUID) (*models.Submission, error)
	List(ctx context.Context, userID uuid.UUID, status, language string, limit, offset int) ([]models.Submission, int64, error)
	GetStats(ctx context.Context, userID uuid.UUID) (*StatsAggregate, error)
	GetMonthlyRuns(ctx context.Context, userID uuid.UUID, since time.Time) (int64, error)
	GetDailyRows(ctx context.Context, userID uuid.UUID, days int) ([]DailyStatRow, error)
}

type SubmissionsRepository struct {
	db *gorm.DB
}

func NewSubmissionsRepository(db *gorm.DB) ISubmissionsRepository {
	return &SubmissionsRepository{db: db}
}

func (r *SubmissionsRepository) Create(ctx context.Context, sub *models.Submission) error {
	return r.db.WithContext(ctx).Create(sub).Error
}

func (r *SubmissionsRepository) FindByID(ctx context.Context, userID, subID uuid.UUID) (*models.Submission, error) {
	var sub models.Submission
	if err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", subID, userID).First(&sub).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSubmissionNotFound
		}
		return nil, err
	}
	return &sub, nil
}

func (r *SubmissionsRepository) List(ctx context.Context, userID uuid.UUID, status, language string, limit, offset int) ([]models.Submission, int64, error) {
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

func (r *SubmissionsRepository) GetStats(ctx context.Context, userID uuid.UUID) (*StatsAggregate, error) {
	var stats StatsAggregate

	r.db.WithContext(ctx).Model(&models.Submission{}).Where("user_id = ?", userID).Count(&stats.TotalSubmissions)
	r.db.WithContext(ctx).Model(&models.Submission{}).Where("user_id = ? AND status = ?", userID, models.SubmissionCompleted).Count(&stats.SuccessfulRuns)
	r.db.WithContext(ctx).Model(&models.Submission{}).Where("user_id = ? AND status = ?", userID, models.SubmissionFailed).Count(&stats.FailedRuns)
	r.db.WithContext(ctx).Model(&models.Submission{}).Where("user_id = ? AND status = ?", userID, models.SubmissionTimedOut).Count(&stats.TimeoutRuns)

	type aggRow struct {
		AvgTime   float64
		TotalTime int64
	}
	var agg aggRow
	r.db.WithContext(ctx).Model(&models.Submission{}).
		Select("COALESCE(AVG(time_taken_ms), 0) as avg_time, COALESCE(SUM(time_taken_ms), 0) as total_time").
		Where("user_id = ?", userID).
		Scan(&agg)

	stats.AvgTimeMs = agg.AvgTime
	stats.TotalTimeMs = agg.TotalTime
	return &stats, nil
}

func (r *SubmissionsRepository) GetMonthlyRuns(ctx context.Context, userID uuid.UUID, since time.Time) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Submission{}).
		Where("user_id = ? AND created_at >= ?", userID, since).
		Count(&count).Error
	return count, err
}

func (r *SubmissionsRepository) GetDailyRows(ctx context.Context, userID uuid.UUID, days int) ([]DailyStatRow, error) {
	if days <= 0 || days > 90 {
		days = 30
	}

	since := time.Now().UTC().AddDate(0, 0, -days)
	var rows []DailyStatRow
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
	return rows, nil
}
