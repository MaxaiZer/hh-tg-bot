package repositories

import (
	"context"
	"github.com/maxaizer/hh-parser/internal/domain/models"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type Data struct {
	db *gorm.DB
}

func NewDataRepository(db *gorm.DB) *Data {
	return &Data{db: db}
}

func (repo *Data) Save(ctx context.Context, id string, data []byte) error {
	return repo.db.WithContext(ctx).Save(models.ArbitraryData{
		ID:    id,
		Value: data,
	}).Error
}

func (repo *Data) Load(ctx context.Context, id string) ([]byte, error) {
	data := &models.ArbitraryData{}
	err := repo.db.WithContext(ctx).First(data, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return data.Value, nil
}

func (repo *Data) LoadAndRemove(ctx context.Context, id string) ([]byte, error) {
	data, err := repo.Load(ctx, id)
	if data == nil || err != nil {
		return nil, err
	}
	err = repo.Remove(ctx, id)
	return data, err
}

func (repo *Data) Remove(ctx context.Context, id string) error {
	return repo.db.WithContext(ctx).Delete(&models.ArbitraryData{}, "id = ?", id).Error
}
