package entities

import "time"

type NotifiedVacancy struct {
	ID            int
	UserID        int64
	VacancyID     string
	LastCheckedAt time.Time
	CreatedAt     time.Time
}

type FailedVacancy struct {
	SearchID  int    `gorm:"primaryKey"`
	VacancyID string `gorm:"primaryKey"`
	Error     string
	Attempts  int `gorm:"default:1"`
	CreatedAt time.Time
	UpdatedAt *time.Time
}
