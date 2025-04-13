package repositories

import (
	"context"
	"errors"
	errs "github.com/maxaizer/hh-parser/internal/domain/errors"
	"github.com/maxaizer/hh-parser/internal/domain/models"
	"gorm.io/gorm"
	"strings"
	"time"
)

type Vacancies struct {
	db *gorm.DB
}

func NewVacanciesRepository(db *gorm.DB) *Vacancies {
	return &Vacancies{db: db}
}

func (v *Vacancies) IsSentToUser(ctx context.Context, vacancy models.NotifiedVacancyID) (bool, error) {
	var notified models.NotifiedVacancy
	err := v.db.WithContext(ctx).
		Where("user_id = ? AND (vacancy_id = ? OR description_hash = ?)",
			vacancy.UserID, vacancy.VacancyID, vacancy.DescriptionHash).
		First(&notified).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}

	err = v.db.WithContext(ctx).
		Model(&models.NotifiedVacancy{}).
		Where("id = ?", notified.ID).
		Update("last_checked_at", time.Now().UTC()).Error
	return true, err
}

func (v *Vacancies) RecordAsSentToUser(ctx context.Context, vacancy models.NotifiedVacancyID) error {

	err := v.db.WithContext(ctx).Create(&models.NotifiedVacancy{
		UserID:          vacancy.UserID,
		VacancyID:       vacancy.VacancyID,
		DescriptionHash: vacancy.DescriptionHash,
		LastCheckedAt:   time.Now().UTC(),
	}).Error
	if err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed") {
		return errs.VacancyAlreadySentToUser
	}

	return err
}

func (v *Vacancies) RemoveOldVacancies(ctx context.Context, expirationTime time.Time) (int64, error) {
	res := v.db.WithContext(ctx).Delete(&models.NotifiedVacancy{}, "last_checked_at < ?", expirationTime.UTC())
	return res.RowsAffected, res.Error
}

func (v *Vacancies) AddFailedToAnalyze(ctx context.Context, searchID int, vacancyID string, error string) error {
	return v.db.WithContext(ctx).Exec(`
        INSERT INTO failed_vacancies (search_id, vacancy_id, error, created_at, updated_at) 
        VALUES (?, ?, ?, ?, ?) 
        ON CONFLICT(search_id, vacancy_id) 
        DO UPDATE SET 
                      attempts = failed_vacancies.attempts + 1,
        			  updated_at = STRFTIME('%Y-%m-%d %H:%M:%f', 'NOW');
    `, searchID, vacancyID, error, time.Now().UTC(), time.Now().UTC()).Error
}

func (v *Vacancies) RemoveFailedToAnalyze(ctx context.Context, maxAttempts int, minUpdateTime time.Time) (int64, error) {
	res := v.db.WithContext(ctx).Delete(&models.FailedVacancy{}, "attempts > ? OR updated_at < ?",
		maxAttempts, minUpdateTime.UTC())
	return res.RowsAffected, res.Error
}

func (v *Vacancies) GetFailedToAnalyze(ctx context.Context) ([]models.FailedVacancy, error) {
	var vacancies []models.FailedVacancy
	err := v.db.WithContext(ctx).Find(&vacancies).Error
	return vacancies, err
}
