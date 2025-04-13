package repositories

import (
	"context"
	"github.com/maxaizer/hh-parser/internal/domain/models"
	"gorm.io/gorm"
)

type Searches struct {
	db *gorm.DB
}

func NewSearchRepository(db *gorm.DB) *Searches {
	return &Searches{db: db}
}

func (repo *Searches) Add(ctx context.Context, jobSearch models.JobSearch) error {
	return repo.db.WithContext(ctx).Create(&jobSearch).Error
}

func (repo *Searches) GetByUser(ctx context.Context, userID int64) ([]models.JobSearch, error) {

	var jobSearches []models.JobSearch
	if err := repo.db.WithContext(ctx).Find(&jobSearches, "user_id = ?", userID).Error; err != nil {
		return nil, err
	}
	return jobSearches, nil
}

func (repo *Searches) GetByID(ctx context.Context, ID int64) (*models.JobSearch, error) {

	var jobSearch models.JobSearch
	if err := repo.db.WithContext(ctx).Find(&jobSearch, "id = ?", ID).Error; err != nil {
		return nil, err
	}
	return &jobSearch, nil
}

func (repo *Searches) GetCountByUser(ctx context.Context, userID int64) (int64, error) {

	var count int64
	if err := repo.db.WithContext(ctx).Model(&models.JobSearch{}).Where("user_id = ?", userID).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (repo *Searches) Update(ctx context.Context, jobSearch models.JobSearch) error {
	return repo.db.WithContext(ctx).Model(&models.JobSearch{}).Where("id = ?", jobSearch.ID).Updates(jobSearch).Error
}

func (repo *Searches) UpdateLastCheckedVacancy(ctx context.Context, id int, vacancy models.Vacancy) error {
	return repo.db.WithContext(ctx).Model(&models.JobSearch{}).Where("id = ?", id).
		Updates(map[string]any{
			"last_checked_vacancy_time": vacancy.PublishedAt.UTC(),
		}).Error
}

func (repo *Searches) Get(ctx context.Context, limit int, offset int) ([]models.JobSearch, error) {

	var jobSearches []models.JobSearch
	if err := repo.db.WithContext(ctx).
		Limit(limit).
		Offset(offset).
		Find(&jobSearches).Error; err != nil {
		return nil, err
	}
	return jobSearches, nil
}

func (repo *Searches) Remove(ctx context.Context, jobSearchID int) error {
	err := repo.db.WithContext(ctx).Delete(&models.JobSearch{ID: jobSearchID}).Error
	return err
}
