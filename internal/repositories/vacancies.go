package repositories

import (
	"context"
	"errors"
	"github.com/maxaizer/hh-parser/internal/entities"
	"gorm.io/gorm"
	"time"
)

type Vacancies struct {
	db *gorm.DB
}

func NewVacanciesRepository(db *gorm.DB) *Vacancies {
	return &Vacancies{db: db}
}

func (v *Vacancies) IsSentToUser(ctx context.Context, userID int64, descriptionHash []byte) (bool, error) {
	var notified entities.NotifiedVacancy
	err := v.db.WithContext(ctx).
		Where("user_id = ? AND description_hash = ?", userID, descriptionHash).
		First(&notified).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}

	err = v.db.WithContext(ctx).
		Model(&entities.NotifiedVacancy{}).
		Where("id = ?", notified.ID).
		Update("last_checked_at", time.Now()).Error
	return true, err
}

func (v *Vacancies) RecordAsSentToUser(ctx context.Context, userID int64, descriptionHash []byte) error {
	return v.db.WithContext(ctx).Create(&entities.NotifiedVacancy{
		UserID:          userID,
		DescriptionHash: descriptionHash,
		LastCheckedAt:   time.Now(),
	}).Error
}

func (v *Vacancies) RemoveOldVacancies(ctx context.Context, expirationTime time.Time) (int64, error) {
	res := v.db.WithContext(ctx).Delete(&entities.NotifiedVacancy{}, "last_checked_at < ?", expirationTime)
	return res.RowsAffected, res.Error
}

func (v *Vacancies) AddFailedToAnalyse(ctx context.Context, searchID int, vacancyID string, error string) error {
	return v.db.WithContext(ctx).Exec(`
        INSERT INTO failed_vacancies (search_id, vacancy_id, error, created_at, updated_at) 
        VALUES (?, ?, ?, ?, ?) 
        ON CONFLICT(search_id, vacancy_id) 
        DO UPDATE SET 
                      attempts = failed_vacancies.attempts + 1,
        			  updated_at = CURRENT_TIMESTAMP;
    `, searchID, vacancyID, error, time.Now().UTC(), time.Now().UTC()).Error
}

func (v *Vacancies) RemoveFailedToAnalyse(ctx context.Context, maxAttempts int, minUpdateTime time.Time) (int64, error) {
	res := v.db.WithContext(ctx).Delete(&entities.FailedVacancy{}, "attempts > ? OR updated_at < ?",
		maxAttempts, minUpdateTime.UTC())
	return res.RowsAffected, res.Error
}

func (v *Vacancies) GetFailedToAnalyse(ctx context.Context) ([]entities.FailedVacancy, error) {
	var vacancies []entities.FailedVacancy
	err := v.db.WithContext(ctx).Find(&vacancies).Error
	return vacancies, err
}
