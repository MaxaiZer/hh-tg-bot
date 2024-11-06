package repositories

import (
	"context"
	"github.com/maxaizer/hh-parser/internal/clients/hh"
	"github.com/maxaizer/hh-parser/internal/entities"
	"gorm.io/gorm"
)

type Searches struct {
	db *gorm.DB
}

func NewSearchRepository(db *gorm.DB) *Searches {
	return &Searches{db: db}
}

func (repo *Searches) Add(ctx context.Context, jobSearch entities.JobSearch) error {
	return repo.db.WithContext(ctx).Create(&jobSearch).Error
}

func (repo *Searches) GetByUser(ctx context.Context, userID int64) ([]entities.JobSearch, error) {

	var jobSearches []entities.JobSearch
	if err := repo.db.WithContext(ctx).Find(&jobSearches, "user_id = ?", userID).Error; err != nil {
		return nil, err
	}
	return jobSearches, nil
}

func (repo *Searches) GetCountByUser(ctx context.Context, userID int64) (int64, error) {

	var count int64
	if err := repo.db.WithContext(ctx).Model(&entities.JobSearch{}).Where("user_id = ?", userID).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (repo *Searches) Update(ctx context.Context, jobSearch entities.JobSearch) error {
	return repo.db.WithContext(ctx).Model(&entities.JobSearch{}).Where("id = ?", jobSearch.ID).Updates(jobSearch).Error
}

func (repo *Searches) UpdateLastCheckedVacancy(ctx context.Context, id int, vacancy hh.VacancyPreview) error {
	return repo.db.WithContext(ctx).Model(&entities.JobSearch{}).Where("id = ?", id).
		Updates(map[string]any{
			"last_checked_vacancy_time": vacancy.PublishedAt.Time,
			//		"last_checked_vacancy_id":   vacancy.ID,
		}).Error
}

func (repo *Searches) Get(ctx context.Context, pageSize int, pageNum int) ([]entities.JobSearch, error) {

	var jobSearches []entities.JobSearch
	if err := repo.db.WithContext(ctx).
		Limit(pageSize).
		Offset(pageSize * (pageNum - 1)).
		Find(&jobSearches).Error; err != nil {
		return nil, err
	}
	return jobSearches, nil
}

func (repo *Searches) Remove(ctx context.Context, jobSearchID int) error {
	err := repo.db.WithContext(ctx).Delete(&entities.JobSearch{ID: jobSearchID}).Error
	return err
}
