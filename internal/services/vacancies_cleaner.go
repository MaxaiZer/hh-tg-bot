package services

import (
	"context"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
	"time"
)

type VacancyCleanupRepository interface {
	RemoveOldVacancies(ctx context.Context, expirationTime time.Time) (int64, error)
}

type VacanciesCleaner struct {
	vacancies            VacancyCleanupRepository
	cron                 *cron.Cron
	expirationTimeInDays int
}

func NewVacanciesCleaner(vacancies VacancyCleanupRepository) (*VacanciesCleaner, error) {
	vc := &VacanciesCleaner{
		vacancies:            vacancies,
		cron:                 cron.New(),
		expirationTimeInDays: 14,
	}

	_, err := vc.cron.AddFunc("0 0 * * *", vc.cleanOldVacancies)
	if err != nil {
		return nil, err
	}

	vc.cron.Start()
	return vc, nil
}

func (vc *VacanciesCleaner) Stop() {
	vc.cron.Stop()
}

func (vc *VacanciesCleaner) cleanOldVacancies() {
	expirationTime := time.Now().Add(-time.Duration(vc.expirationTimeInDays) * 24 * time.Hour)
	rowsAffected, err := vc.vacancies.RemoveOldVacancies(context.Background(), expirationTime)
	if err != nil {
		log.Errorf("Failed to clean old vacancies: %v", err)
	} else {
		log.Infof("Old vacancies was cleaned at %v, affected rows: %v", time.Now(), rowsAffected)
	}
}
