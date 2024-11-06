package repositories

import (
	"context"
	"errors"
	"github.com/maxaizer/hh-parser/internal/entities"
	"gorm.io/gorm"
)

type Regions struct {
	db *gorm.DB
}

func NewRegionsRepository(db *gorm.DB) *Regions {
	return &Regions{db: db}
}

func (repo *Regions) GetIdByName(ctx context.Context, name string) (string, error) {

	var region entities.Region
	name = entities.NormalizeRegionName(name)
	if err := repo.db.WithContext(ctx).First(&region, "normalized_name = ?", name).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}
	return region.ID, nil
}
