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

func (v Vacancies) IsSentToUser(ctx context.Context, userID int64, vacancyID string) (bool, error) {
	var vacancy entities.NotifiedVacancy
	err := v.db.WithContext(ctx).
		Where("user_id = ? AND vacancy_id = ?", userID, vacancyID).
		First(&vacancy).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}

	err = v.db.WithContext(ctx).
		Model(&entities.NotifiedVacancy{}).
		Where("id = ?", vacancy.ID).
		Update("last_checked_at", time.Now()).Error
	return true, err
}

func (v Vacancies) RecordAsSentToUser(ctx context.Context, userID int64, vacancyID string) error {
	return v.db.WithContext(ctx).Create(&entities.NotifiedVacancy{
		UserID:        userID,
		VacancyID:     vacancyID,
		LastCheckedAt: time.Now(),
	}).Error
}

func (v Vacancies) RemoveOldVacancies(ctx context.Context, expirationTime time.Time) (int64, error) {
	res := v.db.WithContext(ctx).Delete(&entities.NotifiedVacancy{}, "last_checked_at < ?", expirationTime)
	return res.RowsAffected, res.Error
}
