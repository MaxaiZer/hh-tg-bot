package repositories

import (
	"fmt"
	"github.com/glebarez/sqlite"
	"github.com/maxaizer/hh-parser/internal/clients/hh"
	"github.com/maxaizer/hh-parser/internal/domain/models"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DbContext struct {
	DB *gorm.DB
}

func NewDbContext(connectionString string) (*DbContext, error) {
	db, err := gorm.Open(sqlite.Open(connectionString), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Error),
	})
	if err != nil {
		return nil, err
	}

	return &DbContext{DB: db}, nil
}

func (c *DbContext) Migrate() error {
	err := c.DB.AutoMigrate(models.Region{})
	if err != nil {
		return fmt.Errorf("failed to migrate Region entity: %w", err)
	}

	err = c.DB.AutoMigrate(models.JobSearch{})
	if err != nil {
		return fmt.Errorf("failed to migrate JobSearch entity: %w", err)
	}

	err = c.DB.AutoMigrate(models.NotifiedVacancy{})
	if err != nil {
		return fmt.Errorf("failed to migrate NotifiedVacancy entity: %w", err)
	}

	err = c.DB.AutoMigrate(models.FailedVacancy{})
	if err != nil {
		return fmt.Errorf("failed to migrate FailedVacancy entity: %w", err)
	}

	err = c.DB.AutoMigrate(models.ArbitraryData{})
	if err != nil {
		return fmt.Errorf("failed to migrate ArbitraryData entity: %w", err)
	}

	var regionsCount int64
	if err = c.DB.Model(models.Region{}).Count(&regionsCount).Error; err != nil {
		return fmt.Errorf("failed to count regions: %w", err)
	}

	if regionsCount == 0 {
		if err = c.PopulateRegions(); err != nil {
			return fmt.Errorf("failed to populate regions: %w", err)
		}
	}

	if err = c.DB.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_user_vacancy_id ON notified_vacancies (user_id, vacancy_id); " +
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_user_vacancy_description ON notified_vacancies (user_id, description_hash);").
		Error; err != nil {
		return fmt.Errorf("failed to create vacancy index: %w", err)
	}

	return nil
}

func (c *DbContext) PopulateRegions() error {
	client := hh.NewClient()
	areas, err := client.GetAreas()
	if err != nil {
		return fmt.Errorf("failed to get areas from client: %w", err)
	}

	var regions []models.Region

	for _, area := range areas {
		region := models.NewRegion(area.ID, area.Name)
		regions = append(regions, region)
	}

	if err = c.DB.Create(regions).Error; err != nil {
		return fmt.Errorf("failed to create regions in the database: %w", err)
	}
	return nil
}

func (c *DbContext) Close() error {
	db, err := c.DB.DB()
	if err != nil {
		return err
	}

	return db.Close()
}
